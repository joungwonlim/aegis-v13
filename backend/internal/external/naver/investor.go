package naver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// FetchInvestorFlow fetches investor trading flow data from Naver Finance
// ⭐ SSOT: Naver Finance 투자자 수급 데이터 호출은 이 함수에서만
func (c *Client) FetchInvestorFlow(ctx context.Context, stockCode string, from, to time.Time) ([]InvestorFlowData, error) {
	var allTrades []InvestorFlowData
	noDataPages := 0

	// Naver Finance 페이지네이션 처리 (최대 150페이지)
	for page := 1; page <= 150; page++ {
		select {
		case <-ctx.Done():
			return allTrades, ctx.Err()
		default:
		}

		url := fmt.Sprintf("https://finance.naver.com/item/frgn.naver?code=%s&page=%d", stockCode, page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return allTrades, fmt.Errorf("create request failed: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://finance.naver.com/")

		resp, err := c.httpClient.Get(ctx, url)
		if err != nil {
			return allTrades, fmt.Errorf("HTTP request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return allTrades, fmt.Errorf("read response body failed: %w", err)
		}

		trades, lastDate, hasMore := c.parseInvestorHTML(string(body), stockCode, from, to)
		allTrades = append(allTrades, trades...)

		// 기준일보다 이전 데이터면 종료
		if !lastDate.IsZero() && lastDate.Before(from) {
			break
		}

		// 더 이상 페이지 없으면 종료
		if !hasMore {
			break
		}

		// 연속으로 데이터 없으면 종료
		if lastDate.IsZero() {
			noDataPages++
			if noDataPages >= 3 {
				break
			}
		} else {
			noDataPages = 0
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"stock_code": stockCode,
		"count":      len(allTrades),
	}).Debug("Fetched investor flow")
	return allTrades, nil
}

// parseInvestorHTML parses Naver Finance HTML page for investor trading data
func (c *Client) parseInvestorHTML(html string, stockCode string, from, to time.Time) ([]InvestorFlowData, time.Time, bool) {
	var trades []InvestorFlowData
	var lastDate time.Time

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return trades, lastDate, false
	}

	// Naver Finance HTML 구조: 두번째 테이블이 데이터 테이블
	tables := doc.Find("table.type2")
	if tables.Length() < 2 {
		return trades, lastDate, false
	}

	table := tables.Eq(1)
	dateRe := regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}$`)

	parseNum := func(s string) int64 {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, ",", "")
		s = strings.ReplaceAll(s, "+", "")
		if s == "" || s == "-" {
			return 0
		}
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}

	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 7 {
			return
		}

		// 날짜 추출
		dateText := strings.TrimSpace(cells.Eq(0).Text())
		if !dateRe.MatchString(dateText) {
			return
		}

		dateStr := strings.ReplaceAll(dateText, ".", "-")
		tradeDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return
		}

		lastDate = tradeDate

		// 기간 필터
		if tradeDate.Before(from) || tradeDate.After(to) {
			return
		}

		// 데이터 추출
		// 컬럼: 날짜 | 종가 | 대비 | 등락률 | 거래량 | 기관 | 외국인
		instNet := parseNum(cells.Eq(5).Text())       // 기관 순매수
		foreignNet := parseNum(cells.Eq(6).Text())    // 외국인 순매수
		individualNet := -(foreignNet + instNet)       // 개인 순매수 (계산)

		trades = append(trades, InvestorFlowData{
			StockCode:      stockCode,
			TradeDate:      tradeDate,
			ForeignNet:     foreignNet,
			InstitutionNet: instNet,
			IndividualNet:  individualNet,
			FinancialNet:   0, // 상세 데이터는 별도 페이지
			InsuranceNet:   0,
			TrustNet:       0,
			PensionNet:     0,
		})
	})

	// 다음 페이지 존재 여부 확인
	hasMore := doc.Find(".pgRR").Length() > 0
	return trades, lastDate, hasMore
}
