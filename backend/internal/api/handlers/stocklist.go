package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/wonny/aegis/v13/backend/internal/portfolio"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// StocklistHandler handles stocklist-related API endpoints
// SSOT: Stocklist (Watchlist) API 핸들러는 이 구조체에서만
type StocklistHandler struct {
	portfolioRepo *portfolio.Repository
	logger        *logger.Logger
}

// NewStocklistHandler creates a new stocklist handler
func NewStocklistHandler(
	portfolioRepo *portfolio.Repository,
	log *logger.Logger,
) *StocklistHandler {
	return &StocklistHandler{
		portfolioRepo: portfolioRepo,
		logger:        log,
	}
}

// WatchlistResponse represents the watchlist list response
type WatchlistResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Watch     []portfolio.WatchlistItem `json:"watch"`
		Candidate []portfolio.WatchlistItem `json:"candidate"`
	} `json:"data"`
}

// GetWatchlist returns all watchlist items grouped by category
// GET /api/v1/watchlist
func (h *StocklistHandler) GetWatchlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, err := h.portfolioRepo.GetWatchlistAll(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get watchlist")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve watchlist")
		return
	}

	// Group by category
	response := WatchlistResponse{Success: true}
	response.Data.Watch = make([]portfolio.WatchlistItem, 0)
	response.Data.Candidate = make([]portfolio.WatchlistItem, 0)

	for _, item := range items {
		if item.Category == "watch" {
			response.Data.Watch = append(response.Data.Watch, item)
		} else if item.Category == "candidate" {
			response.Data.Candidate = append(response.Data.Candidate, item)
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// GetWatchlistByCategory returns watchlist items by category
// GET /api/v1/watchlist/watch or GET /api/v1/watchlist/candidate
func (h *StocklistHandler) GetWatchlistByCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	category := vars["category"]

	if category != "watch" && category != "candidate" {
		respondError(w, http.StatusBadRequest, "Invalid category (must be 'watch' or 'candidate')")
		return
	}

	items, err := h.portfolioRepo.GetWatchlistByCategory(ctx, category)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get watchlist by category")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve watchlist")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    items,
	})
}

// CreateWatchlistRequest represents a watchlist create request
type CreateWatchlistRequest struct {
	StockCode    string `json:"stock_code"`
	Category     string `json:"category"`
	AlertEnabled bool   `json:"alert_enabled"`
}

// CreateWatchlistItem creates a new watchlist entry
// POST /api/v1/watchlist
func (h *StocklistHandler) CreateWatchlistItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateWatchlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validation
	if req.StockCode == "" {
		respondError(w, http.StatusBadRequest, "stock_code is required")
		return
	}
	if req.Category == "" {
		req.Category = "watch" // default
	}
	if req.Category != "watch" && req.Category != "candidate" {
		respondError(w, http.StatusBadRequest, "Invalid category (must be 'watch' or 'candidate')")
		return
	}

	item, err := h.portfolioRepo.CreateWatchlistItem(ctx, req.StockCode, req.Category, req.AlertEnabled)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"stock_code": req.StockCode,
		}).Error("Failed to create watchlist item")
		respondError(w, http.StatusInternalServerError, "Failed to create watchlist item")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    item,
	})
}

// UpdateWatchlistRequest represents a watchlist update request
type UpdateWatchlistRequest struct {
	Category     *string `json:"category,omitempty"`
	AlertEnabled *bool   `json:"alert_enabled,omitempty"`
}

// UpdateWatchlistItem updates a watchlist entry
// PUT /api/v1/watchlist/{id}
func (h *StocklistHandler) UpdateWatchlistItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	var req UpdateWatchlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate category if provided
	if req.Category != nil && *req.Category != "watch" && *req.Category != "candidate" {
		respondError(w, http.StatusBadRequest, "Invalid category (must be 'watch' or 'candidate')")
		return
	}

	item, err := h.portfolioRepo.UpdateWatchlistItem(ctx, id, req.Category, req.AlertEnabled)
	if err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"id": id,
		}).Error("Failed to update watchlist item")
		respondError(w, http.StatusInternalServerError, "Failed to update watchlist item")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    item,
	})
}

// DeleteWatchlistItem deletes a watchlist entry
// DELETE /api/v1/watchlist/{id}
func (h *StocklistHandler) DeleteWatchlistItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if err := h.portfolioRepo.DeleteWatchlistItem(ctx, id); err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"id": id,
		}).Error("Failed to delete watchlist item")
		respondError(w, http.StatusInternalServerError, "Failed to delete watchlist item")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}
