package kis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TR IDs for order operations
const (
	// 주문 조회
	TRIDOrdersReal    = "TTTC8001R"
	TRIDOrdersVirtual = "VTTC8001R"

	// 매수
	TRIDBuyReal    = "TTTC0802U"
	TRIDBuyVirtual = "VTTC0802U"

	// 매도
	TRIDSellReal    = "TTTC0801U"
	TRIDSellVirtual = "VTTC0801U"

	// 취소
	TRIDCancelReal    = "TTTC0803U"
	TRIDCancelVirtual = "VTTC0803U"
)

// GetOrders returns orders within date range
func (c *Client) GetOrders(ctx context.Context, startDate, endDate string) ([]Order, error) {
	path := "/uapi/domestic-stock/v1/trading/inquire-daily-ccld"

	trID := TRIDOrdersReal
	if c.cfg.IsVirtual {
		trID = TRIDOrdersVirtual
	}

	accountNo := c.cfg.AccountNo
	cano := accountNo[:8]
	acntPrdtCd := accountNo[8:10]

	// Default dates if not provided
	if startDate == "" {
		startDate = time.Now().Format("20060102")
	}
	if endDate == "" {
		endDate = time.Now().Format("20060102")
	}

	params := fmt.Sprintf("?CANO=%s&ACNT_PRDT_CD=%s&INQR_STRT_DT=%s&INQR_END_DT=%s&SLL_BUY_DVSN_CD=00&INQR_DVSN=00&PDNO=&CCLD_DVSN=00&ORD_GNO_BRNO=&ODNO=&INQR_DVSN_3=00&INQR_DVSN_1=&CTX_AREA_FK100=&CTX_AREA_NK100=",
		cano, acntPrdtCd, startDate, endDate)

	resp, err := c.request(ctx, http.MethodGet, path+params, trID, nil)
	if err != nil {
		return nil, fmt.Errorf("orders request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("orders API error status %d: %s", resp.StatusCode, string(body))
	}

	var result ordersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode orders response: %w", err)
	}

	if result.RtCd != "0" {
		return nil, fmt.Errorf("orders API error: %s - %s", result.MsgCd, result.Msg1)
	}

	orders := make([]Order, 0, len(result.Output1))
	for _, out := range result.Output1 {
		order := Order{
			OrderNo:          out.Odno,
			OrigOrderNo:      out.OrgnOdno,
			StockCode:        out.Pdno,
			StockName:        out.PrdtName,
			OrderSide:        parseOrderSide(out.SllBuyDvsnCd),
			OrderPrice:       parseIntSafe(out.OrdUnpr),
			OrderQuantity:    parseIntSafe(out.OrdQty),
			ExecutedQuantity: parseIntSafe(out.TotCcldQty),
			ExecutedPrice:    parseIntSafe(out.AvgPrvs),
			RemainingQty:     parseIntSafe(out.RmnQty),
			Status:           parseOrderStatus(out),
			OrderTime:        out.OrdTmd,
			OrderDate:        out.OrdDt,
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetUnfilledOrders returns only unfilled (pending) orders
func (c *Client) GetUnfilledOrders(ctx context.Context) ([]Order, error) {
	orders, err := c.GetOrders(ctx, "", "")
	if err != nil {
		return nil, err
	}

	unfilled := make([]Order, 0)
	for _, o := range orders {
		if o.Status == OrderStatusPending || o.Status == OrderStatusPartial {
			unfilled = append(unfilled, o)
		}
	}
	return unfilled, nil
}

// GetFilledOrders returns only filled orders
func (c *Client) GetFilledOrders(ctx context.Context) ([]Order, error) {
	orders, err := c.GetOrders(ctx, "", "")
	if err != nil {
		return nil, err
	}

	filled := make([]Order, 0)
	for _, o := range orders {
		if o.Status == OrderStatusFilled {
			filled = append(filled, o)
		}
	}
	return filled, nil
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*PlaceOrderResult, error) {
	path := "/uapi/domestic-stock/v1/trading/order-cash"

	// Determine TR ID
	var trID string
	if req.Side == OrderSideBuy {
		trID = TRIDBuyReal
		if c.cfg.IsVirtual {
			trID = TRIDBuyVirtual
		}
	} else {
		trID = TRIDSellReal
		if c.cfg.IsVirtual {
			trID = TRIDSellVirtual
		}
	}

	accountNo := c.cfg.AccountNo
	cano := accountNo[:8]
	acntPrdtCd := accountNo[8:10]

	// Order division code: 00=지정가, 01=시장가
	ordDvsn := "00"
	if req.Type == OrderTypeMarket {
		ordDvsn = "01"
	}

	// Build request body
	body := placeOrderRequestBody{
		CANO:         cano,
		ACNT_PRDT_CD: acntPrdtCd,
		PDNO:         req.StockCode,
		ORD_DVSN:     ordDvsn,
		ORD_QTY:      fmt.Sprintf("%d", req.Quantity),
		ORD_UNPR:     fmt.Sprintf("%d", req.Price),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal order body: %w", err)
	}

	// Get hashkey for POST request
	hashkey, err := c.getHashkey(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("get hashkey: %w", err)
	}

	// Make request with hashkey
	resp, err := c.requestWithHashkey(ctx, http.MethodPost, path, trID, hashkey, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("place order request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("place order API error status %d: %s", resp.StatusCode, string(respBody))
	}

	var result placeOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode place order response: %w", err)
	}

	orderResult := &PlaceOrderResult{
		Success:   result.RtCd == "0",
		OrderNo:   result.Output.ODNO,
		OrderTime: result.Output.ORD_TMD,
		Message:   result.Msg1,
	}

	if !orderResult.Success {
		c.logger.WithFields(map[string]interface{}{
			"stock_code": req.StockCode,
			"side":       req.Side,
			"error":      result.Msg1,
		}).Error("Order placement failed")
	} else {
		c.logger.WithFields(map[string]interface{}{
			"stock_code": req.StockCode,
			"side":       req.Side,
			"order_no":   orderResult.OrderNo,
			"quantity":   req.Quantity,
			"price":      req.Price,
		}).Info("Order placed successfully")
	}

	return orderResult, nil
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(ctx context.Context, orderNo string) (*PlaceOrderResult, error) {
	path := "/uapi/domestic-stock/v1/trading/order-rvsecncl"

	trID := TRIDCancelReal
	if c.cfg.IsVirtual {
		trID = TRIDCancelVirtual
	}

	accountNo := c.cfg.AccountNo
	cano := accountNo[:8]
	acntPrdtCd := accountNo[8:10]

	body := cancelOrderRequestBody{
		CANO:               cano,
		ACNT_PRDT_CD:       acntPrdtCd,
		KRX_FWDG_ORD_ORGNO: "",
		ORGN_ODNO:          orderNo,
		ORD_DVSN:           "00",
		RVSE_CNCL_DVSN_CD:  "02", // 02: 취소
		ORD_QTY:            "0",
		ORD_UNPR:           "0",
		QTY_ALL_ORD_YN:     "Y", // 전량 취소
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal cancel body: %w", err)
	}

	hashkey, err := c.getHashkey(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("get hashkey: %w", err)
	}

	resp, err := c.requestWithHashkey(ctx, http.MethodPost, path, trID, hashkey, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("cancel order request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cancel order API error status %d: %s", resp.StatusCode, string(respBody))
	}

	var result placeOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode cancel response: %w", err)
	}

	orderResult := &PlaceOrderResult{
		Success:   result.RtCd == "0",
		OrderNo:   result.Output.ODNO,
		OrderTime: result.Output.ORD_TMD,
		Message:   result.Msg1,
	}

	if orderResult.Success {
		c.logger.WithFields(map[string]interface{}{
			"order_no": orderNo,
		}).Info("Order cancelled successfully")
	}

	return orderResult, nil
}

// getHashkey generates hashkey for POST requests
func (c *Client) getHashkey(ctx context.Context, body interface{}) (string, error) {
	path := "/uapi/hashkey"

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s%s", c.cfg.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.cfg.AppKey)
	req.Header.Set("appsecret", c.cfg.AppSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result hashkeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Hash, nil
}

// requestWithHashkey makes a POST request with hashkey header
func (c *Client) requestWithHashkey(ctx context.Context, method, path, trID, hashkey string, body io.Reader) (*http.Response, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.cfg.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.cfg.AppKey)
	req.Header.Set("appsecret", c.cfg.AppSecret)
	req.Header.Set("tr_id", trID)
	req.Header.Set("hashkey", hashkey)

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// Helper functions
func parseOrderSide(code string) OrderSide {
	if code == "01" {
		return OrderSideSell
	}
	return OrderSideBuy // "02" or default
}

func parseOrderStatus(out struct {
	OrdDt        string `json:"ord_dt"`
	Odno         string `json:"odno"`
	OrgnOdno     string `json:"orgn_odno"`
	SllBuyDvsnCd string `json:"sll_buy_dvsn_cd"`
	Pdno         string `json:"pdno"`
	PrdtName     string `json:"prdt_name"`
	OrdQty       string `json:"ord_qty"`
	OrdUnpr      string `json:"ord_unpr"`
	OrdTmd       string `json:"ord_tmd"`
	TotCcldQty   string `json:"tot_ccld_qty"`
	AvgPrvs      string `json:"avg_prvs"`
	RmnQty       string `json:"rmn_qty"`
	CnclYn       string `json:"cncl_yn"`
	OrdDvsnName  string `json:"ord_dvsn_name"`
}) OrderStatus {
	if out.CnclYn == "Y" {
		return OrderStatusCancelled
	}

	orderQty := parseIntSafe(out.OrdQty)
	executedQty := parseIntSafe(out.TotCcldQty)
	remainingQty := parseIntSafe(out.RmnQty)

	if remainingQty == 0 && executedQty == orderQty {
		return OrderStatusFilled
	}
	if executedQty > 0 && remainingQty > 0 {
		return OrderStatusPartial
	}
	return OrderStatusPending
}

// generateSimpleHash generates a simple hash for debugging (not for production)
func generateSimpleHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
