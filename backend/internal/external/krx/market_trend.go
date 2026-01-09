package krx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FetchMarketTrend fetches market-wide investor trading data from Naver API (KRX data)
// ⭐ SSOT: KRX 시장 지표 호출은 이 함수에서만
func (c *Client) FetchMarketTrend(ctx context.Context, indexName string) (*MarketTrendData, error) {
	// Map index name to code
	indexCode := indexName
	if indexName == "KOSDAQ" {
		indexCode = "KOSDAQ"
	} else if indexName == "KOSPI" {
		indexCode = "KOSPI"
	}

	url := fmt.Sprintf("%s/api/index/%s/trend", c.baseURL, indexCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers to mimic browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://finance.naver.com/")

	resp, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Parse response
	var trend MarketTrendResponse
	if err := json.Unmarshal(body, &trend); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if trend.Bizdate == "" {
		return nil, nil
	}

	// Parse trade date
	tradeDate, err := time.Parse("20060102", trend.Bizdate)
	if err != nil {
		return nil, fmt.Errorf("parse trade date: %w", err)
	}

	// Parse net buy volumes
	foreignNet := parseNetBuyVolume(trend.ForeignValue)
	instNet := parseNetBuyVolume(trend.InstitutionValue)
	personalNet := parseNetBuyVolume(trend.PersonalValue)

	c.logger.WithFields(map[string]interface{}{
		"index":        indexName,
		"trade_date":   tradeDate.Format("2006-01-02"),
		"foreign_net":  foreignNet,
		"inst_net":     instNet,
		"personal_net": personalNet,
	}).Debug("Fetched market trend")

	return &MarketTrendData{
		TradeDate:      tradeDate,
		ForeignNet:     foreignNet,
		InstitutionNet: instNet,
		IndividualNet:  personalNet,
	}, nil
}

// parseNetBuyVolume parses net buy volume string like "+1,459,781" or "-1,240,182" to float64
func parseNetBuyVolume(s string) float64 {
	if s == "" {
		return 0
	}

	// Remove commas and spaces
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	// Handle sign
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}

	// Parse to float
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	if negative {
		return -val
	}
	return val
}
