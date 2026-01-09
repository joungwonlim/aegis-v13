package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/wonny/aegis/v13/backend/internal/realtime"
	"github.com/wonny/aegis/v13/backend/internal/realtime/cache"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

const (
	// MaxWebSocketSymbols is the maximum number of symbols that can be subscribed via WebSocket
	MaxWebSocketSymbols = 40

	// Reconnect settings
	reconnectDelay    = 5 * time.Second
	maxReconnectDelay = 5 * time.Minute

	// Ping/Pong settings
	pingInterval = 30 * time.Second
	pongWait     = 60 * time.Second
	writeWait    = 10 * time.Second
)

// KISWebSocketClient manages real-time price data via KIS WebSocket
// ⭐ SSOT: KIS WebSocket 연결 및 40개 심볼 관리는 이 클라이언트에서만
type KISWebSocketClient struct {
	config        *config.Config
	logger        *logger.Logger
	cache         *cache.PriceCache
	priorityQueue *PriorityQueue

	conn          *websocket.Conn
	connMu        sync.RWMutex

	activeSymbols map[string]bool
	symbolsMu     sync.RWMutex

	stopCh        chan struct{}
	doneCh        chan struct{}
	reconnecting  bool
	reconnectMu   sync.Mutex
}

// NewKISWebSocketClient creates a new KIS WebSocket client
func NewKISWebSocketClient(cfg *config.Config, log *logger.Logger, priceCache *cache.PriceCache) *KISWebSocketClient {
	return &KISWebSocketClient{
		config:        cfg,
		logger:        log,
		cache:         priceCache,
		priorityQueue: NewPriorityQueue(),
		activeSymbols: make(map[string]bool),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// Start starts the WebSocket client
func (c *KISWebSocketClient) Start(ctx context.Context) error {
	c.logger.Info("Starting KIS WebSocket client")

	if err := c.connect(ctx); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	go c.readLoop(ctx)
	go c.pingLoop(ctx)
	go c.priorityUpdateLoop(ctx)

	return nil
}

// Stop stops the WebSocket client
func (c *KISWebSocketClient) Stop() {
	c.logger.Info("Stopping KIS WebSocket client")

	close(c.stopCh)

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connMu.Unlock()

	<-c.doneCh
}

// UpdatePriority updates the priority of a symbol
func (c *KISWebSocketClient) UpdatePriority(priority *realtime.SymbolPriority) {
	c.priorityQueue.Update(priority)
	c.rebalanceSymbols()
}

// RemoveSymbol removes a symbol from tracking
func (c *KISWebSocketClient) RemoveSymbol(code string) {
	c.priorityQueue.Remove(code)
	c.rebalanceSymbols()
}

// connect establishes WebSocket connection
func (c *KISWebSocketClient) connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	// Build WebSocket URL
	wsURL := c.buildWebSocketURL()

	c.logger.WithField("url", wsURL).Debug("Connecting to KIS WebSocket")

	// Create WebSocket connection
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.conn = conn

	// Set read deadline
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	c.logger.Info("Connected to KIS WebSocket")

	// Subscribe to initial symbols
	c.subscribeTop40()

	return nil
}

// buildWebSocketURL builds the KIS WebSocket URL with authentication
func (c *KISWebSocketClient) buildWebSocketURL() string {
	// KIS WebSocket endpoint (example - adjust based on actual KIS API)
	// Real implementation would need proper authentication token
	return fmt.Sprintf("wss://ops.koreainvestment.com:21000/tryitout/%s", c.config.KIS.AppKey)
}

// readLoop reads messages from WebSocket
func (c *KISWebSocketClient) readLoop(ctx context.Context) {
	defer close(c.doneCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.logger.WithError(err).Error("Failed to read message")
			c.handleDisconnect(ctx)
			continue
		}

		if err := c.handleMessage(message); err != nil {
			c.logger.WithError(err).Error("Failed to handle message")
		}
	}
}

// handleMessage processes a WebSocket message
func (c *KISWebSocketClient) handleMessage(message []byte) error {
	// Parse KIS WebSocket message format
	var msg KISMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	// Convert to PriceTick
	tick, err := c.convertToPriceTick(&msg)
	if err != nil {
		return fmt.Errorf("convert to price tick: %w", err)
	}

	// Update cache
	if c.cache.Update(tick) {
		c.logger.WithFields(map[string]interface{}{
			"code":  tick.Code,
			"price": tick.Price,
		}).Debug("Updated price from WebSocket")
	}

	return nil
}

// handleDisconnect handles WebSocket disconnection and reconnects
func (c *KISWebSocketClient) handleDisconnect(ctx context.Context) {
	c.reconnectMu.Lock()
	if c.reconnecting {
		c.reconnectMu.Unlock()
		return
	}
	c.reconnecting = true
	c.reconnectMu.Unlock()

	defer func() {
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
	}()

	c.logger.Warn("WebSocket disconnected, attempting to reconnect")

	delay := reconnectDelay
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-time.After(delay):
		}

		if err := c.connect(ctx); err != nil {
			c.logger.WithError(err).WithField("delay", delay).Error("Reconnect failed, retrying")

			// Exponential backoff
			delay *= 2
			if delay > maxReconnectDelay {
				delay = maxReconnectDelay
			}
			continue
		}

		c.logger.Info("Reconnected to KIS WebSocket")
		return
	}
}

// pingLoop sends periodic pings to keep connection alive
func (c *KISWebSocketClient) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				continue
			}

			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				c.logger.WithError(err).Error("Failed to send ping")
			}
		}
	}
}

