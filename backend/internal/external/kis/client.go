package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Client handles communication with KIS (한국투자증권) API
// ⭐ SSOT: KIS API 호출은 이 클라이언트에서만
type Client struct {
	httpClient *httputil.Client
	logger     *logger.Logger
	cfg        config.KISConfig

	// Token management
	accessToken string
	tokenExpiry time.Time
	tokenMu     sync.RWMutex
}

// NewClient creates a new KIS API client
func NewClient(cfg config.KISConfig, httpClient *httputil.Client, log *logger.Logger) *Client {
	return &Client{
		httpClient: httpClient,
		logger:     log,
		cfg:        cfg,
	}
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// getToken gets a valid access token, refreshing if necessary
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	// Need to refresh token
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// Request new token
	url := fmt.Sprintf("%s/oauth2/tokenP", c.cfg.BaseURL)
	body := fmt.Sprintf(`{"grant_type":"client_credentials","appkey":"%s","appsecret":"%s"}`,
		c.cfg.AppKey, c.cfg.AppSecret)

	resp, err := c.httpClient.Post(ctx, url, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second) // 1분 여유

	c.logger.WithFields(map[string]interface{}{
		"expires_in": tokenResp.ExpiresIn,
	}).Info("KIS access token refreshed")

	return c.accessToken, nil
}

// request makes an authenticated request to KIS API
func (c *Client) request(ctx context.Context, method, path string, trID string, body io.Reader) (*http.Response, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.cfg.BaseURL, path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.cfg.AppKey)
	req.Header.Set("appsecret", c.cfg.AppSecret)
	req.Header.Set("tr_id", trID)

	// Use underlying http client directly for custom headers
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// StockPrice represents a stock price from KIS
type StockPrice struct {
	StockCode  string    `json:"stck_shrn_iscd"`
	TradeDate  string    `json:"stck_bsop_date"`
	OpenPrice  float64   `json:"stck_oprc,string"`
	HighPrice  float64   `json:"stck_hgpr,string"`
	LowPrice   float64   `json:"stck_lwpr,string"`
	ClosePrice float64   `json:"stck_clpr,string"`
	Volume     int64     `json:"acml_vol,string"`
	TradingVal int64     `json:"acml_tr_pbmn,string"`
	FetchedAt  time.Time `json:"-"`
}

// GetDailyPrice gets daily price for a stock
func (c *Client) GetDailyPrice(ctx context.Context, stockCode string, date time.Time) (*StockPrice, error) {
	path := "/uapi/domestic-stock/v1/quotations/inquire-daily-price"
	trID := "FHKST01010400" // 국내주식 일별 시세

	params := fmt.Sprintf("?fid_cond_mrkt_div_code=J&fid_input_iscd=%s&fid_period_div_code=D&fid_org_adj_prc=0",
		stockCode)

	resp, err := c.request(ctx, http.MethodGet, path+params, trID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Output []StockPrice `json:"output"`
		RtCd   string       `json:"rt_cd"`
		MsgCd  string       `json:"msg_cd"`
		Msg1   string       `json:"msg1"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.RtCd != "0" {
		return nil, fmt.Errorf("API error: %s - %s", result.MsgCd, result.Msg1)
	}

	if len(result.Output) == 0 {
		return nil, fmt.Errorf("no price data for %s", stockCode)
	}

	price := &result.Output[0]
	price.StockCode = stockCode
	price.FetchedAt = time.Now()

	return price, nil
}

// GetCurrentPrice gets real-time current price for a stock
func (c *Client) GetCurrentPrice(ctx context.Context, stockCode string) (*StockPrice, error) {
	path := "/uapi/domestic-stock/v1/quotations/inquire-price"
	trID := "FHKST01010100" // 국내주식 현재가

	params := fmt.Sprintf("?fid_cond_mrkt_div_code=J&fid_input_iscd=%s", stockCode)

	resp, err := c.request(ctx, http.MethodGet, path+params, trID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Output struct {
			StockCode  string `json:"stck_shrn_iscd"`
			OpenPrice  string `json:"stck_oprc"`
			HighPrice  string `json:"stck_hgpr"`
			LowPrice   string `json:"stck_lwpr"`
			ClosePrice string `json:"stck_prpr"`
			Volume     string `json:"acml_vol"`
			TradingVal string `json:"acml_tr_pbmn"`
		} `json:"output"`
		RtCd  string `json:"rt_cd"`
		MsgCd string `json:"msg_cd"`
		Msg1  string `json:"msg1"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.RtCd != "0" {
		return nil, fmt.Errorf("API error: %s - %s", result.MsgCd, result.Msg1)
	}

	price := &StockPrice{
		StockCode: stockCode,
		TradeDate: time.Now().Format("20060102"),
		FetchedAt: time.Now(),
	}

	// Parse string values
	fmt.Sscanf(result.Output.OpenPrice, "%f", &price.OpenPrice)
	fmt.Sscanf(result.Output.HighPrice, "%f", &price.HighPrice)
	fmt.Sscanf(result.Output.LowPrice, "%f", &price.LowPrice)
	fmt.Sscanf(result.Output.ClosePrice, "%f", &price.ClosePrice)
	fmt.Sscanf(result.Output.Volume, "%d", &price.Volume)
	fmt.Sscanf(result.Output.TradingVal, "%d", &price.TradingVal)

	return price, nil
}
