package kis

import "time"

// ============================================================
// Balance & Position Types
// ============================================================

// Balance represents account balance summary
type Balance struct {
	TotalDeposit     int64   `json:"total_deposit"`      // 예수금
	AvailableCash    int64   `json:"available_cash"`     // 출금가능금액
	TotalPurchase    int64   `json:"total_purchase"`     // 매입금액합계
	TotalEvaluation  int64   `json:"total_evaluation"`   // 평가금액합계
	TotalProfitLoss  int64   `json:"total_profit_loss"`  // 평가손익합계
	ProfitLossRate   float64 `json:"profit_loss_rate"`   // 수익률
	TotalAsset       int64   `json:"total_asset"`        // 총자산
}

// Position represents a stock position
type Position struct {
	StockCode         string  `json:"stock_code"`
	StockName         string  `json:"stock_name"`
	Quantity          int64   `json:"quantity"`           // 보유수량
	AvailableQuantity int64   `json:"available_quantity"` // 매도가능수량
	AvgBuyPrice       int64   `json:"avg_buy_price"`      // 평균매입가
	CurrentPrice      int64   `json:"current_price"`      // 현재가
	EvalAmount        int64   `json:"eval_amount"`        // 평가금액
	PurchaseAmount    int64   `json:"purchase_amount"`    // 매입금액
	ProfitLoss        int64   `json:"profit_loss"`        // 평가손익
	ProfitLossRate    float64 `json:"profit_loss_rate"`   // 수익률
}

// ============================================================
// Order Types
// ============================================================

// OrderSide represents buy or sell
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents order type
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"  // 지정가
	OrderTypeMarket OrderType = "market" // 시장가
)

// OrderStatus represents order status
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"   // 미체결
	OrderStatusPartial   OrderStatus = "partial"   // 부분체결
	OrderStatusFilled    OrderStatus = "filled"    // 전량체결
	OrderStatusCancelled OrderStatus = "cancelled" // 취소
)

// Order represents a stock order
type Order struct {
	OrderNo          string      `json:"order_no"`
	OrigOrderNo      string      `json:"orig_order_no"`  // 원주문번호 (정정/취소시)
	StockCode        string      `json:"stock_code"`
	StockName        string      `json:"stock_name"`
	OrderSide        OrderSide   `json:"order_side"`     // buy, sell
	OrderType        OrderType   `json:"order_type"`     // limit, market
	OrderPrice       int64       `json:"order_price"`    // 주문가격
	OrderQuantity    int64       `json:"order_quantity"` // 주문수량
	ExecutedQuantity int64       `json:"executed_qty"`   // 체결수량
	ExecutedPrice    int64       `json:"executed_price"` // 체결가격
	RemainingQty     int64       `json:"remaining_qty"`  // 잔여수량
	Status           OrderStatus `json:"status"`
	OrderTime        string      `json:"order_time"` // HHMMSS
	OrderDate        string      `json:"order_date"` // YYYYMMDD
}

// PlaceOrderRequest represents a request to place an order
type PlaceOrderRequest struct {
	StockCode string    `json:"stock_code"`
	Side      OrderSide `json:"side"`       // buy, sell
	Type      OrderType `json:"type"`       // limit, market
	Quantity  int64     `json:"quantity"`
	Price     int64     `json:"price"`      // 0 for market order
}

// PlaceOrderResult represents the result of placing an order
type PlaceOrderResult struct {
	Success   bool   `json:"success"`
	OrderNo   string `json:"order_no"`
	OrderTime string `json:"order_time"`
	Message   string `json:"message"`
}

// ============================================================
// WebSocket Types
// ============================================================

// TickData represents real-time price tick
type TickData struct {
	Symbol     string    `json:"symbol"`
	Price      int64     `json:"price"`
	Change     int64     `json:"change"`
	ChangeRate float64   `json:"change_rate"`
	Volume     int64     `json:"volume"`
	AccVolume  int64     `json:"acc_volume"`
	TradeTime  string    `json:"trade_time"`
	ReceivedAt time.Time `json:"received_at"`
}

// ExecutionNotice represents real-time execution notification
type ExecutionNotice struct {
	OrderNo       string    `json:"order_no"`
	OrigOrderNo   string    `json:"orig_order_no"`
	StockCode     string    `json:"stock_code"`
	StockName     string    `json:"stock_name"`
	OrderSide     string    `json:"order_side"`     // 매수/매도
	OrderQuantity int64     `json:"order_quantity"`
	OrderPrice    int64     `json:"order_price"`
	ExecutedQty   int64     `json:"executed_qty"`
	ExecutedPrice int64     `json:"executed_price"`
	ExecutedAmt   int64     `json:"executed_amount"`
	RemainingQty  int64     `json:"remaining_qty"`
	ExecutedTime  string    `json:"executed_time"`
	RejectReason  string    `json:"reject_reason"`
	ReceivedAt    time.Time `json:"received_at"`
}

// ============================================================
// KIS API Response Types (Internal)
// ============================================================

