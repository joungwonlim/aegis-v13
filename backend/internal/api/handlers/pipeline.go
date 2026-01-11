package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// PipelineHandler handles pipeline-related API endpoints (S2-S5)
// ⭐ SSOT: 파이프라인 API 핸들러는 여기서만
type PipelineHandler struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewPipelineHandler creates a new pipeline handler
func NewPipelineHandler(pool *pgxpool.Pool, log *logger.Logger) *PipelineHandler {
	return &PipelineHandler{
		pool:   pool,
		logger: log,
	}
}

// UniverseItem represents a stock in universe (S1)
type UniverseItem struct {
	StockCode    string  `json:"stockCode"`
	StockName    string  `json:"stockName"`
	Market       string  `json:"market"`
	CurrentPrice float64 `json:"currentPrice"`
	ChangeRate   float64 `json:"changeRate"`
	Volume       int64   `json:"volume"`
	MarketCap    int64   `json:"marketCap"`
}

// SignalItem represents a stock with signal scores (S2)
type SignalItem struct {
	StockCode   string  `json:"stockCode"`
	StockName   string  `json:"stockName"`
	Market      string  `json:"market"`
	CalcDate    string  `json:"calcDate"`
	Momentum    float64 `json:"momentum"`
	Technical   float64 `json:"technical"`
	Value       float64 `json:"value"`
	Quality     float64 `json:"quality"`
	Flow        float64 `json:"flow"`
	Event       float64 `json:"event"`
	TotalScore  float64 `json:"totalScore"`
}

// ScreenedItem represents a stock that passed hard cut (S3)
type ScreenedItem struct {
	StockCode   string   `json:"stockCode"`
	StockName   string   `json:"stockName"`
	Market      string   `json:"market"`
	CalcDate    string   `json:"calcDate"`
	Momentum    float64  `json:"momentum"`
	Technical   float64  `json:"technical"`
	Value       float64  `json:"value"`
	Quality     float64  `json:"quality"`
	Flow        float64  `json:"flow"`
	Event       float64  `json:"event"`
	TotalScore  float64  `json:"totalScore"`
	PER         *float64 `json:"per"`
	PBR         *float64 `json:"pbr"`
	ROE         *float64 `json:"roe"`
	PassedAll   bool     `json:"passedAll"`
}

// RankingItem represents a ranked stock (S4)
type RankedItem struct {
	StockCode    string  `json:"stockCode"`
	StockName    string  `json:"stockName"`
	Market       string  `json:"market"`
	Rank         int     `json:"rank"`
	TotalScore   float64 `json:"totalScore"`
	Momentum     float64 `json:"momentum"`
	Technical    float64 `json:"technical"`
	Value        float64 `json:"value"`
	Quality      float64 `json:"quality"`
	Flow         float64 `json:"flow"`
	Event        float64 `json:"event"`
	CurrentPrice float64 `json:"currentPrice"`
	ChangeRate   float64 `json:"changeRate"`
}

// PortfolioItem represents a portfolio position (S5)
type PortfolioItem struct {
	StockCode    string  `json:"stockCode"`
	StockName    string  `json:"stockName"`
	Market       string  `json:"market"`
	Weight       float64 `json:"weight"`
	TargetQty    int     `json:"targetQty"`
	Action       string  `json:"action"`
	Reason       string  `json:"reason"`
	CurrentPrice float64 `json:"currentPrice"`
	ChangeRate   float64 `json:"changeRate"`
}

