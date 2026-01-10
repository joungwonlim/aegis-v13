package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/wonny/aegis/v13/backend/internal/external/kis"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// TradingHandler handles trading-related API endpoints
// ⭐ SSOT: 거래 API 핸들러는 이 구조체에서만
type TradingHandler struct {
	kisClient   *kis.Client
	kisWSClient *kis.WSClient
	logger      *logger.Logger
}

// NewTradingHandler creates a new trading handler
func NewTradingHandler(kisClient *kis.Client, kisWSClient *kis.WSClient, log *logger.Logger) *TradingHandler {
	return &TradingHandler{
		kisClient:   kisClient,
		kisWSClient: kisWSClient,
		logger:      log,
	}
}

// ============================================================
// Balance & Positions
// ============================================================

// GetBalance returns account balance and positions
// GET /api/trading/balance
func (h *TradingHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	balance, positions, err := h.kisClient.GetBalance(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get balance")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve balance")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"balance":   balance,
		"positions": positions,
	})
}

// GetPositions returns only positions
// GET /api/trading/positions
func (h *TradingHandler) GetPositions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	positions, err := h.kisClient.GetPositions(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get positions")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve positions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"positions": positions,
		"count":     len(positions),
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
// GET /api/trading/price/{stock_code}
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
