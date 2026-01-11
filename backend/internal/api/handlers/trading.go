package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/wonny/aegis/v13/backend/internal/external/kis"
	"github.com/wonny/aegis/v13/backend/internal/portfolio"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// TradingHandler handles trading-related API endpoints
// ⭐ SSOT: 거래 API 핸들러는 이 구조체에서만
type TradingHandler struct {
	kisClient     *kis.Client
	kisWSClient   *kis.WSClient
	portfolioRepo *portfolio.Repository
	logger        *logger.Logger
}

// NewTradingHandler creates a new trading handler
func NewTradingHandler(kisClient *kis.Client, kisWSClient *kis.WSClient, portfolioRepo *portfolio.Repository, log *logger.Logger) *TradingHandler {
	return &TradingHandler{
		kisClient:     kisClient,
		kisWSClient:   kisWSClient,
		portfolioRepo: portfolioRepo,
		logger:        log,
	}
}

// ============================================================
// Balance & Positions
// ============================================================

// GetBalance returns account balance and positions with exit monitoring status
// GET /api/trading/balance
func (h *TradingHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	balance, positions, err := h.kisClient.GetBalance(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get balance")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve balance")
		return
	}

	// Get exit monitoring statuses from DB
	monitoringStatuses, err := h.portfolioRepo.GetExitMonitoringAll(ctx)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get exit monitoring statuses")
		// Continue without monitoring status
	}

	// Create map for quick lookup
	monitoringMap := make(map[string]bool)
	for _, status := range monitoringStatuses {
		monitoringMap[status.StockCode] = status.Enabled
	}

	// Get market info for all positions (single IN query - efficient)
	stockCodes := make([]string, len(positions))
	for i, pos := range positions {
		stockCodes[i] = pos.StockCode
	}
	marketMap, err := h.portfolioRepo.GetStockMarkets(ctx, stockCodes)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get stock markets")
		marketMap = make(map[string]string)
	}

	// Merge positions with monitoring status and market
	result := make([]PositionWithMonitoring, len(positions))
	for i, pos := range positions {
		result[i] = PositionWithMonitoring{
			Position:              pos,
			Market:                marketMap[pos.StockCode],
			ExitMonitoringEnabled: monitoringMap[pos.StockCode],
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"balance":   balance,
		"positions": result,
	})
}

// PositionWithMonitoring extends KIS position with exit monitoring status and market info
type PositionWithMonitoring struct {
	kis.Position
	Market                string `json:"market"`
	ExitMonitoringEnabled bool   `json:"exit_monitoring_enabled"`
}

// GetPositions returns only positions with exit monitoring status
// GET /api/trading/positions
func (h *TradingHandler) GetPositions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get positions from KIS
	positions, err := h.kisClient.GetPositions(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get positions")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve positions")
		return
	}

	// Get exit monitoring statuses from DB
	monitoringStatuses, err := h.portfolioRepo.GetExitMonitoringAll(ctx)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get exit monitoring statuses")
		// Continue without monitoring status
	}

	// Create map for quick lookup
	monitoringMap := make(map[string]bool)
	for _, status := range monitoringStatuses {
		monitoringMap[status.StockCode] = status.Enabled
	}

	// Get market info for all positions (single IN query - efficient)
	stockCodes := make([]string, len(positions))
	for i, pos := range positions {
		stockCodes[i] = pos.StockCode
	}
	marketMap, err := h.portfolioRepo.GetStockMarkets(ctx, stockCodes)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get stock markets")
		marketMap = make(map[string]string)
	}

	// Merge positions with monitoring status and market
	result := make([]PositionWithMonitoring, len(positions))
	for i, pos := range positions {
		result[i] = PositionWithMonitoring{
			Position:              pos,
			Market:                marketMap[pos.StockCode],
			ExitMonitoringEnabled: monitoringMap[pos.StockCode],
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"positions": result,
		"count":     len(result),
	})
}

// ============================================================
// Orders
// ============================================================

// GetOrders returns orders within date range
// GET /api/trading/orders?start_date=20240101&end_date=20240131
func (h *TradingHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	orders, err := h.kisClient.GetOrders(ctx, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get orders")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve orders")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"count":  len(orders),
	})
}

// GetUnfilledOrders returns pending/partial orders
// GET /api/trading/orders/unfilled
func (h *TradingHandler) GetUnfilledOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orders, err := h.kisClient.GetUnfilledOrders(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get unfilled orders")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve unfilled orders")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"count":  len(orders),
	})
}

