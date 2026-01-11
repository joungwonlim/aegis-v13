package krx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MarketCapItem represents a single stock's market cap data from KRX API
type MarketCapItem struct {
	StockCode         string    `json:"ISU_SRT_CD"`       // 종목코드
	StockName         string    `json:"ISU_ABBRV"`        // 종목명
	MarketCap         int64     `json:"MKTCAP"`           // 시가총액
	SharesOutstanding int64     `json:"LIST_SHRS"`        // 상장주식수
	ClosePrice        int64     `json:"TDD_CLSPRC"`       // 종가
	TradeDate         time.Time `json:"-"`                // 거래일 (파싱 후 설정)
}

// krxMarketCapResponse represents KRX API response
type krxMarketCapResponse struct {
	OutBlock1 []krxMarketCapRow `json:"OutBlock_1"`
}

// krxMarketCapRow represents a row in KRX market cap response
type krxMarketCapRow struct {
	ISU_SRT_CD  string `json:"ISU_SRT_CD"`  // 종목코드 (단축)
	ISU_ABBRV   string `json:"ISU_ABBRV"`   // 종목명
	TDD_CLSPRC  string `json:"TDD_CLSPRC"`  // 종가
	MKTCAP      string `json:"MKTCAP"`      // 시가총액
	LIST_SHRS   string `json:"LIST_SHRS"`   // 상장주식수
}

// FetchMarketCaps fetches market cap and shares outstanding from KRX for all stocks
// ⭐ SSOT: KRX 시가총액/상장주식수 조회는 이 함수에서만
func (c *Client) FetchMarketCaps(ctx context.Context, market string) ([]MarketCapItem, error) {
	// KRX API endpoint for market cap
	// pykrx uses: http://data.krx.co.kr/comm/bldAttendant/getJsonData.cmd
	krxURL := "http://data.krx.co.kr/comm/bldAttendant/getJsonData.cmd"

	// Determine market code
	var mktId string
	switch strings.ToUpper(market) {
	case "KOSPI":
		mktId = "STK"
	case "KOSDAQ":
		mktId = "KSQ"
	default:
		return nil, fmt.Errorf("unsupported market: %s", market)
	}

	// Use yesterday's date if today is before market close
	tradeDate := time.Now()
	if tradeDate.Hour() < 16 {
		tradeDate = tradeDate.AddDate(0, 0, -1)
	}
	// Skip weekends
	for tradeDate.Weekday() == time.Saturday || tradeDate.Weekday() == time.Sunday {
		tradeDate = tradeDate.AddDate(0, 0, -1)
	}
	trdDd := tradeDate.Format("20060102")

	// Build form data
	formData := url.Values{
		"bld":         {"dbms/MDC/STAT/standard/MDCSTAT01501"},
		"locale":      {"ko_KR"},
		"mktId":       {mktId},
		"trdDd":       {trdDd},
		"share":       {"1"},
		"money":       {"1"},
		"csvxls_isNo": {"false"},
	}

	c.logger.WithFields(map[string]interface{}{
		"market":     market,
		"trade_date": trdDd,
	}).Info("Fetching market caps from KRX")

	// Create HTTP request with browser-like headers (KRX blocks bot requests)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, krxURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers to mimic browser request (required by KRX)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Origin", "http://data.krx.co.kr")
	req.Header.Set("Referer", "http://data.krx.co.kr/contents/MDC/MDI/mdiLoader/index.cmd?menuId=MDC0201020101")

	// Make request using standard http client (bypass our wrapper for this special case)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KRX API request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"status_code": resp.StatusCode,
		"body_size":   len(body),
	}).Debug("KRX API response received")

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KRX API returned status %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	// Parse response
	var apiResp krxMarketCapResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// Log first 500 chars of response for debugging
		preview := string(body)
		if len(preview) > 500 {
			preview = preview[:500]
		}
		c.logger.WithField("response_preview", preview).Error("Failed to parse KRX response")
		return nil, fmt.Errorf("decode KRX response: %w", err)
	}

	if len(apiResp.OutBlock1) == 0 {
		c.logger.Warn("KRX API returned empty data")
		return nil, nil
	}

	// Convert to MarketCapItem
	result := make([]MarketCapItem, 0, len(apiResp.OutBlock1))
	parsedDate, _ := time.Parse("20060102", trdDd)

	for _, row := range apiResp.OutBlock1 {
		// Parse numeric values (remove commas)
		marketCap := parseKRXNumber(row.MKTCAP)
		shares := parseKRXNumber(row.LIST_SHRS)
		closePrice := parseKRXNumber(row.TDD_CLSPRC)

		// Skip if essential data is missing
		if row.ISU_SRT_CD == "" || shares == 0 {
			continue
		}

		result = append(result, MarketCapItem{
			StockCode:         row.ISU_SRT_CD,
			StockName:         row.ISU_ABBRV,
			MarketCap:         marketCap,
			SharesOutstanding: shares,
			ClosePrice:        closePrice,
			TradeDate:         parsedDate,
		})
	}

	c.logger.WithFields(map[string]interface{}{
		"market": market,
		"count":  len(result),
	}).Info("Fetched market caps from KRX")

	return result, nil
}

// FetchAllMarketCaps fetches market caps for both KOSPI and KOSDAQ
func (c *Client) FetchAllMarketCaps(ctx context.Context) ([]MarketCapItem, error) {
	var allItems []MarketCapItem

	// Fetch KOSPI
	kospiItems, err := c.FetchMarketCaps(ctx, "KOSPI")
	if err != nil {
		return nil, fmt.Errorf("fetch KOSPI market caps: %w", err)
	}
	allItems = append(allItems, kospiItems...)

	// Fetch KOSDAQ
	kosdaqItems, err := c.FetchMarketCaps(ctx, "KOSDAQ")
	if err != nil {
		return nil, fmt.Errorf("fetch KOSDAQ market caps: %w", err)
	}
	allItems = append(allItems, kosdaqItems...)

	c.logger.WithFields(map[string]interface{}{
		"kospi_count":  len(kospiItems),
		"kosdaq_count": len(kosdaqItems),
		"total_count":  len(allItems),
	}).Info("Fetched all market caps from KRX")

	return allItems, nil
}

// parseKRXNumber parses KRX number format (with commas) to int64
func parseKRXNumber(s string) int64 {
	// Remove commas and whitespace
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
