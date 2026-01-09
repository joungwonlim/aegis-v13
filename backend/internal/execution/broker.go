package execution

import (
	"context"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Broker defines interface for broker operations
// ⭐ SSOT: 증권사 연동 인터페이스는 여기서만 정의
type Broker interface {
	// GetCurrentPrice retrieves current price for a stock
	GetCurrentPrice(ctx context.Context, code string) (float64, error)

	// SubmitOrder submits an order to the broker
	SubmitOrder(ctx context.Context, order *contracts.Order) (*OrderResult, error)

	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string) error

	// GetOrderStatus retrieves order status
	GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)

	// GetBalance retrieves account balance
	GetBalance(ctx context.Context) (*Balance, error)

	// GetHoldings retrieves current holdings
	GetHoldings(ctx context.Context) ([]Holding, error)
}

// OrderResult represents order submission result
type OrderResult struct {
	OrderID   string // 증권사 주문번호
	Status    contracts.Status
	Message   string
	Timestamp string
}

// OrderStatus represents order status
type OrderStatus struct {
	OrderID     string
	Status      contracts.Status
	FilledQty   int
	FilledPrice float64
	RemainingQty int
	UpdatedAt   string
}

// Balance represents account balance
type Balance struct {
	Cash            float64 // 현금
	StockValue      float64 // 주식 평가액
	TotalValue      float64 // 총 평가액
	AvailableCash   float64 // 인출 가능 현금
	PurchasePower   float64 // 매수 가능 금액
	UnrealizedPnL   float64 // 평가손익
	UnrealizedPnLPct float64 // 평가손익률
}

// Holding represents a stock holding
type Holding struct {
	Code            string
	Name            string
	Qty             int
	AvgPrice        float64
	CurrentPrice    float64
	MarketValue     float64
	UnrealizedPnL   float64
	UnrealizedPnLPct float64
}

// MockBroker implements Broker interface for testing
// ⭐ 실제 운영에서는 KIS Broker 사용
type MockBroker struct {
	prices   map[string]float64
	holdings map[string]Holding
	balance  Balance
}

// NewMockBroker creates a new mock broker
func NewMockBroker() *MockBroker {
	return &MockBroker{
		prices:   make(map[string]float64),
		holdings: make(map[string]Holding),
		balance: Balance{
			Cash:          10_000_000, // 천만원
			AvailableCash: 10_000_000,
			PurchasePower: 10_000_000,
		},
	}
}

// GetCurrentPrice retrieves current price
func (b *MockBroker) GetCurrentPrice(ctx context.Context, code string) (float64, error) {
	if price, exists := b.prices[code]; exists {
		return price, nil
	}
	// Default mock price
	return 50000, nil
}

// SubmitOrder submits an order
func (b *MockBroker) SubmitOrder(ctx context.Context, order *contracts.Order) (*OrderResult, error) {
	return &OrderResult{
		OrderID:   "MOCK-" + order.Code,
		Status:    contracts.StatusSubmitted,
		Message:   "Order submitted successfully",
		Timestamp: order.CreatedAt.Format("15:04:05"),
	}, nil
}

// CancelOrder cancels an order
func (b *MockBroker) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

// GetOrderStatus retrieves order status
func (b *MockBroker) GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error) {
	return &OrderStatus{
		OrderID:      orderID,
		Status:       contracts.StatusFilled,
		FilledQty:    100,
		FilledPrice:  50000,
		RemainingQty: 0,
		UpdatedAt:    "15:04:05",
	}, nil
}

// GetBalance retrieves account balance
func (b *MockBroker) GetBalance(ctx context.Context) (*Balance, error) {
	return &b.balance, nil
}

// GetHoldings retrieves current holdings
func (b *MockBroker) GetHoldings(ctx context.Context) ([]Holding, error) {
	holdings := make([]Holding, 0, len(b.holdings))
	for _, h := range b.holdings {
		holdings = append(holdings, h)
	}
	return holdings, nil
}

// SetPrice sets mock price for testing
func (b *MockBroker) SetPrice(code string, price float64) {
	b.prices[code] = price
}

// SetHolding sets mock holding for testing
func (b *MockBroker) SetHolding(holding Holding) {
	b.holdings[holding.Code] = holding
}