// GetUniverse returns universe stocks (S1)
// GET /api/v1/pipeline/universe?market=KOSPI|KOSDAQ
func (h *PipelineHandler) GetUniverse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	market := r.URL.Query().Get("market")

	// Get latest universe snapshot
	var snapshotDate time.Time
	var eligibleStocks []string
	var totalCount int

	err := h.pool.QueryRow(ctx, `
		SELECT snapshot_date, eligible_stocks, total_count
		FROM data.universe_snapshots
		ORDER BY snapshot_date DESC
		LIMIT 1
	`).Scan(&snapshotDate, &eligibleStocks, &totalCount)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get universe snapshot")
		respondError(w, http.StatusInternalServerError, "Failed to get universe snapshot")
		return
	}

	// Build query with market filter
	marketFilter := ""
	if market != "" && market != "ALL" {
		marketFilter = " AND s.market = '" + market + "'"
	}

	query := `
		WITH latest_prices AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				close_price::float8 as close_price,
				volume,
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
			CASE
				WHEN lp.prev_close > 0 THEN ((lp.close_price - lp.prev_close) / lp.prev_close * 100)::float8
				ELSE 0::float8
			END as change_rate,
			COALESCE(lp.volume, 0) as volume,
			COALESCE(mc.market_cap / 100000000, 0)::bigint as market_cap_bil
		FROM data.stocks s
		LEFT JOIN latest_prices lp ON s.code = lp.stock_code
		LEFT JOIN (
			SELECT DISTINCT ON (stock_code) stock_code, market_cap
			FROM data.market_cap
			ORDER BY stock_code, trade_date DESC
		) mc ON s.code = mc.stock_code
		WHERE s.code = ANY($1)
		  AND s.status = 'active'
		` + marketFilter + `
		ORDER BY mc.market_cap DESC NULLS LAST
	`

	rows, err := h.pool.Query(ctx, query, eligibleStocks)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query universe stocks")
		respondError(w, http.StatusInternalServerError, "Query error")
		return
	}
	defer rows.Close()

	items := make([]UniverseItem, 0)
	for rows.Next() {
		var item UniverseItem
		err := rows.Scan(
			&item.StockCode,
			&item.StockName,
			&item.Market,
			&item.CurrentPrice,
			&item.ChangeRate,
			&item.Volume,
			&item.MarketCap,
		)
		if err != nil {
			h.logger.WithError(err).Error("Failed to scan universe item")
			continue
		}
		items = append(items, item)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"date":       snapshotDate.Format("2006-01-02"),
			"market":     market,
			"count":      len(items),
			"totalCount": totalCount,
			"items":      items,
		},
	})
}

// GetSignals returns signal scores for all stocks (S2)
// GET /api/v1/pipeline/signals?market=KOSPI|KOSDAQ|ALL
func (h *PipelineHandler) GetSignals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	market := r.URL.Query().Get("market")

	// Get latest calc date
	var latestDate time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT MAX(calc_date) FROM signals.factor_scores
	`).Scan(&latestDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest calc date")
		respondError(w, http.StatusInternalServerError, "Failed to get latest calc date")
		return
	}

	// Build query with optional market filter
	marketFilter := ""
	args := []interface{}{latestDate}
	if market != "" && market != "ALL" {
		marketFilter = " AND s.market = $2"
		args = append(args, market)
	}

	query := `
		SELECT
			f.stock_code,
			s.name as stock_name,
			s.market,
			f.calc_date,
			f.momentum::float8,
			f.technical::float8,
			f.value::float8,
			f.quality::float8,
			f.flow::float8,
			f.event::float8,
			COALESCE(f.total_score, 0)::float8 as total_score
		FROM signals.factor_scores f
		JOIN data.stocks s ON f.stock_code = s.code
		WHERE f.calc_date = $1
		  ` + marketFilter + `
		  AND s.status = 'active'
		ORDER BY f.total_score DESC NULLS LAST
	`

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query signals")
		respondError(w, http.StatusInternalServerError, "Query error")
		return
	}
	defer rows.Close()

	items := make([]SignalItem, 0)
	for rows.Next() {
		var item SignalItem
		var calcDate time.Time
		err := rows.Scan(
			&item.StockCode,
			&item.StockName,
			&item.Market,
			&calcDate,
			&item.Momentum,
			&item.Technical,
			&item.Value,
			&item.Quality,
			&item.Flow,
			&item.Event,
			&item.TotalScore,
		)
		if err != nil {
			h.logger.WithError(err).Error("Failed to scan signal item")
			continue
		}
		item.CalcDate = calcDate.Format("2006-01-02")
		items = append(items, item)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"date":   latestDate.Format("2006-01-02"),
			"market": market,
			"count":  len(items),
			"items":  items,
		},
	})
}

// GetScreened returns stocks that passed hard cut filtering (S2 Screener)
// GET /api/v1/pipeline/screened?market=KOSPI|KOSDAQ|ALL
// Hard Cut 조건 (재무 지표만):
//   - PER > 0 AND <= 50 (고평가/적자 제외)
//   - PBR >= 0.2 (자산가치 필터)
//   - ROE >= 5 (수익성 필터)
func (h *PipelineHandler) GetScreened(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	market := r.URL.Query().Get("market")

	// Get latest calc date
	var latestDate time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT MAX(calc_date) FROM signals.factor_scores
	`).Scan(&latestDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest calc date")
		respondError(w, http.StatusInternalServerError, "Failed to get latest calc date")
		return
	}

	// Build query with hard cut conditions
	marketFilter := ""
	args := []interface{}{latestDate}
	if market != "" && market != "ALL" {
		marketFilter = " AND s.market = $2"
		args = append(args, market)
	}

	// Join with fundamentals for PER/PBR/ROE
	// Use latest fundamentals data for each stock
	// Hard Cut: 재무 지표만 사용 (팩터 조건 없음)
	query := `
		WITH latest_fundamentals AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				per,
				pbr,
				roe
			FROM data.fundamentals
			ORDER BY stock_code, report_date DESC
		)
		SELECT
			f.stock_code,
			s.name as stock_name,
			s.market,
			f.calc_date,
			f.momentum::float8,
			f.technical::float8,
			f.value::float8,
			f.quality::float8,
			f.flow::float8,
			f.event::float8,
			COALESCE(f.total_score, 0)::float8 as total_score,
			lf.per::float8,
			lf.pbr::float8,
			lf.roe::float8
		FROM signals.factor_scores f
		JOIN data.stocks s ON f.stock_code = s.code
		LEFT JOIN latest_fundamentals lf ON f.stock_code = lf.stock_code
		WHERE f.calc_date = $1
		  ` + marketFilter + `
		  AND s.status = 'active'
		  -- Hard Cut: 재무 지표만 (PER/PBR/ROE)
		  AND lf.per > 0 AND lf.per <= 50
		  AND lf.pbr >= 0.2
		  AND lf.roe >= 5
		ORDER BY f.total_score DESC NULLS LAST
	`

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query screened stocks")
		respondError(w, http.StatusInternalServerError, "Query error")
		return
	}
	defer rows.Close()

	items := make([]ScreenedItem, 0)
	for rows.Next() {
		var item ScreenedItem
		var calcDate time.Time
		err := rows.Scan(
			&item.StockCode,
			&item.StockName,
			&item.Market,
			&calcDate,
			&item.Momentum,
			&item.Technical,
			&item.Value,
			&item.Quality,
			&item.Flow,
			&item.Event,
			&item.TotalScore,
			&item.PER,
			&item.PBR,
			&item.ROE,
		)
		if err != nil {
			h.logger.WithError(err).Error("Failed to scan screened item")
			continue
		}
		item.CalcDate = calcDate.Format("2006-01-02")
		item.PassedAll = true
		items = append(items, item)
	}

	// Get total count before screening for stats
	var totalBeforeScreening int
	countQuery := `
		SELECT COUNT(*)
		FROM signals.factor_scores f
		JOIN data.stocks s ON f.stock_code = s.code
		WHERE f.calc_date = $1
		  ` + marketFilter + `
		  AND s.status = 'active'
	`
	h.pool.QueryRow(ctx, countQuery, args...).Scan(&totalBeforeScreening)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"date":        latestDate.Format("2006-01-02"),
			"market":      market,
			"count":       len(items),
			"totalBefore": totalBeforeScreening,
			"filteredOut": totalBeforeScreening - len(items),
			"passRate":    float64(len(items)) / float64(totalBeforeScreening) * 100,
			"hardCutConditions": map[string]string{
				"per": "> 0 AND <= 50 (고평가/적자 제외)",
				"pbr": ">= 0.2 (자산가치 필터)",
				"roe": ">= 5 (수익성 필터)",
			},
			"items": items,
		},
	})
}