// GetFilledOrders returns filled orders
// GET /api/trading/orders/filled
func (h *TradingHandler) GetFilledOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orders, err := h.kisClient.GetFilledOrders(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get filled orders")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve filled orders")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"count":  len(orders),
	})
}

// ============================================================
// Place & Cancel Orders
// ============================================================

// PlaceOrderRequest represents an order placement request
type PlaceOrderRequest struct {
	StockCode string `json:"stock_code"`
	Side      string `json:"side"`     // "buy" or "sell"
	Type      string `json:"type"`     // "limit" or "market"
	Quantity  int64  `json:"quantity"`
	Price     int64  `json:"price"`    // 0 for market orders
}

// PlaceOrder places a new order
// POST /api/trading/orders
func (h *TradingHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.StockCode == "" {
		respondError(w, http.StatusBadRequest, "stock_code is required")
		return
	}
	if req.Side != "buy" && req.Side != "sell" {
		respondError(w, http.StatusBadRequest, "side must be 'buy' or 'sell'")
		return
	}
	if req.Type != "limit" && req.Type != "market" {
		respondError(w, http.StatusBadRequest, "type must be 'limit' or 'market'")
		return
	}
	if req.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}
	if req.Type == "limit" && req.Price <= 0 {
		respondError(w, http.StatusBadRequest, "price is required for limit orders")
		return
	}

	// Convert to KIS types
	orderSide := kis.OrderSideBuy
	if req.Side == "sell" {
		orderSide = kis.OrderSideSell
	}

	orderType := kis.OrderTypeLimit
	if req.Type == "market" {
		orderType = kis.OrderTypeMarket
	}

	// Place order
	result, err := h.kisClient.PlaceOrder(ctx, kis.PlaceOrderRequest{
		StockCode: req.StockCode,
		Side:      orderSide,
		Type:      orderType,
		Quantity:  req.Quantity,
		Price:     req.Price,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to place order")
		respondError(w, http.StatusInternalServerError, "Failed to place order")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"stock_code": req.StockCode,
		"side":       req.Side,
		"quantity":   req.Quantity,
		"order_no":   result.OrderNo,
	}).Info("Order placed")

	respondJSON(w, http.StatusOK, result)
}

// CancelOrderRequest represents an order cancellation request
type CancelOrderRequest struct {
	OrderNo string `json:"order_no"`
}

// CancelOrder cancels an existing order
// DELETE /api/trading/orders/{order_no}
func (h *TradingHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get order_no from URL path
	orderNo := r.URL.Query().Get("order_no")
	if orderNo == "" {
		respondError(w, http.StatusBadRequest, "order_no is required")
		return
	}

	result, err := h.kisClient.CancelOrder(ctx, orderNo)
	if err != nil {
		h.logger.WithError(err).Error("Failed to cancel order")
		respondError(w, http.StatusInternalServerError, "Failed to cancel order")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"order_no": orderNo,
	}).Info("Order cancelled")

	respondJSON(w, http.StatusOK, result)
}

// ============================================================
// Price Queries
// ============================================================

// GetCurrentPrice returns current price for a stock
// GET /api/trading/price?stock_code=005930
func (h *TradingHandler) GetCurrentPrice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stockCode := r.URL.Query().Get("stock_code")
	if stockCode == "" {
		respondError(w, http.StatusBadRequest, "stock_code is required")
		return
	}

	price, err := h.kisClient.GetCurrentPrice(ctx, stockCode)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get current price")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve current price")
		return
	}

	respondJSON(w, http.StatusOK, price)
}

// RealtimePrice represents a simplified price response for frontend
type RealtimePrice struct {
	Price      float64 `json:"price"`
	Change     float64 `json:"change"`
	ChangeRate float64 `json:"change_rate"`
	Volume     int64   `json:"volume,omitempty"`
	UpdatedAt  string  `json:"updated_at,omitempty"`
}

