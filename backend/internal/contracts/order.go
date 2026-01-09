package contracts

import "time"

// Order represents an execution order passed from S6 to broker
// ⭐ SSOT: S6 → Broker 주문 정보 전달
type Order struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Side      OrderSide `json:"side"` // BUY or SELL
	Qty       int       `json:"qty"`
	Price     float64   `json:"price"`      // 0 for market order
	OrderType OrderType `json:"order_type"` // MARKET or LIMIT
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrderSide represents buy or sell
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents market or limit order
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

// Status represents order status
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusSubmitted Status = "SUBMITTED"
	StatusFilled    Status = "FILLED"
	StatusCanceled  Status = "CANCELED"
	StatusRejected  Status = "REJECTED"
)

// IsMarketOrder checks if the order is a market order
func (o *Order) IsMarketOrder() bool {
	return o.OrderType == OrderTypeMarket
}

// IsFilled checks if the order is filled
func (o *Order) IsFilled() bool {
	return o.Status == StatusFilled
}
