package handlers

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/internal/external/naver"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// RankingHandler handles ranking-related API endpoints
// ⭐ SSOT: 랭킹 API 핸들러는 이 구조체에서만
type RankingHandler struct {
	pool        *pgxpool.Pool
	naverClient *naver.Client
	logger      *logger.Logger
}

// NewRankingHandler creates a new ranking handler
func NewRankingHandler(pool *pgxpool.Pool, naverClient *naver.Client, log *logger.Logger) *RankingHandler {
	return &RankingHandler{
		pool:        pool,
		naverClient: naverClient,
		logger:      log,
	}
}

// RankingItem represents a single ranking item
type RankingItem struct {
	ID           int     `json:"ID"`
	SnapshotDate string  `json:"SnapshotDate"`
	SnapshotTime string  `json:"SnapshotTime"`
	Category     string  `json:"Category"`
	Market       string  `json:"Market"`
	RankPosition int     `json:"RankPosition"`
	StockCode    string  `json:"StockCode"`
	StockName    string  `json:"StockName"`
	CurrentPrice float64 `json:"CurrentPrice"`
	PriceChange  float64 `json:"PriceChange"`
	ChangeRate   float64 `json:"ChangeRate"`
	Volume       int64   `json:"Volume"`
	TradingValue float64 `json:"TradingValue"` // numeric(15,0) -> float64
	HighPrice    float64 `json:"HighPrice"`
	LowPrice     float64 `json:"LowPrice"`
	MarketCap    int64   `json:"MarketCap"`
	CreatedAt    string  `json:"CreatedAt"`
}

