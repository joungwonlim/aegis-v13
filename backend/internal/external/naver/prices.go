package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FetchPrices fetches daily price data for a stock from Naver Finance
// ⭐ SSOT: Naver Finance 가격 API 호출은 이 함수에서만
func (c *Client) FetchPrices(ctx context.Context, stockCode string, from, to time.Time) ([]PriceData, error) {
	fromStr := strings.ReplaceAll(from.Format("2006-01-02"), "-", "")
	toStr := strings.ReplaceAll(to.Format("2006-01-02"), "-", "")

	// Naver Finance Chart API
	fullURL := fmt.Sprintf(
		"https://fchart.stock.naver.com/siseJson.naver?symbol=%s&requestType=1&startTime=%s&endTime=%s&timeframe=day",
		stockCode, fromStr, toStr,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://finance.naver.com/")

	resp, err := c.httpClient.Get(ctx, fullURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	prices, err := c.parsePriceResponse(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"stock_code": stockCode,
		"count":      len(prices),
	}).Debug("Fetched prices")
	return prices, nil
}

// parsePriceResponse parses Naver Finance JSON response
func (c *Client) parsePriceResponse(body string) ([]PriceData, error) {
	body = strings.TrimSpace(body)
	body = strings.ReplaceAll(body, "'", "\"")

	// Try JSON parsing first
	var rawData [][]interface{}
	if err := json.Unmarshal([]byte(body), &rawData); err == nil {
		return c.parsePriceJSON(rawData)
	}

	// Fallback to regex parsing
	return c.parsePriceRegex(body)
}

// parsePriceJSON parses JSON array format
func (c *Client) parsePriceJSON(rawData [][]interface{}) ([]PriceData, error) {
	var prices []PriceData
	for i, row := range rawData {
		if i == 0 || len(row) < 6 {
			continue // Skip header
		}

		dateStr, ok := row[0].(string)
		if !ok {
			continue
		}
		dateStr = strings.Trim(dateStr, "\"")
		if len(dateStr) == 8 {
			dateStr = dateStr[:4] + "-" + dateStr[4:6] + "-" + dateStr[6:8]
		}

		tradeDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		openPrice := toInt64(row[1])
		highPrice := toInt64(row[2])
		lowPrice := toInt64(row[3])
		closePrice := toInt64(row[4])
		volume := toInt64(row[5])

		prices = append(prices, PriceData{
			StockCode:    "", // Will be set by caller
			TradeDate:    tradeDate,
			OpenPrice:    openPrice,
			HighPrice:    highPrice,
			LowPrice:     lowPrice,
			ClosePrice:   closePrice,
			Volume:       volume,
			TradingValue: closePrice * volume,
		})
	}
	return prices, nil
}

// parsePriceRegex parses using regex (fallback)
func (c *Client) parsePriceRegex(body string) ([]PriceData, error) {
	re := regexp.MustCompile(`\["(\d{8})",\s*(\d+),\s*(\d+),\s*(\d+),\s*(\d+),\s*(\d+)\]`)
	matches := re.FindAllStringSubmatch(body, -1)

	var prices []PriceData
	for _, match := range matches {
		if len(match) < 7 {
			continue
		}

		dateStr := match[1][:4] + "-" + match[1][4:6] + "-" + match[1][6:8]
		tradeDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		openPrice, _ := strconv.ParseInt(match[2], 10, 64)
		highPrice, _ := strconv.ParseInt(match[3], 10, 64)
		lowPrice, _ := strconv.ParseInt(match[4], 10, 64)
		closePrice, _ := strconv.ParseInt(match[5], 10, 64)
		volume, _ := strconv.ParseInt(match[6], 10, 64)

		prices = append(prices, PriceData{
			StockCode:    "",
			TradeDate:    tradeDate,
			OpenPrice:    openPrice,
			HighPrice:    highPrice,
			LowPrice:     lowPrice,
			ClosePrice:   closePrice,
			Volume:       volume,
			TradingValue: closePrice * volume,
		})
	}
	return prices, nil
}

// toInt64 converts various types to int64
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	default:
		return 0
	}
}

// FetchMarketCap fetches market capitalization data
// TODO: Implement from Naver Finance or calculate from price * shares
func (c *Client) FetchMarketCap(ctx context.Context, stockCode string, date time.Time) (*MarketCapData, error) {
	c.logger.WithField("stock_code", stockCode).Warn("Market cap fetching not implemented yet")
	return nil, fmt.Errorf("not implemented")
}
