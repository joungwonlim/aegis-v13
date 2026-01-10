package kis

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// WebSocket URLs
const (
	WSURLReal = "ws://ops.koreainvestment.com:21000/"
	WSURLDemo = "ws://ops.koreainvestment.com:31000/"

	// TR IDs
	TRIDTickReal      = "H0STCNT0" // 실시간 체결가
	TRIDExecutionReal = "H0STCNI0" // 실전 체결통보
	TRIDExecutionDemo = "H0STCNI9" // 모의 체결통보

	// Limits
	MaxSubscriptionsPerSession = 41

	// Timing
	PingInterval         = 30 * time.Second
	ReconnectInitialDelay = 1 * time.Second
	ReconnectMaxDelay     = 30 * time.Second
	MaxReconnectAttempts  = 10
)

// WSClient handles KIS WebSocket connection
type WSClient struct {
	cfg         config.KISConfig
	logger      *logger.Logger
	approvalKey string
	htsID       string

	conn      *websocket.Conn
	connMu    sync.Mutex
	connected bool

	subscriptions       map[string]bool
	executionSubscribed bool
	subMu               sync.RWMutex

	// Callbacks
	onTick       func(*TickData)
	onExecution  func(*ExecutionNotice)
	onError      func(error)
	onConnected  func()
	onDisconnect func()

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewWSClient creates a new WebSocket client
func NewWSClient(cfg config.KISConfig, log *logger.Logger) *WSClient {
	return &WSClient{
		cfg:           cfg,
		logger:        log,
		subscriptions: make(map[string]bool),
		stopCh:        make(chan struct{}),
	}
}

// SetHtsID sets HTS ID for execution notifications
func (c *WSClient) SetHtsID(htsID string) {
	c.htsID = htsID
}

// Callback setters
func (c *WSClient) OnTick(fn func(*TickData))             { c.onTick = fn }
func (c *WSClient) OnExecution(fn func(*ExecutionNotice)) { c.onExecution = fn }
func (c *WSClient) OnError(fn func(error))                { c.onError = fn }
func (c *WSClient) OnConnected(fn func())                 { c.onConnected = fn }
func (c *WSClient) OnDisconnect(fn func())                { c.onDisconnect = fn }

// Connect establishes WebSocket connection
func (c *WSClient) Connect(ctx context.Context) error {
	// Get approval key
	if err := c.getApprovalKey(ctx); err != nil {
		return fmt.Errorf("get approval key: %w", err)
	}

	// Connect WebSocket
	if err := c.connect(ctx); err != nil {
		return fmt.Errorf("websocket connect: %w", err)
	}

	// Start read loop
	c.wg.Add(1)
	go c.readLoop()

	// Start ping loop
	c.wg.Add(1)
	go c.pingLoop()

	c.logger.Info("KIS WebSocket connected")
	return nil
}

// getApprovalKey gets WebSocket approval key
func (c *WSClient) getApprovalKey(ctx context.Context) error {
	url := c.cfg.BaseURL + "/oauth2/Approval"
	body := map[string]string{
		"grant_type": "client_credentials",
		"appkey":     c.cfg.AppKey,
		"secretkey":  c.cfg.AppSecret,
	}

	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		ApprovalKey string `json:"approval_key"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	c.approvalKey = result.ApprovalKey
	return nil
}

// connect establishes WebSocket connection
func (c *WSClient) connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	wsURL := WSURLReal
	if c.cfg.IsVirtual {
		wsURL = WSURLDemo
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	c.connected = true

	if c.onConnected != nil {
		c.onConnected()
	}

	return nil
}

// Disconnect closes the connection
func (c *WSClient) Disconnect() error {
	close(c.stopCh)

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.connected = false
	}
	c.connMu.Unlock()

	c.wg.Wait()

	if c.onDisconnect != nil {
		c.onDisconnect()
	}

	c.logger.Info("KIS WebSocket disconnected")
	return nil
}

// IsConnected returns connection status
func (c *WSClient) IsConnected() bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.connected
}

// Subscribe subscribes to tick data for symbols
func (c *WSClient) Subscribe(symbols ...string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, symbol := range symbols {
		if c.subscriptions[symbol] {
			continue
		}

		if len(c.subscriptions) >= MaxSubscriptionsPerSession {
			return fmt.Errorf("max subscriptions reached (%d)", MaxSubscriptionsPerSession)
		}

		if err := c.sendSubscribe(symbol, "1"); err != nil {
			return fmt.Errorf("subscribe %s: %w", symbol, err)
		}

		c.subscriptions[symbol] = true
		c.logger.WithFields(map[string]interface{}{
			"symbol": symbol,
		}).Debug("Subscribed to tick data")
	}

	return nil
}

// Unsubscribe removes symbol subscriptions
func (c *WSClient) Unsubscribe(symbols ...string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, symbol := range symbols {
		if !c.subscriptions[symbol] {
			continue
		}

		if err := c.sendSubscribe(symbol, "2"); err != nil {
			return fmt.Errorf("unsubscribe %s: %w", symbol, err)
		}

		delete(c.subscriptions, symbol)
	}

	return nil
}

// sendSubscribe sends subscription message
func (c *WSClient) sendSubscribe(symbol, trType string) error {
	msg := wsMessage{
		Header: wsHeader{
			ApprovalKey: c.approvalKey,
			Custtype:    "P",
			TrType:      trType,
			ContentType: "utf-8",
		},
		Body: wsBody{
			Input: wsInput{
				TrID:  TRIDTickReal,
				TrKey: symbol,
			},
		},
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	return c.conn.WriteJSON(msg)
}

// SubscribeExecution subscribes to execution notifications
func (c *WSClient) SubscribeExecution() error {
	if c.htsID == "" {
		return fmt.Errorf("HTS ID not set")
	}

	c.subMu.Lock()
	defer c.subMu.Unlock()

	if c.executionSubscribed {
		return nil
	}

	trID := TRIDExecutionReal
	if c.cfg.IsVirtual {
		trID = TRIDExecutionDemo
	}

	msg := wsMessage{
		Header: wsHeader{
			ApprovalKey: c.approvalKey,
			Custtype:    "P",
			TrType:      "1",
			ContentType: "utf-8",
		},
		Body: wsBody{
			Input: wsInput{
				TrID:  trID,
				TrKey: c.htsID,
			},
		},
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return err
	}

	c.executionSubscribed = true
	c.logger.Info("Subscribed to execution notifications")
	return nil
}

// UnsubscribeExecution unsubscribes from execution notifications
func (c *WSClient) UnsubscribeExecution() error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	if !c.executionSubscribed {
		return nil
	}

	trID := TRIDExecutionReal
	if c.cfg.IsVirtual {
		trID = TRIDExecutionDemo
	}

	msg := wsMessage{
		Header: wsHeader{
			ApprovalKey: c.approvalKey,
			Custtype:    "P",
			TrType:      "2",
			ContentType: "utf-8",
		},
		Body: wsBody{
			Input: wsInput{
				TrID:  trID,
				TrKey: c.htsID,
			},
		},
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return err
	}

	c.executionSubscribed = false
	return nil
}

// GetSubscriptions returns current subscriptions
func (c *WSClient) GetSubscriptions() []string {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	symbols := make([]string, 0, len(c.subscriptions))
	for symbol := range c.subscriptions {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// SubscriptionCount returns number of subscriptions
func (c *WSClient) SubscriptionCount() int {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return len(c.subscriptions)
}

// readLoop handles incoming messages
func (c *WSClient) readLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.connMu.Lock()
		conn := c.conn
		c.connMu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			if c.onError != nil {
				c.onError(fmt.Errorf("read error: %w", err))
			}
			c.handleDisconnect()
			return
		}

		c.handleMessage(message)
	}
}

// handleMessage processes incoming message
func (c *WSClient) handleMessage(data []byte) {
	// Handle PINGPONG
	if strings.Contains(string(data), "PINGPONG") {
		c.connMu.Lock()
		if c.conn != nil {
			c.conn.WriteMessage(websocket.TextMessage, data)
		}
		c.connMu.Unlock()
		return
	}

	// KIS format: encrypted|TR_ID|count|data
	parts := strings.Split(string(data), "|")
	if len(parts) < 4 {
		return // JSON response (subscription confirmation)
	}

	encrypted := parts[0]
	trID := parts[1]
	body := parts[3]

	// Tick data
	if trID == TRIDTickReal {
		tick := c.parseTickData(body)
		if tick != nil && c.onTick != nil {
			c.onTick(tick)
		}
		return
	}

	// Execution notification
	if trID == TRIDExecutionReal || trID == TRIDExecutionDemo {
		if encrypted == "1" {
			decrypted, err := c.decryptData(body)
			if err != nil {
				c.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Error("Failed to decrypt execution data")
				return
			}
			body = decrypted
		}

		exec := c.parseExecutionData(body)
		if exec != nil && c.onExecution != nil {
			c.onExecution(exec)
		}
	}
}

// parseTickData parses tick data from KIS format
// Fields: symbol^time^price^sign^change^changeRate^...^volume^accVolume^...
func (c *WSClient) parseTickData(body string) *TickData {
	fields := strings.Split(body, "^")
	if len(fields) < 14 {
		return nil
	}

	price, _ := strconv.ParseInt(fields[2], 10, 64)
	change, _ := strconv.ParseInt(fields[4], 10, 64)
	changeRate, _ := strconv.ParseFloat(fields[5], 64)
	volume, _ := strconv.ParseInt(fields[12], 10, 64)
	accVolume, _ := strconv.ParseInt(fields[13], 10, 64)

	return &TickData{
		Symbol:     fields[0],
		Price:      price,
		Change:     change,
		ChangeRate: changeRate,
		Volume:     volume,
		AccVolume:  accVolume,
		TradeTime:  fields[1],
		ReceivedAt: time.Now(),
	}
}

// parseExecutionData parses execution notification
func (c *WSClient) parseExecutionData(body string) *ExecutionNotice {
	fields := strings.Split(body, "^")
	if len(fields) < 23 {
		return nil
	}

	orderQty, _ := strconv.ParseInt(fields[16], 10, 64)
	orderPrice, _ := strconv.ParseInt(fields[22], 10, 64)
	execQty, _ := strconv.ParseInt(fields[9], 10, 64)
	execPrice, _ := strconv.ParseInt(fields[10], 10, 64)

	orderSide := "알수없음"
	switch fields[4] {
	case "01":
		orderSide = "매도"
	case "02":
		orderSide = "매수"
	}

	rejectReason := ""
	if fields[12] == "Y" {
		rejectReason = "주문거부"
	}

	return &ExecutionNotice{
		OrderNo:       fields[2],
		OrigOrderNo:   fields[3],
		StockCode:     fields[8],
		StockName:     fields[21],
		OrderSide:     orderSide,
		OrderQuantity: orderQty,
		OrderPrice:    orderPrice,
		ExecutedQty:   execQty,
		ExecutedPrice: execPrice,
		ExecutedAmt:   execQty * execPrice,
		RemainingQty:  orderQty - execQty,
		ExecutedTime:  fields[11],
		RejectReason:  rejectReason,
		ReceivedAt:    time.Now(),
	}
}

// decryptData decrypts AES-256-CBC encrypted data
func (c *WSClient) decryptData(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	// Key: first 32 bytes of appSecret
	key := []byte(c.cfg.AppSecret)
	if len(key) > 32 {
		key = key[:32]
	} else if len(key) < 32 {
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	}

	// IV: first 16 bytes of appSecret
	iv := []byte(c.cfg.AppSecret)
	if len(iv) > 16 {
		iv = iv[:16]
	} else if len(iv) < 16 {
		padded := make([]byte, 16)
		copy(padded, iv)
		iv = padded
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS7 padding
	if len(plaintext) > 0 {
		padding := int(plaintext[len(plaintext)-1])
		if padding <= len(plaintext) {
			plaintext = plaintext[:len(plaintext)-padding]
		}
	}

	return string(plaintext), nil
}

// pingLoop sends periodic pings
func (c *WSClient) pingLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.connMu.Lock()
			if c.conn != nil {
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.connMu.Unlock()
					if c.onError != nil {
						c.onError(fmt.Errorf("ping error: %w", err))
					}
					c.handleDisconnect()
					return
				}
			}
			c.connMu.Unlock()
		}
	}
}

// handleDisconnect handles connection loss
func (c *WSClient) handleDisconnect() {
	c.connMu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	if c.onDisconnect != nil {
		c.onDisconnect()
	}
}

// Reconnect attempts to reconnect
func (c *WSClient) Reconnect(ctx context.Context) error {
	delay := ReconnectInitialDelay

	for attempt := 1; attempt <= MaxReconnectAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		c.logger.WithFields(map[string]interface{}{
			"attempt": attempt,
		}).Info("Attempting WebSocket reconnection")

		// Get new approval key
		if err := c.getApprovalKey(ctx); err != nil {
			delay = delay * 2
			if delay > ReconnectMaxDelay {
				delay = ReconnectMaxDelay
			}
			continue
		}

		// Reconnect
		if err := c.connect(ctx); err != nil {
			delay = delay * 2
			if delay > ReconnectMaxDelay {
				delay = ReconnectMaxDelay
			}
			continue
		}

		// Restore subscriptions
		c.subMu.RLock()
		symbols := make([]string, 0, len(c.subscriptions))
		for symbol := range c.subscriptions {
			symbols = append(symbols, symbol)
		}
		wasExecutionSubscribed := c.executionSubscribed
		c.subMu.RUnlock()

		// Clear and resubscribe
		c.subMu.Lock()
		c.subscriptions = make(map[string]bool)
		c.executionSubscribed = false
		c.subMu.Unlock()

		for _, symbol := range symbols {
			c.Subscribe(symbol)
		}

		if wasExecutionSubscribed {
			c.SubscribeExecution()
		}

		// Restart loops
		c.stopCh = make(chan struct{})
		c.wg.Add(2)
		go c.readLoop()
		go c.pingLoop()

		c.logger.Info("WebSocket reconnected successfully")
		return nil
	}

	return fmt.Errorf("max reconnect attempts reached")
}

// Internal message types
type wsMessage struct {
	Header wsHeader `json:"header"`
	Body   wsBody   `json:"body,omitempty"`
}

type wsHeader struct {
	ApprovalKey string `json:"approval_key,omitempty"`
	Custtype    string `json:"custtype,omitempty"`
	TrType      string `json:"tr_type,omitempty"`
	ContentType string `json:"content-type,omitempty"`
}

type wsBody struct {
	Input wsInput `json:"input,omitempty"`
}

type wsInput struct {
	TrID  string `json:"tr_id"`
	TrKey string `json:"tr_key"`
}
