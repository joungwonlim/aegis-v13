package naver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Client handles communication with Naver Finance
// ⭐ SSOT: Naver Finance API 호출은 이 클라이언트에서만
type Client struct {
	httpClient *httputil.Client
	logger     *logger.Logger
	baseURL    string
}

// NewClient creates a new Naver Finance client
func NewClient(httpClient *httputil.Client, log *logger.Logger) *Client {
	return &Client{
		httpClient: httpClient,
		logger:     log,
		baseURL:    "https://finance.naver.com",
	}
}

// fetchHTML fetches HTML from Naver Finance
func (c *Client) fetchHTML(ctx context.Context, path string, params url.Values) (string, error) {
	fullURL := fmt.Sprintf("%s%s", c.baseURL, path)
	if len(params) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, params.Encode())
	}

	resp, err := c.httpClient.Get(ctx, fullURL)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// PriceData represents daily price data
type PriceData struct {
	StockCode    string
	TradeDate    time.Time
	OpenPrice    int64
	HighPrice    int64
	LowPrice     int64
	ClosePrice   int64
	Volume       int64
	TradingValue int64
}

// InvestorFlowData represents investor trading flow
type InvestorFlowData struct {
	StockCode     string
	TradeDate     time.Time
	ForeignNet    int64 // 외국인 순매수
	InstitutionNet int64 // 기관 순매수
	IndividualNet int64 // 개인 순매수
	FinancialNet  int64 // 금융투자
	InsuranceNet  int64 // 보험
	TrustNet      int64 // 투신
	PensionNet    int64 // 연기금
}

// MarketCapData represents market capitalization
type MarketCapData struct {
	StockCode string
	TradeDate time.Time
	MarketCap int64
}
