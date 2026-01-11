package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// StockHandler handles stock data API endpoints
// ⭐ SSOT: 종목 데이터 API 핸들러는 이 구조체에서만
type StockHandler struct {
	priceRepo        *s0_data.PriceRepository
	investorFlowRepo *s0_data.InvestorFlowRepository
	dataRepo         *s0_data.Repository
	logger           *logger.Logger
}

// NewStockHandler creates a new stock handler
func NewStockHandler(priceRepo *s0_data.PriceRepository, investorFlowRepo *s0_data.InvestorFlowRepository, dataRepo *s0_data.Repository, log *logger.Logger) *StockHandler {
	return &StockHandler{
		priceRepo:        priceRepo,
		investorFlowRepo: investorFlowRepo,
		dataRepo:         dataRepo,
		logger:           log,
	}
}

// DailyPriceResponse represents a daily price record for API response
type DailyPriceResponse struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// GetDailyPrices returns daily price data for a stock
// GET /api/stocks/{code}/daily?days=365
func (h *StockHandler) GetDailyPrices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	code := vars["code"]

	if code == "" {
		respondError(w, http.StatusBadRequest, "stock code is required")
		return
	}

	// Parse days parameter (default: 365)
	days := 365
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	// Calculate date range
	to := time.Now()
	from := to.AddDate(0, 0, -days)

	// Get prices from repository
	prices, err := h.priceRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"code": code,
			"days": days,
		}).Error("Failed to get daily prices")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve daily prices")
		return
	}

	// Convert to response format
	result := make([]DailyPriceResponse, len(prices))
	for i, p := range prices {
		result[i] = DailyPriceResponse{
			Date:   p.Date.Format("2006-01-02"),
			Open:   float64(p.Open),
			High:   float64(p.High),
			Low:    float64(p.Low),
			Close:  float64(p.Close),
			Volume: p.Volume,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// InvestorTradingResponse represents investor trading data for API response
// Matches frontend InvestorTrading type
type InvestorTradingResponse struct {
	Date        string  `json:"date"`
	ClosePrice  int64   `json:"close_price"`
	PriceChange int64   `json:"price_change"`
	ChangeRate  float64 `json:"change_rate"`
	Volume      int64   `json:"volume"`
	ForeignNet  int64   `json:"foreign_net"`
	InstNet     int64   `json:"inst_net"`
	IndivNet    int64   `json:"indiv_net"`
}

// GetInvestorTrading returns investor trading data for a stock
// GET /api/stocks/{code}/investor-trading?days=365
func (h *StockHandler) GetInvestorTrading(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	code := vars["code"]

	if code == "" {
		respondError(w, http.StatusBadRequest, "stock code is required")
		return
	}

	// Parse days parameter (default: 365)
	days := 365
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	// Calculate date range
	to := time.Now()
	from := to.AddDate(0, 0, -days)

	// Get investor flows from repository
	flows, err := h.investorFlowRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"code": code,
			"days": days,
		}).Error("Failed to get investor trading")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve investor trading")
		return
	}

	// Get prices for the same date range
	prices, err := h.priceRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get prices for investor trading")
		// Continue without price data
	}

	// Create price map for quick lookup
	priceMap := make(map[string]*struct {
		Close      int64
		PrevClose  int64
		Volume     int64
		ChangeRate float64
	})
	for i, p := range prices {
		dateStr := p.Date.Format("2006-01-02")
		entry := &struct {
			Close      int64
			PrevClose  int64
			Volume     int64
			ChangeRate float64
		}{
			Close:  p.Close,
			Volume: p.Volume,
		}
		// Calculate prev close and change rate
		if i > 0 {
			entry.PrevClose = prices[i-1].Close
			if entry.PrevClose > 0 {
				entry.ChangeRate = float64(p.Close-entry.PrevClose) / float64(entry.PrevClose) * 100
			}
		}
		priceMap[dateStr] = entry
	}

	// Convert to response format, merging with price data
	result := make([]InvestorTradingResponse, len(flows))
	for i, f := range flows {
		dateStr := f.Date.Format("2006-01-02")
		resp := InvestorTradingResponse{
			Date:       dateStr,
			ForeignNet: f.ForeignNet,
			InstNet:    f.InstitutionNet,
			IndivNet:   f.IndividualNet,
		}

		// Add price data if available
		if priceData, ok := priceMap[dateStr]; ok {
			resp.ClosePrice = priceData.Close
			resp.PriceChange = priceData.Close - priceData.PrevClose
			resp.ChangeRate = priceData.ChangeRate
			resp.Volume = priceData.Volume
		}

		result[i] = resp
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// GetStockDescription returns stock description (기업개요)
// GET /api/stocks/{code}/description
func (h *StockHandler) GetStockDescription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	code := vars["code"]

	if code == "" {
		respondError(w, http.StatusBadRequest, "stock code is required")
		return
	}

	// Get description from repository
	description, err := h.dataRepo.GetStockDescription(ctx, code)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"code": code,
		}).Error("Failed to get stock description")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve stock description")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"description": description,
	})
}
