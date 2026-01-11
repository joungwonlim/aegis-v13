package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// investorTrendResponse represents the Naver Stock API response for investor trend
type investorTrendResponse struct {
	ItemCode                    string `json:"itemCode"`
	Bizdate                     string `json:"bizdate"`
	ForeignerPureBuyQuant       string `json:"foreignerPureBuyQuant"`
	OrganPureBuyQuant           string `json:"organPureBuyQuant"`
	IndividualPureBuyQuant      string `json:"individualPureBuyQuant"`
	ClosePrice                  string `json:"closePrice"`
	CompareToPreviousClosePrice string `json:"compareToPreviousClosePrice"`
	AccumulatedTradingVolume    string `json:"accumulatedTradingVolume"`
}

// FetchInvestorFlow fetches investor trading flow data from Naver Stock API
// ⭐ SSOT: Naver Finance 투자자 수급 데이터 호출은 이 함수에서만
func (c *Client) FetchInvestorFlow(ctx context.Context, stockCode string, from, to time.Time) ([]InvestorFlowData, error) {
	// Use the new Naver Stock mobile API
	url := fmt.Sprintf("https://m.stock.naver.com/api/stock/%s/trend?period=day", stockCode)

	resp, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp []investorTrendResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var result []InvestorFlowData
	for _, item := range apiResp {
		// Parse date (format: "20260109")
		tradeDate, err := time.Parse("20060102", item.Bizdate)
		if err != nil {
			c.logger.WithError(err).WithField("bizdate", item.Bizdate).Warn("Failed to parse date")
			continue
		}

		// Filter by date range
		if tradeDate.Before(from) || tradeDate.After(to) {
			continue
		}

		// Parse quantities (format: "-6,672,482" or "+1,990,474")
		foreignNet := parseQuantity(item.ForeignerPureBuyQuant)
		instNet := parseQuantity(item.OrganPureBuyQuant)
		indivNet := parseQuantity(item.IndividualPureBuyQuant)

		result = append(result, InvestorFlowData{
			StockCode:      stockCode,
			TradeDate:      tradeDate,
			ForeignNet:     foreignNet,
			InstitutionNet: instNet,
			IndividualNet:  indivNet,
			FinancialNet:   0, // Not available in this API
			InsuranceNet:   0,
			TrustNet:       0,
			PensionNet:     0,
		})
	}

	// Sort by date ascending (API returns descending)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	c.logger.WithFields(map[string]interface{}{
		"stock_code": stockCode,
		"count":      len(result),
	}).Debug("Fetched investor flow")

	return result, nil
}

// parseQuantity parses quantity strings like "-6,672,482" or "+1,990,474"
func parseQuantity(s string) int64 {
	if s == "" {
		return 0
	}
	// Remove commas and plus sign
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "+", "")
	s = strings.TrimSpace(s)

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