// GetRanking returns ranking data by category and market
// GET /api/v1/ranking/{category}?market=KOSPI|KOSDAQ
func (h *RankingHandler) GetRanking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	category := vars["category"]
	market := r.URL.Query().Get("market")

	if category == "" {
		category = "top"
	}
	if market == "" {
		market = "KOSPI"
	}

	// 네이버에서 실시간 조회 (종목코드만 가져와서 DB와 결합)
	naverCategories := map[string]naver.RankingCategory{
		"high52week":   naver.RankingHigh52Week,   // 52주 신고가
		"low52week":    naver.RankingLow52Week,    // 52주 신저가
		"upper":        naver.RankingUpper,        // 상승률
		"lower":        naver.RankingLower,        // 하락률
		"trading":      naver.RankingVolume,       // 거래량상위
		"tradingValue": naver.RankingValue,        // 거래대금상위
		"quantHigh":    naver.RankingVolumeSurge,  // 거래량급증
		"top":          naver.RankingMarketCap,    // 시가총액 (검색상위)
		"new":          naver.RankingNewListing,   // 신규상장
	}

	if naverCat, ok := naverCategories[category]; ok && h.naverClient != nil {
		h.getRankingFromNaver(w, r, naverCat, category, market)
		return
	}

	// Get latest trade date
	var latestDate time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT MAX(trade_date) FROM data.daily_prices
		WHERE trade_date <= CURRENT_DATE
	`).Scan(&latestDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest trade date")
		respondError(w, http.StatusInternalServerError, "Failed to get latest trade date")
		return
	}

	// Calculate date range
	fromDate := latestDate.AddDate(0, 0, -7)
	fromDate52Week := latestDate.AddDate(-1, 0, 0) // 52주 = 1년

	// Build query based on category
	orderClause := h.getOrderClause(category)

	query := `
		WITH latest_prices AS (
			SELECT
				p.stock_code,
				p.close_price::float8 as close_price,
				p.open_price::float8 as open_price,
				p.high_price::float8 as high_price,
				p.low_price::float8 as low_price,
				p.volume,
				p.trading_value::float8 as trading_value,
				LAG(p.close_price::float8) OVER (PARTITION BY p.stock_code ORDER BY p.trade_date) as prev_close
			FROM data.daily_prices p
			WHERE p.trade_date >= $1
			  AND p.trade_date <= $2
		),
		week52_stats AS (
			SELECT
				stock_code,
				MAX(high_price)::float8 as high_52week,
				MIN(low_price)::float8 as low_52week
			FROM data.daily_prices
			WHERE trade_date >= $4
			  AND trade_date <= $2
			GROUP BY stock_code
		),
		ranked AS (
			SELECT
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lp.close_price as current_price,
				COALESCE(lp.close_price - lp.prev_close, 0)::float8 as price_change,
				CASE
					WHEN lp.prev_close > 0 THEN ((lp.close_price - lp.prev_close) / lp.prev_close * 100)::float8
					ELSE 0::float8
				END as change_rate,
				COALESCE(lp.volume, 0) as volume,
				COALESCE(lp.trading_value, 0)::float8 as trading_value,
				COALESCE(lp.high_price, 0)::float8 as high_price,
				COALESCE(lp.low_price, 0)::float8 as low_price,
				COALESCE(mc.market_cap / 100000000, 0)::bigint as market_cap_bil,
				COALESCE(w52.high_52week, 0)::float8 as high_52week,
				COALESCE(w52.low_52week, 0)::float8 as low_52week,
				CASE WHEN w52.high_52week > 0 THEN (lp.close_price / w52.high_52week * 100)::float8 ELSE 0 END as pct_from_high,
				CASE WHEN w52.low_52week > 0 THEN (lp.close_price / w52.low_52week * 100)::float8 ELSE 0 END as pct_from_low,
				ROW_NUMBER() OVER (` + orderClause + `) as rank_position
			FROM data.stocks s
			LEFT JOIN (
				SELECT DISTINCT ON (stock_code) *
				FROM latest_prices
				ORDER BY stock_code, prev_close DESC NULLS LAST
			) lp ON s.code = lp.stock_code
			LEFT JOIN (
				SELECT DISTINCT ON (stock_code) stock_code, market_cap
				FROM data.market_cap
				ORDER BY stock_code, trade_date DESC
			) mc ON s.code = mc.stock_code
			LEFT JOIN week52_stats w52 ON s.code = w52.stock_code
			WHERE s.market = $3
			  AND s.status = 'active'
			  AND lp.close_price IS NOT NULL
			  AND lp.close_price > 0
		)
		SELECT
			stock_code,
			stock_name,
			market,
			current_price,
			price_change,
			change_rate,
			volume,
			trading_value,
			high_price,
			low_price,
			market_cap_bil,
			rank_position
		FROM ranked
		ORDER BY rank_position
	`

	rows, err := h.pool.Query(ctx, query, fromDate, latestDate, market, fromDate52Week)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"from":     fromDate,
			"to":       latestDate,
			"market":   market,
			"category": category,
		}).Error("Failed to query ranking")
		respondError(w, http.StatusInternalServerError, "Query error: "+err.Error())
		return
	}
	defer rows.Close()

	items := make([]RankingItem, 0)
	now := time.Now()
	snapshotDate := latestDate.Format("2006-01-02")
	snapshotTime := now.Format("15:04:05")

	scanErrors := 0
	for rows.Next() {
		var item RankingItem
		err := rows.Scan(
			&item.StockCode,
			&item.StockName,
			&item.Market,
			&item.CurrentPrice,
			&item.PriceChange,
			&item.ChangeRate,
			&item.Volume,
			&item.TradingValue,
			&item.HighPrice,
			&item.LowPrice,
			&item.MarketCap,
			&item.RankPosition,
		)
		if err != nil {
			if scanErrors == 0 {
				h.logger.WithError(err).Error("Failed to scan ranking item")
			}
			scanErrors++
			continue
		}

		item.ID = item.RankPosition
		item.SnapshotDate = snapshotDate
		item.SnapshotTime = snapshotTime
		item.Category = category
		item.CreatedAt = now.Format(time.RFC3339)

		items = append(items, item)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"category": category,
			"count":    len(items),
			"items":    items,
		},
	})
}

// getOrderClause returns ORDER BY clause based on category
// NOTE: WINDOW 함수 내에서는 같은 SELECT의 alias를 참조할 수 없으므로 실제 컬럼/표현식 사용
func (h *RankingHandler) getOrderClause(category string) string {
	switch category {
	case "trading":
		return "ORDER BY lp.volume DESC NULLS LAST"
	case "capitalization", "top":
		return "ORDER BY mc.market_cap DESC NULLS LAST"
	case "quantHigh":
		return "ORDER BY CASE WHEN lp.prev_close > 0 THEN (lp.close_price - lp.prev_close) / lp.prev_close ELSE 0 END DESC NULLS LAST"
	case "quantLow":
		return "ORDER BY CASE WHEN lp.prev_close > 0 THEN (lp.close_price - lp.prev_close) / lp.prev_close ELSE 0 END ASC NULLS LAST"
	case "priceTop":
		return "ORDER BY lp.close_price DESC NULLS LAST"
	case "upper":
		return "ORDER BY CASE WHEN lp.prev_close > 0 THEN (lp.close_price - lp.prev_close) / lp.prev_close ELSE 0 END DESC NULLS LAST"
	case "lower":
		return "ORDER BY CASE WHEN lp.prev_close > 0 THEN (lp.close_price - lp.prev_close) / lp.prev_close ELSE 0 END ASC NULLS LAST"
	case "high52week":
		// 52주 신고가: 현재가가 52주 최고가에 가까운 순 (95% 이상만, 100%에 가까울수록 상위)
		return "ORDER BY CASE WHEN w52.high_52week > 0 THEN lp.close_price / w52.high_52week ELSE 0 END DESC NULLS LAST"
	case "low52week":
		// 52주 신저가: 현재가가 52주 최저가에 가까운 순 (105% 이하만, 100%에 가까울수록 상위)
		return "ORDER BY CASE WHEN w52.low_52week > 0 THEN lp.close_price / w52.low_52week ELSE 999 END ASC NULLS LAST"
	default:
		return "ORDER BY mc.market_cap DESC NULLS LAST"
	}
}

// GetRankingStatus returns ranking collection status
// GET /api/v1/ranking/status
func (h *RankingHandler) GetRankingStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get latest data date
	var latestDate time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(trade_date), CURRENT_DATE) FROM data.daily_prices
	`).Scan(&latestDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest date")
		latestDate = time.Now()
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"categories": []string{
				"trading",      // 거래량상위
				"tradingValue", // 거래대금상위
				"quantHigh",    // 거래량급증
				"upper",        // 상승률
				"lower",        // 하락률
				"high52week",   // 52주 신고가
				"low52week",    // 52주 신저가
				"top",          // 시가총액
				"new",          // 신규상장
			},
			"is_running":      false,
			"last_collection": latestDate.Format(time.RFC3339),
			"markets":         []string{"KOSPI", "KOSDAQ"},
			"source":          "naver", // 데이터 출처
		},
	})
}