// priorityUpdateLoop periodically rebalances symbols based on priority
func (c *KISWebSocketClient) priorityUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.rebalanceSymbols()
		}
	}
}

// rebalanceSymbols rebalances the 40 WebSocket symbols based on priority
func (c *KISWebSocketClient) rebalanceSymbols() {
	top40 := c.priorityQueue.GetTop(MaxWebSocketSymbols)

	// Determine which symbols to add/remove
	newSymbols := make(map[string]bool)
	for _, priority := range top40 {
		newSymbols[priority.Code] = true
	}

	c.symbolsMu.Lock()

	// Find symbols to remove
	var toRemove []string
	for code := range c.activeSymbols {
		if !newSymbols[code] {
			toRemove = append(toRemove, code)
		}
	}

	// Find symbols to add
	var toAdd []string
	for code := range newSymbols {
		if !c.activeSymbols[code] {
			toAdd = append(toAdd, code)
		}
	}

	c.symbolsMu.Unlock()

	// Apply changes
	if len(toRemove) > 0 {
		c.unsubscribeSymbols(toRemove)
	}

	if len(toAdd) > 0 {
		c.subscribeSymbols(toAdd)
	}

	if len(toAdd) > 0 || len(toRemove) > 0 {
		c.logger.WithFields(map[string]interface{}{
			"added":   len(toAdd),
			"removed": len(toRemove),
			"total":   len(newSymbols),
		}).Info("Rebalanced WebSocket symbols")
	}
}

// subscribeTop40 subscribes to top 40 priority symbols
func (c *KISWebSocketClient) subscribeTop40() {
	top40 := c.priorityQueue.GetTop(MaxWebSocketSymbols)
	codes := make([]string, len(top40))
	for i, priority := range top40 {
		codes[i] = priority.Code
	}
	c.subscribeSymbols(codes)
}

// subscribeSymbols subscribes to symbols
func (c *KISWebSocketClient) subscribeSymbols(codes []string) {
	if len(codes) == 0 {
		return
	}

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return
	}

	for _, code := range codes {
		msg := c.buildSubscribeMessage(code)

		if err := conn.WriteJSON(msg); err != nil {
			c.logger.WithError(err).WithField("code", code).Error("Failed to subscribe")
			continue
		}

		c.symbolsMu.Lock()
		c.activeSymbols[code] = true
		c.symbolsMu.Unlock()

		c.logger.WithField("code", code).Debug("Subscribed to symbol")
	}
}

// unsubscribeSymbols unsubscribes from symbols
func (c *KISWebSocketClient) unsubscribeSymbols(codes []string) {
	if len(codes) == 0 {
		return
	}

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return
	}

	for _, code := range codes {
		msg := c.buildUnsubscribeMessage(code)

		if err := conn.WriteJSON(msg); err != nil {
			c.logger.WithError(err).WithField("code", code).Error("Failed to unsubscribe")
			continue
		}

		c.symbolsMu.Lock()
		delete(c.activeSymbols, code)
		c.symbolsMu.Unlock()

		c.logger.WithField("code", code).Debug("Unsubscribed from symbol")
	}
}

// buildSubscribeMessage builds a subscribe message for KIS WebSocket
func (c *KISWebSocketClient) buildSubscribeMessage(code string) map[string]interface{} {
	// KIS WebSocket subscribe message format (example - adjust based on actual API)
	return map[string]interface{}{
		"header": map[string]interface{}{
			"approval_key": c.config.KIS.AppKey,
			"custtype":     "P",
			"tr_type":      "1", // Subscribe
			"content-type": "utf-8",
		},
		"body": map[string]interface{}{
			"input": map[string]interface{}{
				"tr_id":   "H0STCNT0",  // Real-time stock price
				"tr_key":  code,
			},
		},
	}
}

// buildUnsubscribeMessage builds an unsubscribe message for KIS WebSocket
func (c *KISWebSocketClient) buildUnsubscribeMessage(code string) map[string]interface{} {
	// KIS WebSocket unsubscribe message format
	return map[string]interface{}{
		"header": map[string]interface{}{
			"approval_key": c.config.KIS.AppKey,
			"custtype":     "P",
			"tr_type":      "2", // Unsubscribe
			"content-type": "utf-8",
		},
		"body": map[string]interface{}{
			"input": map[string]interface{}{
				"tr_id":   "H0STCNT0",
				"tr_key":  code,
			},
		},
	}
}

// convertToPriceTick converts KIS message to PriceTick
func (c *KISWebSocketClient) convertToPriceTick(msg *KISMessage) (*realtime.PriceTick, error) {
	// Parse KIS message format and convert to PriceTick
	// This is a simplified example - actual implementation depends on KIS API format

	return &realtime.PriceTick{
		Code:      msg.Code,
		Price:     msg.CurrentPrice,
		Change:    msg.Change,
		Volume:    msg.Volume,
		Timestamp: time.Now(),
		Source:    string(realtime.SourceKISWebSocket),
		IsStale:   false,
	}, nil
}

// GetActiveSymbols returns the currently subscribed symbols
func (c *KISWebSocketClient) GetActiveSymbols() []string {
	c.symbolsMu.RLock()
	defer c.symbolsMu.RUnlock()

	codes := make([]string, 0, len(c.activeSymbols))
	for code := range c.activeSymbols {
		codes = append(codes, code)
	}
	return codes
}

// KISMessage represents a message from KIS WebSocket
type KISMessage struct {
	Code         string `json:"code"`
	CurrentPrice int64  `json:"current_price"`
	Change       int64  `json:"change"`
	Volume       int64  `json:"volume"`
	// Add more fields as needed based on actual KIS API format
}