// GetRanking returns ranked stocks with signal scores (S4)
// GET /api/v1/pipeline/ranking?market=KOSPI|KOSDAQ|ALL
func (h *PipelineHandler) GetRanking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	market := r.URL.Query().Get("market")

	// Get latest rank date
	var latestDate time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT MAX(rank_date) FROM selection.ranking_results
	`).Scan(&latestDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest rank date")
		respondError(w, http.StatusInternalServerError, "Failed to get latest rank date")
		return
	}

	// Build query with optional market filter
	marketFilter := ""
	args := []interface{}{latestDate}
	if market != "" && market != "ALL" {
		marketFilter = " AND s.market = $2"
		args = append(args, market)
	}

	// S4 Ranking: S3 Screener 통과 종목만 포함 (PER/PBR/ROE 조건)
	query := `
		WITH latest_fundamentals AS (
			SELECT DISTINCT ON (stock_code)
				stock_code, per, pbr, roe
			FROM data.fundamentals
			ORDER BY stock_code, report_date DESC
		),
		price_dates AS (
			SELECT DISTINCT trade_date FROM data.daily_prices
			ORDER BY trade_date DESC LIMIT 2
		),
		latest_prices AS (
			SELECT
				p1.stock_code,
				p1.close_price as current_price,
				COALESCE(p2.close_price, p1.close_price) as prev_price
			FROM data.daily_prices p1
			LEFT JOIN data.daily_prices p2
				ON p1.stock_code = p2.stock_code
				AND p2.trade_date = (SELECT MIN(trade_date) FROM price_dates)
			WHERE p1.trade_date = (SELECT MAX(trade_date) FROM price_dates)
		)
		SELECT
			r.stock_code,
			s.name as stock_name,
			s.market,
			r.rank,
			r.total_score::float8,
			COALESCE(r.momentum, 0)::float8,
			COALESCE(r.technical, 0)::float8,
			COALESCE(r.value, 0)::float8,
			COALESCE(r.quality, 0)::float8,
			COALESCE(r.flow, 0)::float8,
			COALESCE(r.event, 0)::float8,
			COALESCE(lp.current_price, 0)::float8 as current_price,
			CASE
				WHEN lp.prev_price > 0 THEN ((lp.current_price - lp.prev_price) / lp.prev_price * 100)::float8
				ELSE 0::float8
			END as change_rate
		FROM selection.ranking_results r
		JOIN data.stocks s ON r.stock_code = s.code
		JOIN latest_fundamentals lf ON r.stock_code = lf.stock_code
		LEFT JOIN latest_prices lp ON r.stock_code = lp.stock_code
		WHERE r.rank_date = $1
		  ` + marketFilter + `
		  AND s.status = 'active'
		  -- S3 Screener Hard Cut 조건 (재무 지표)
		  AND lf.per > 0 AND lf.per <= 50
		  AND lf.pbr >= 0.2
		  AND lf.roe >= 5
		ORDER BY r.rank
	`

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query ranking")
		respondError(w, http.StatusInternalServerError, "Query error")
		return
	}
	defer rows.Close()

	items := make([]RankedItem, 0)
	for rows.Next() {
		var item RankedItem
		err := rows.Scan(
			&item.StockCode,
			&item.StockName,
			&item.Market,
			&item.Rank,
			&item.TotalScore,
			&item.Momentum,
			&item.Technical,
			&item.Value,
			&item.Quality,
			&item.Flow,
			&item.Event,
			&item.CurrentPrice,
			&item.ChangeRate,
		)
		if err != nil {
			h.logger.WithError(err).Error("Failed to scan ranking item")
			continue
		}
		items = append(items, item)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"date":   latestDate.Format("2006-01-02"),
			"market": market,
			"count":  len(items),
			"items":  items,
		},
	})
}

// GetPortfolio returns target portfolio positions (S5)
// GET /api/v1/pipeline/portfolio
func (h *PipelineHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get latest target date
	var latestDate time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT MAX(target_date) FROM portfolio.target_positions
	`).Scan(&latestDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest target date")
		respondError(w, http.StatusInternalServerError, "Failed to get latest target date")
		return
	}

	query := `
		WITH latest_prices AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				close_price::float8 as close_price,
				LAG(close_price) OVER (PARTITION BY stock_code ORDER BY trade_date) as prev_close
			FROM data.daily_prices
			WHERE trade_date <= $1
			ORDER BY stock_code, trade_date DESC
		)
		SELECT
			p.stock_code,
			COALESCE(p.stock_name, s.name) as stock_name,
			s.market,
			p.weight::float8,
			p.target_qty,
			p.action,
			COALESCE(p.reason, '') as reason,
			COALESCE(lp.close_price, 0) as current_price,
			CASE
				WHEN lp.prev_close > 0 THEN ((lp.close_price - lp.prev_close) / lp.prev_close * 100)::float8
				ELSE 0::float8
			END as change_rate
		FROM portfolio.target_positions p
		JOIN data.stocks s ON p.stock_code = s.code
		LEFT JOIN latest_prices lp ON p.stock_code = lp.stock_code
		WHERE p.target_date = $1
		ORDER BY p.weight DESC
	`

	rows, err := h.pool.Query(ctx, query, latestDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query portfolio")
		respondError(w, http.StatusInternalServerError, "Query error")
		return
	}
	defer rows.Close()

	items := make([]PortfolioItem, 0)
	totalWeight := 0.0
	for rows.Next() {
		var item PortfolioItem
		err := rows.Scan(
			&item.StockCode,
			&item.StockName,
			&item.Market,
			&item.Weight,
			&item.TargetQty,
			&item.Action,
			&item.Reason,
			&item.CurrentPrice,
			&item.ChangeRate,
		)
		if err != nil {
			h.logger.WithError(err).Error("Failed to scan portfolio item")
			continue
		}
		totalWeight += item.Weight
		items = append(items, item)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"date":        latestDate.Format("2006-01-02"),
			"count":       len(items),
			"totalWeight": totalWeight,
			"cash":        1.0 - totalWeight,
			"positions":   items,
		},
	})
}