// balanceResponse represents KIS balance API response
type balanceResponse struct {
	RtCd    string `json:"rt_cd"`
	MsgCd   string `json:"msg_cd"`
	Msg1    string `json:"msg1"`
	Output1 []struct {
		Pdno         string `json:"pdno"`           // 종목코드
		PrdtName     string `json:"prdt_name"`      // 종목명
		HldgQty      string `json:"hldg_qty"`       // 보유수량
		OrdPsblQty   string `json:"ord_psbl_qty"`   // 주문가능수량
		PchsAvgPric  string `json:"pchs_avg_pric"`  // 매입평균가
		Prpr         string `json:"prpr"`           // 현재가
		EvluAmt      string `json:"evlu_amt"`       // 평가금액
		PchsAmt      string `json:"pchs_amt"`       // 매입금액
		EvluPflsAmt  string `json:"evlu_pfls_amt"`  // 평가손익
		EvluPflsRt   string `json:"evlu_pfls_rt"`   // 수익률
	} `json:"output1"`
	Output2 []struct {
		DncaTotAmt      string `json:"dnca_tot_amt"`       // 예수금총금액
		PrvsRcdlExccAmt string `json:"prvs_rcdl_excc_amt"` // 출금가능금액
		PchsAmtSmtlAmt  string `json:"pchs_amt_smtl_amt"`  // 매입금액합계
		EvluAmtSmtlAmt  string `json:"evlu_amt_smtl_amt"`  // 평가금액합계
		EvluPflsSmtlAmt string `json:"evlu_pfls_smtl_amt"` // 평가손익합계
		TotEvluAmt      string `json:"tot_evlu_amt"`       // 총평가금액
	} `json:"output2"`
}

// ordersResponse represents KIS orders API response
type ordersResponse struct {
	RtCd         string `json:"rt_cd"`
	MsgCd        string `json:"msg_cd"`
	Msg1         string `json:"msg1"`
	CtxAreaFK100 string `json:"ctx_area_fk100"` // 연속조회키
	CtxAreaNK100 string `json:"ctx_area_nk100"` // 연속조회키
	Output1      []struct {
		OrdDt        string `json:"ord_dt"`          // 주문일자
		Odno         string `json:"odno"`            // 주문번호
		OrgnOdno     string `json:"orgn_odno"`       // 원주문번호
		SllBuyDvsnCd string `json:"sll_buy_dvsn_cd"` // 01:매도, 02:매수
		Pdno         string `json:"pdno"`            // 종목코드
		PrdtName     string `json:"prdt_name"`       // 종목명
		OrdQty       string `json:"ord_qty"`         // 주문수량
		OrdUnpr      string `json:"ord_unpr"`        // 주문단가
		OrdTmd       string `json:"ord_tmd"`         // 주문시간
		TotCcldQty   string `json:"tot_ccld_qty"`    // 총체결수량
		AvgPrvs      string `json:"avg_prvs"`        // 체결평균가
		RmnQty       string `json:"rmn_qty"`         // 잔여수량
		CnclYn       string `json:"cncl_yn"`         // 취소여부
		OrdDvsnName  string `json:"ord_dvsn_name"`   // 주문구분명
	} `json:"output1"`
}

// placeOrderRequest represents KIS place order request body
type placeOrderRequestBody struct {
	CANO         string `json:"CANO"`          // 계좌번호
	ACNT_PRDT_CD string `json:"ACNT_PRDT_CD"`  // 계좌상품코드
	PDNO         string `json:"PDNO"`          // 종목코드
	ORD_DVSN     string `json:"ORD_DVSN"`      // 00:지정가, 01:시장가
	ORD_QTY      string `json:"ORD_QTY"`       // 주문수량
	ORD_UNPR     string `json:"ORD_UNPR"`      // 주문단가
}

// placeOrderResponse represents KIS place order response
type placeOrderResponse struct {
	RtCd   string `json:"rt_cd"`
	MsgCd  string `json:"msg_cd"`
	Msg1   string `json:"msg1"`
	Output struct {
		KRX_FWDG_ORD_ORGNO string `json:"KRX_FWDG_ORD_ORGNO"`
		ODNO               string `json:"ODNO"`    // 주문번호
		ORD_TMD            string `json:"ORD_TMD"` // 주문시각
	} `json:"output"`
}

// cancelOrderRequest represents KIS cancel order request body
type cancelOrderRequestBody struct {
	CANO              string `json:"CANO"`
	ACNT_PRDT_CD      string `json:"ACNT_PRDT_CD"`
	KRX_FWDG_ORD_ORGNO string `json:"KRX_FWDG_ORD_ORGNO"`
	ORGN_ODNO         string `json:"ORGN_ODNO"`         // 원주문번호
	ORD_DVSN          string `json:"ORD_DVSN"`          // 00
	RVSE_CNCL_DVSN_CD string `json:"RVSE_CNCL_DVSN_CD"` // 02:취소
	ORD_QTY           string `json:"ORD_QTY"`           // 0
	ORD_UNPR          string `json:"ORD_UNPR"`          // 0
	QTY_ALL_ORD_YN    string `json:"QTY_ALL_ORD_YN"`    // Y:전량
}

// hashkeyRequest represents KIS hashkey request body
type hashkeyRequest struct {
	Body interface{} `json:"body"`
}

// hashkeyResponse represents KIS hashkey response
type hashkeyResponse struct {
	Hash string `json:"HASH"`
}
