package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RankingItem represents a single ranking item (종목코드만 사용)
type RankingItem struct {
	Rank      int
	StockCode string
}

// RankingCategory represents the type of ranking
type RankingCategory string

const (
	RankingHigh52Week   RankingCategory = "high52week"   // 52주 신고가
	RankingLow52Week    RankingCategory = "low52week"    // 52주 신저가
	RankingUpper        RankingCategory = "upper"        // 상승률
	RankingLower        RankingCategory = "lower"        // 하락률
	RankingVolume       RankingCategory = "trading"      // 거래량상위
	RankingValue        RankingCategory = "tradingValue" // 거래대금상위
	RankingVolumeSurge  RankingCategory = "quantHigh"    // 거래량급증
	RankingMarketCap    RankingCategory = "top"          // 시가총액 (검색상위 대용)
	RankingNewListing   RankingCategory = "new"          // 신규상장
)

// API 타입 1: m.stock.naver.com (52주, 상승/하락)
var mobileAPIEndpoints = map[RankingCategory]string{
	RankingHigh52Week: "high52week",
	RankingLow52Week:  "low52week",
	RankingUpper:      "up",
	RankingLower:      "down",
	RankingMarketCap:  "marketValue",
}

// API 타입 2: api.stock.naver.com (거래량, 거래대금, 거래량급증)
type stockAPIConfig struct {
	Type     string
	SortType string
}

var stockAPIEndpoints = map[RankingCategory]stockAPIConfig{
	RankingVolume:      {Type: "ALL", SortType: "ACC_TRADING_VOLUME"},
	RankingValue:       {Type: "ALL", SortType: "ACC_TRADING_VALUE"},
	RankingVolumeSurge: {Type: "ALL", SortType: "TRADING_VOLUME_INCREASE"},
	RankingNewListing:  {Type: "NEW", SortType: ""},
}

// Mobile API response (m.stock.naver.com)
type mobileAPIResponse struct {
	StockListSortType     string            `json:"stockListSortType"`
	StockListCategoryType string            `json:"stockListCategoryType"`
	Stocks                []mobileStockItem `json:"stocks"`
}

type mobileStockItem struct {
	ItemCode  string `json:"itemCode"`
	StockName string `json:"stockName"`
}

// Stock API response (api.stock.naver.com)
type stockAPIResponse struct {
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalCount int              `json:"totalCount"`
	Stocks     []stockAPIItem   `json:"stocks"`
}

type stockAPIItem struct {
	ItemCode  string `json:"itemCode"`
	StockName string `json:"stockName"`
}

// GetRanking fetches ranking data from Naver API
// Returns only stock codes in ranking order
// market: "KOSPI" or "KOSDAQ"
func (c *Client) GetRanking(ctx context.Context, category RankingCategory, market string) ([]RankingItem, error) {
	// Check mobile API first
	if endpoint, ok := mobileAPIEndpoints[category]; ok {
		return c.getRankingFromMobileAPI(ctx, endpoint, market, category)
	}

	// Check stock API
	if config, ok := stockAPIEndpoints[category]; ok {
		return c.getRankingFromStockAPI(ctx, config, market, category)
	}

	return nil, fmt.Errorf("unknown ranking category: %s", category)
}

// getRankingFromMobileAPI fetches from m.stock.naver.com
func (c *Client) getRankingFromMobileAPI(ctx context.Context, endpoint, market string, category RankingCategory) ([]RankingItem, error) {
	apiURL := fmt.Sprintf("https://m.stock.naver.com/api/stocks/%s/%s?page=1&pageSize=100", endpoint, market)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp mobileAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	items := make([]RankingItem, 0, len(apiResp.Stocks))
	for i, stock := range apiResp.Stocks {
		items = append(items, RankingItem{
			Rank:      i + 1,
			StockCode: stock.ItemCode,
		})
	}

	c.logger.WithFields(map[string]interface{}{
		"category": category,
		"market":   market,
		"count":    len(items),
		"source":   "m.stock.naver.com",
	}).Debug("Fetched ranking from Naver Mobile API")

	return items, nil
}

// getRankingFromStockAPI fetches from api.stock.naver.com
func (c *Client) getRankingFromStockAPI(ctx context.Context, config stockAPIConfig, market string, category RankingCategory) ([]RankingItem, error) {
	var apiURL string
	if config.SortType == "" {
		// 신규상장 등 sortType이 없는 경우
		apiURL = fmt.Sprintf(
			"https://api.stock.naver.com/stock/exchange/%s?type=%s&page=1&pageSize=100",
			market, config.Type,
		)
	} else {
		apiURL = fmt.Sprintf(
			"https://api.stock.naver.com/stock/exchange/%s?type=%s&sortType=%s&page=1&pageSize=100",
			market, config.Type, config.SortType,
		)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp stockAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	items := make([]RankingItem, 0, len(apiResp.Stocks))
	for i, stock := range apiResp.Stocks {
		items = append(items, RankingItem{
			Rank:      i + 1,
			StockCode: stock.ItemCode,
		})
	}

	c.logger.WithFields(map[string]interface{}{
		"category": category,
		"market":   market,
		"count":    len(items),
		"source":   "api.stock.naver.com",
	}).Debug("Fetched ranking from Naver Stock API")

	return items, nil
}

// RankingData contains all ranking data with timestamp
type RankingData struct {
	Category  RankingCategory
	Market    string
	Items     []RankingItem
	FetchedAt time.Time
}

// GetAllRankings fetches all ranking categories for a market
func (c *Client) GetAllRankings(ctx context.Context, market string) ([]RankingData, error) {
	categories := []RankingCategory{
		RankingHigh52Week,
		RankingLow52Week,
		RankingUpper,
		RankingLower,
		RankingVolume,
		RankingValue,
		RankingVolumeSurge,
		RankingMarketCap,
		RankingNewListing,
	}

	var results []RankingData
	for _, cat := range categories {
		items, err := c.GetRanking(ctx, cat, market)
		if err != nil {
			c.logger.WithError(err).WithFields(map[string]interface{}{
				"category": cat,
				"market":   market,
			}).Warn("Failed to fetch ranking")
			continue
		}

		results = append(results, RankingData{
			Category:  cat,
			Market:    market,
			Items:     items,
			FetchedAt: time.Now(),
		})
	}

	return results, nil
}
