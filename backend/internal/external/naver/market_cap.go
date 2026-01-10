package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RankingStockItem represents a stock item from Naver ranking API
type RankingStockItem struct {
	ItemCode          string `json:"itemcode"`
	ItemName          string `json:"itemname"`
	NowVal            string `json:"nowVal"`            // 현재가
	MarketSum         string `json:"marketSum"`         // 시가총액 (원)
	ListedStockCnt    string `json:"listedStockCnt"`    // 상장주식수
}

// FetchMarketCap fetches market cap for a single stock
// ⭐ SSOT: Naver 시가총액 호출은 이 함수에서만
func (c *Client) FetchMarketCapForStock(ctx context.Context, stockCode string) (*MarketCapData, error) {
	// Use ranking API with specific stock filter
	// This is a workaround since there's no direct single-stock market cap API
	url := fmt.Sprintf("https://stock.naver.com/api/domestic/market/stock/default?orderType=marketSum&marketType=KOSPI&page=1&pageSize=100")

	resp, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response as array
	var items []RankingStockItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Find the stock
	for _, item := range items {
		if item.ItemCode == stockCode {
			return parseMarketCapData(item)
		}
	}

	// Try KOSDAQ if not found in KOSPI
	url = fmt.Sprintf("https://stock.naver.com/api/domestic/market/stock/default?orderType=marketSum&marketType=KOSDAQ&page=1&pageSize=100")
	resp2, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp2.Body.Close()

	items = nil
	if err := json.NewDecoder(resp2.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	for _, item := range items {
		if item.ItemCode == stockCode {
			return parseMarketCapData(item)
		}
	}

	return nil, fmt.Errorf("stock %s not found in ranking", stockCode)
}

// FetchAllMarketCaps fetches market caps for all stocks in a market
// ⭐ SSOT: 전체 종목 시가총액 호출은 이 함수에서만
func (c *Client) FetchAllMarketCaps(ctx context.Context, market string) ([]MarketCapData, error) {
	allData := []MarketCapData{}

	// Fetch multiple pages (KOSPI/KOSDAQ each have ~1000 stocks, 100 per page)
	maxPages := 15
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("https://stock.naver.com/api/domestic/market/stock/default?orderType=marketSum&marketType=%s&page=%d&pageSize=100", market, page)

		resp, err := c.httpClient.Get(ctx, url)
		if err != nil {
			c.logger.WithError(err).WithField("page", page).Warn("Failed to fetch market cap page")
			continue
		}

		var items []RankingStockItem
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			resp.Body.Close()
			c.logger.WithError(err).WithField("page", page).Warn("Failed to decode market cap page")
			continue
		}
		resp.Body.Close()

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			data, err := parseMarketCapData(item)
			if err != nil {
				c.logger.WithError(err).WithField("stock_code", item.ItemCode).Debug("Failed to parse market cap")
				continue
			}
			allData = append(allData, *data)
		}

		c.logger.WithFields(map[string]interface{}{
			"market": market,
			"page":   page,
			"count":  len(items),
		}).Debug("Fetched market cap page")
	}

	return allData, nil
}

// parseMarketCapData parses RankingStockItem into MarketCapData
func parseMarketCapData(item RankingStockItem) (*MarketCapData, error) {
	// Parse market cap (remove commas, handle decimal point)
	marketCapStr := strings.ReplaceAll(item.MarketSum, ",", "")
	marketCapFloat, err := strconv.ParseFloat(marketCapStr, 64)
	if err != nil {
		return nil, fmt.Errorf("parse market cap: %w", err)
	}
	marketCap := int64(marketCapFloat)

	// Parse shares outstanding (remove commas, handle decimal point)
	sharesStr := strings.ReplaceAll(item.ListedStockCnt, ",", "")
	sharesFloat, err := strconv.ParseFloat(sharesStr, 64)
	if err != nil {
		return nil, fmt.Errorf("parse shares outstanding: %w", err)
	}
	shares := int64(sharesFloat)

	return &MarketCapData{
		StockCode:         item.ItemCode,
		TradeDate:         time.Now().Truncate(24 * time.Hour),
		MarketCap:         marketCap,
		SharesOutstanding: shares,
	}, nil
}