// getRankingFromNaver fetches stock codes from Naver Finance and enriches with DB data
func (h *RankingHandler) getRankingFromNaver(w http.ResponseWriter, r *http.Request, naverCat naver.RankingCategory, category, market string) {
	ctx := r.Context()

	// 1. 네이버에서 종목코드만 가져옴
	naverItems, err := h.naverClient.GetRanking(ctx, naverCat, market)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"category": category,
			"market":   market,
		}).Error("Failed to fetch ranking from Naver")
		respondError(w, http.StatusInternalServerError, "Failed to fetch ranking from Naver: "+err.Error())
		return
	}

	if len(naverItems) == 0 {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"category": category,
				"count":    0,
				"items":    []RankingItem{},
				"source":   "naver",
			},
		})
		return
	}

	// 2. 종목코드 목록 추출 (순서 유지)
	stockCodes := make([]string, 0, len(naverItems))
	rankMap := make(map[string]int) // 종목코드 -> 순위
	for _, item := range naverItems {
		stockCodes = append(stockCodes, item.StockCode)
		rankMap[item.StockCode] = item.Rank
	}

	// 3. DB에서 종목 상세 정보 조회
	query := `
		WITH latest_prices AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				close_price::float8 as close_price,
				high_price::float8 as high_price,
				low_price::float8 as low_price,
				volume,
				trading_value::float8 as trading_value,
				LAG(close_price::float8) OVER (PARTITION BY stock_code ORDER BY trade_date) as prev_close
			FROM data.daily_prices
			WHERE stock_code = ANY($1)
			ORDER BY stock_code, trade_date DESC
		)
		SELECT
			s.code as stock_code,
			s.name as stock_name,
			s.market,
			COALESCE(lp.close_price, 0) as current_price,
			COALESCE(lp.close_price - lp.prev_close, 0)::float8 as price_change,
			CASE
				WHEN lp.prev_close > 0 THEN ((lp.close_price - lp.prev_close) / lp.prev_close * 100)::float8
				ELSE 0::float8
			END as change_rate,
			COALESCE(lp.volume, 0) as volume,
			COALESCE(lp.trading_value, 0)::float8 as trading_value,
			COALESCE(lp.high_price, 0)::float8 as high_price,
			COALESCE(lp.low_price, 0)::float8 as low_price,
			COALESCE(mc.market_cap / 100000000, 0)::bigint as market_cap_bil
		FROM data.stocks s
		LEFT JOIN latest_prices lp ON s.code = lp.stock_code
		LEFT JOIN (
			SELECT DISTINCT ON (stock_code) stock_code, market_cap
			FROM data.market_cap
			ORDER BY stock_code, trade_date DESC
		) mc ON s.code = mc.stock_code
		WHERE s.code = ANY($1)
	`

	rows, err := h.pool.Query(ctx, query, stockCodes)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query stock details from DB")
		respondError(w, http.StatusInternalServerError, "Database query error: "+err.Error())
		return
	}
	defer rows.Close()

	// 4. DB 결과를 맵으로 저장
	stockMap := make(map[string]RankingItem)
	now := time.Now()
	snapshotDate := now.Format("2006-01-02")
	snapshotTime := now.Format("15:04:05")

	for rows.Next() {
		var item RankingItem
		err := rows.Scan(
			&item.StockCode,
			&item.StockName,
			&item.Market,
			&item.CurrentPrice,
			&item.PriceChange,
			&item.ChangeRate,
			&item.Volume,
			&item.TradingValue,
			&item.HighPrice,
			&item.LowPrice,
			&item.MarketCap,
		)
		if err != nil {
			h.logger.WithError(err).Error("Failed to scan stock row")
			continue
		}
		item.SnapshotDate = snapshotDate
		item.SnapshotTime = snapshotTime
		item.Category = category
		item.CreatedAt = now.Format(time.RFC3339)
		stockMap[item.StockCode] = item
	}

	// 5. 네이버 순위 순서대로 결과 생성
	result := make([]RankingItem, 0, len(naverItems))
	for _, naverItem := range naverItems {
		if item, ok := stockMap[naverItem.StockCode]; ok {
			item.ID = naverItem.Rank
			item.RankPosition = naverItem.Rank
			result = append(result, item)
		}
	}

	h.logger.WithFields(map[string]interface{}{
		"category":    category,
		"market":      market,
		"naver_count": len(naverItems),
		"db_count":    len(result),
		"source":      "naver+db",
	}).Info("Fetched ranking from Naver (codes) + DB (details)")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"category":  category,
			"count":     len(result),
			"items":     result,
			"source":    "naver",
			"fetchedAt": now.Format(time.RFC3339), // 조회 시간
		},
	})
}
