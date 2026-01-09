package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/collector"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/quality"
	"github.com/wonny/aegis/v13/backend/internal/s1_universe"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// DataHandler handles data-related API endpoints
// ⭐ SSOT: 데이터 API 핸들러는 이 구조체에서만
type DataHandler struct {
	dataRepo     *s0_data.Repository
	universeRepo *s1_universe.Repository
	collector    *collector.Collector
	qualityGate  *quality.QualityGate
	logger       *logger.Logger
}

// NewDataHandler creates a new data handler
func NewDataHandler(
	dataRepo *s0_data.Repository,
	universeRepo *s1_universe.Repository,
	col *collector.Collector,
	qualityGate *quality.QualityGate,
	log *logger.Logger,
) *DataHandler {
	return &DataHandler{
		dataRepo:     dataRepo,
		universeRepo: universeRepo,
		collector:    col,
		qualityGate:  qualityGate,
		logger:       log,
	}
}

// GetQuality returns the latest data quality snapshot
// GET /api/data/quality
func (h *DataHandler) GetQuality(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get latest quality snapshot
	snapshot, err := h.dataRepo.GetLatestQualitySnapshot(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get quality snapshot")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve quality snapshot")
		return
	}

	respondJSON(w, http.StatusOK, snapshot)
}

// GetUniverse returns the latest universe
// GET /api/data/universe
func (h *DataHandler) GetUniverse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get latest universe
	universe, err := h.universeRepo.GetLatestUniverse(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get universe")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve universe")
		return
	}

	respondJSON(w, http.StatusOK, universe)
}

// CollectRequest represents a data collection request
type CollectRequest struct {
	Type string `json:"type"` // "all", "prices", "investor", "disclosure", "market_caps"
	From string `json:"from"` // Optional: date range start (YYYY-MM-DD)
	To   string `json:"to"`   // Optional: date range end (YYYY-MM-DD)
}

// CollectResponse represents a data collection response
type CollectResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Results interface{} `json:"results,omitempty"`
}

// Collect triggers data collection
// POST /api/data/collect
func (h *DataHandler) Collect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req CollectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Default type
	if req.Type == "" {
		req.Type = "all"
	}

	// Parse date range
	var from, to time.Time
	var err error

	if req.From != "" {
		from, err = time.Parse("2006-01-02", req.From)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid 'from' date format (expected YYYY-MM-DD)")
			return
		}
	} else {
		// Default: last 30 days
		from = time.Now().AddDate(0, 0, -30)
	}

	if req.To != "" {
		to, err = time.Parse("2006-01-02", req.To)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid 'to' date format (expected YYYY-MM-DD)")
			return
		}
	} else {
		to = time.Now()
	}

	h.logger.WithFields(map[string]interface{}{
		"type": req.Type,
		"from": from.Format("2006-01-02"),
		"to":   to.Format("2006-01-02"),
	}).Info("Data collection triggered")

	// Perform collection based on type
	cfg := collector.Config{Workers: 5}

	switch req.Type {
	case "prices":
		results, err := h.collector.FetchAllPrices(ctx, from, to, cfg)
		if err != nil {
			h.logger.WithError(err).Error("Failed to collect prices")
			respondError(w, http.StatusInternalServerError, "Failed to collect prices")
			return
		}
		respondJSON(w, http.StatusOK, CollectResponse{
			Status:  "success",
			Message: "Price data collected",
			Type:    req.Type,
			Results: results,
		})

	case "investor":
		results, err := h.collector.FetchAllInvestorFlow(ctx, from, to, cfg)
		if err != nil {
			h.logger.WithError(err).Error("Failed to collect investor flow")
			respondError(w, http.StatusInternalServerError, "Failed to collect investor flow")
			return
		}
		respondJSON(w, http.StatusOK, CollectResponse{
			Status:  "success",
			Message: "Investor flow data collected",
			Type:    req.Type,
			Results: results,
		})

	case "disclosure":
		if err := h.collector.FetchDisclosures(ctx, from, to); err != nil {
			h.logger.WithError(err).Error("Failed to collect disclosures")
			respondError(w, http.StatusInternalServerError, "Failed to collect disclosures")
			return
		}
		respondJSON(w, http.StatusOK, CollectResponse{
			Status:  "success",
			Message: "Disclosure data collected",
			Type:    req.Type,
		})

	case "market_caps":
		if err := h.collector.FetchMarketCaps(ctx); err != nil {
			h.logger.WithError(err).Error("Failed to collect market caps")
			respondError(w, http.StatusInternalServerError, "Failed to collect market caps")
			return
		}
		respondJSON(w, http.StatusOK, CollectResponse{
			Status:  "success",
			Message: "Market cap data collected",
			Type:    req.Type,
		})

	case "all":
		// Collect all data
		if err := h.collector.FetchAll(ctx, from, to, cfg); err != nil {
			h.logger.WithError(err).Error("Failed to collect all data")
			respondError(w, http.StatusInternalServerError, "Failed to collect all data")
			return
		}

		// Also collect market caps and disclosures
		if err := h.collector.FetchMarketCaps(ctx); err != nil {
			h.logger.WithError(err).Warn("Failed to collect market caps during 'all'")
		}

		dartFrom := to.AddDate(0, 0, -7)
		if err := h.collector.FetchDisclosures(ctx, dartFrom, to); err != nil {
			h.logger.WithError(err).Warn("Failed to collect disclosures during 'all'")
		}

		respondJSON(w, http.StatusOK, CollectResponse{
			Status:  "success",
			Message: "All data collected",
			Type:    req.Type,
		})

	default:
		respondError(w, http.StatusBadRequest, "Invalid collection type (valid: all, prices, investor, disclosure, market_caps)")
		return
	}
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