// GetPrices returns current prices for multiple stocks
// GET /api/trading/prices?symbols=005930,073570,035720
func (h *TradingHandler) GetPrices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	symbolsStr := r.URL.Query().Get("symbols")
	if symbolsStr == "" {
		respondError(w, http.StatusBadRequest, "symbols is required")
		return
	}

	symbols := strings.Split(symbolsStr, ",")
	if len(symbols) == 0 {
		respondError(w, http.StatusBadRequest, "at least one symbol is required")
		return
	}

	// 최대 50개 종목까지 허용
	if len(symbols) > 50 {
		symbols = symbols[:50]
	}

	prices := make(map[string]RealtimePrice)

	for i, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		if symbol == "" {
			continue
		}

		// KIS API Rate Limit: 초당 20건 → 50ms 딜레이로 안전하게 처리
		if i > 0 {
			time.Sleep(50 * time.Millisecond)
		}

		price, err := h.kisClient.GetCurrentPrice(ctx, symbol)
		if err != nil {
			h.logger.WithError(err).WithFields(map[string]interface{}{
				"symbol": symbol,
			}).Warn("Failed to get price for symbol")
			continue
		}

		prices[symbol] = RealtimePrice{
			Price:      price.ClosePrice,
			Change:     price.Change,
			ChangeRate: price.ChangeRate,
			Volume:     price.Volume,
			UpdatedAt:  price.FetchedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"prices": prices,
	})
}

// ============================================================
// WebSocket Status
// ============================================================

// GetWebSocketStatus returns WebSocket connection status
// GET /api/trading/ws/status
func (h *TradingHandler) GetWebSocketStatus(w http.ResponseWriter, r *http.Request) {
	if h.kisWSClient == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connected":     false,
			"subscriptions": []string{},
			"message":       "WebSocket client not initialized",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"connected":     h.kisWSClient.IsConnected(),
		"subscriptions": h.kisWSClient.GetSubscriptions(),
		"count":         h.kisWSClient.SubscriptionCount(),
	})
}

// SubscribeRequest represents a WebSocket subscription request
type SubscribeRequest struct {
	Symbols []string `json:"symbols"`
}

// Subscribe subscribes to real-time tick data
// POST /api/trading/ws/subscribe
func (h *TradingHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if h.kisWSClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WebSocket client not initialized")
		return
	}

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Symbols) == 0 {
		respondError(w, http.StatusBadRequest, "symbols is required")
		return
	}

	if err := h.kisWSClient.Subscribe(req.Symbols...); err != nil {
		h.logger.WithError(err).Error("Failed to subscribe")
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "subscribed",
		"symbols":       req.Symbols,
		"subscriptions": h.kisWSClient.GetSubscriptions(),
	})
}

// Unsubscribe unsubscribes from real-time tick data
// POST /api/trading/ws/unsubscribe
func (h *TradingHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	if h.kisWSClient == nil {
		respondError(w, http.StatusServiceUnavailable, "WebSocket client not initialized")
		return
	}

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Symbols) == 0 {
		respondError(w, http.StatusBadRequest, "symbols is required")
		return
	}

	if err := h.kisWSClient.Unsubscribe(req.Symbols...); err != nil {
		h.logger.WithError(err).Error("Failed to unsubscribe")
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "unsubscribed",
		"symbols":       req.Symbols,
		"subscriptions": h.kisWSClient.GetSubscriptions(),
	})
}

// ============================================================
// Exit Monitoring
// ============================================================

// UpdateExitMonitoringRequest represents exit monitoring update request
type UpdateExitMonitoringRequest struct {
	Enabled bool `json:"enabled"`
}

// UpdateExitMonitoring updates exit monitoring status for a position
// PATCH /api/trading/positions/{stock_code}/exit-monitoring
func (h *TradingHandler) UpdateExitMonitoring(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	stockCode := vars["stock_code"]

	if stockCode == "" {
		respondError(w, http.StatusBadRequest, "stock_code is required")
		return
	}

	var req UpdateExitMonitoringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update exit monitoring status
	if err := h.portfolioRepo.SetExitMonitoring(ctx, stockCode, req.Enabled); err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"stock_code": stockCode,
			"enabled":    req.Enabled,
		}).Error("Failed to update exit monitoring")
		respondError(w, http.StatusInternalServerError, "Failed to update exit monitoring")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"stock_code": stockCode,
		"enabled":    req.Enabled,
	}).Info("Exit monitoring updated")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"stock_code": stockCode,
		"enabled":    req.Enabled,
	})
}

// GetExitMonitoringStatus returns exit monitoring status for all positions
// GET /api/trading/exit-monitoring
func (h *TradingHandler) GetExitMonitoringStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	statuses, err := h.portfolioRepo.GetExitMonitoringAll(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get exit monitoring status")
		respondError(w, http.StatusInternalServerError, "Failed to get exit monitoring status")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"statuses": statuses,
		"count":    len(statuses),
	})
}
