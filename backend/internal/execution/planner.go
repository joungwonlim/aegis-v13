package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Planner implements S6: Execution planning
// ⭐ SSOT: S6 주문 계획 로직은 여기서만
type Planner struct {
	broker Broker
	config ExecutionConfig
	logger *logger.Logger
}

// ExecutionConfig defines execution parameters
type ExecutionConfig struct {
	OrderType      contracts.OrderType // LIMIT or MARKET
	SlippageBps    int                 // 슬리피지 (10 = 0.1%)
	MaxOrderSize   int64               // 최대 주문 금액
	SplitThreshold int64               // 분할 주문 기준
}

// NewPlanner creates a new execution planner
func NewPlanner(broker Broker, config ExecutionConfig, logger *logger.Logger) *Planner {
	return &Planner{
		broker: broker,
		config: config,
		logger: logger,
	}
}

// Plan creates execution orders from target portfolio
func (p *Planner) Plan(ctx context.Context, target *contracts.TargetPortfolio) ([]contracts.Order, error) {
	orders := make([]contracts.Order, 0)

	// 1. 매도 주문 먼저 (자금 확보)
	for _, pos := range target.Positions {
		if pos.Action == contracts.ActionSell {
			order, err := p.createSellOrder(ctx, pos)
			if err != nil {
				p.logger.WithFields(map[string]interface{}{
					"code":  pos.Code,
					"error": err,
				}).Warn("Failed to create sell order")
				continue
			}
			orders = append(orders, order)
		}
	}

	// 2. 매수 주문
	for _, pos := range target.Positions {
		if pos.Action == contracts.ActionBuy {
			order, err := p.createBuyOrder(ctx, pos)
			if err != nil {
				p.logger.WithFields(map[string]interface{}{
					"code":  pos.Code,
					"error": err,
				}).Warn("Failed to create buy order")
				continue
			}
			orders = append(orders, order)
		}
	}

	p.logger.WithFields(map[string]interface{}{
		"total_orders": len(orders),
		"sell_orders":  p.countOrders(orders, contracts.OrderSideSell),
		"buy_orders":   p.countOrders(orders, contracts.OrderSideBuy),
	}).Info("Execution plan created")

	return orders, nil
}

// createBuyOrder creates a buy order from target position
// ⭐ P0 수정: TargetValue에서 수량 계산 (0 수량 방지)
func (p *Planner) createBuyOrder(ctx context.Context, pos contracts.TargetPosition) (contracts.Order, error) {
	// 1. 현재가 조회
	currentPrice, err := p.broker.GetCurrentPrice(ctx, pos.Code)
	if err != nil {
		return contracts.Order{}, fmt.Errorf("failed to get current price: %w", err)
	}

	// 2. 수량 계산 (TargetValue / 현재가)
	// ⭐ 계약: Portfolio는 TargetValue만 제공, Execution이 수량 계산
	qty := pos.CalculateQty(int64(currentPrice))
	if qty <= 0 {
		return contracts.Order{}, fmt.Errorf("calculated qty is zero (targetValue=%d, price=%.0f)", pos.TargetValue, currentPrice)
	}

	// 3. 주문가격 결정 (지정가/시장가)
	orderPrice, err := p.getTargetPrice(ctx, pos.Code, contracts.OrderSideBuy)
	if err != nil {
		return contracts.Order{}, fmt.Errorf("failed to get target price: %w", err)
	}

	return contracts.Order{
		Code:      pos.Code,
		Name:      pos.Name,
		Side:      contracts.OrderSideBuy,
		Qty:       qty,
		Price:     orderPrice,
		OrderType: p.config.OrderType,
		Status:    contracts.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// createSellOrder creates a sell order from target position
// ⭐ P0 수정: TargetValue에서 수량 계산 (0 수량 방지)
func (p *Planner) createSellOrder(ctx context.Context, pos contracts.TargetPosition) (contracts.Order, error) {
	// 1. 현재가 조회
	currentPrice, err := p.broker.GetCurrentPrice(ctx, pos.Code)
	if err != nil {
		return contracts.Order{}, fmt.Errorf("failed to get current price: %w", err)
	}

	// 2. 수량 계산 (TargetValue / 현재가)
	qty := pos.CalculateQty(int64(currentPrice))
	if qty <= 0 {
		return contracts.Order{}, fmt.Errorf("calculated qty is zero (targetValue=%d, price=%.0f)", pos.TargetValue, currentPrice)
	}

	// 3. 주문가격 결정 (지정가/시장가)
	orderPrice, err := p.getTargetPrice(ctx, pos.Code, contracts.OrderSideSell)
	if err != nil {
		return contracts.Order{}, fmt.Errorf("failed to get target price: %w", err)
	}

	return contracts.Order{
		Code:      pos.Code,
		Name:      pos.Name,
		Side:      contracts.OrderSideSell,
		Qty:       qty,
		Price:     orderPrice,
		OrderType: p.config.OrderType,
		Status:    contracts.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// getTargetPrice calculates target price with slippage
func (p *Planner) getTargetPrice(ctx context.Context, code string, side contracts.OrderSide) (float64, error) {
	if p.config.OrderType == contracts.OrderTypeMarket {
		return 0, nil // 시장가
	}

	currentPrice, err := p.broker.GetCurrentPrice(ctx, code)
	if err != nil {
		return 0, fmt.Errorf("failed to get current price: %w", err)
	}

	// 지정가: 슬리피지 적용
	slippage := float64(p.config.SlippageBps) / 10000

	if side == contracts.OrderSideBuy {
		return currentPrice * (1 + slippage), nil
	}
	return currentPrice * (1 - slippage), nil
}

// splitOrder splits large order into smaller chunks
// ⭐ P0 수정: Price=0(시장가)일 때 0 나눗셈 방지
func (p *Planner) splitOrder(order contracts.Order) []contracts.Order {
	// 시장가 주문(Price=0)이면 분할 불가 - 그대로 반환
	if order.Price <= 0 {
		p.logger.WithFields(map[string]interface{}{
			"code":  order.Code,
			"qty":   order.Qty,
			"price": order.Price,
		}).Warn("Cannot split market order (price=0), returning as-is")
		return []contracts.Order{order}
	}

	orderValue := int64(order.Qty) * int64(order.Price)

	if orderValue < p.config.SplitThreshold {
		return []contracts.Order{order}
	}

	// 분할
	chunks := make([]contracts.Order, 0)
	remaining := order.Qty
	chunkSize := int(p.config.MaxOrderSize / int64(order.Price))

	// chunkSize가 0이면 분할 불가 (MaxOrderSize < Price)
	if chunkSize <= 0 {
		p.logger.WithFields(map[string]interface{}{
			"code":         order.Code,
			"price":        order.Price,
			"maxOrderSize": p.config.MaxOrderSize,
		}).Warn("chunkSize is zero, returning order as-is")
		return []contracts.Order{order}
	}

	for remaining > 0 {
		qty := remaining
		if qty > chunkSize {
			qty = chunkSize
		}

		chunk := order
		chunk.Qty = qty
		chunks = append(chunks, chunk)
		remaining -= qty
	}

	p.logger.WithFields(map[string]interface{}{
		"code":       order.Code,
		"totalQty":   order.Qty,
		"chunks":     len(chunks),
		"chunkSize":  chunkSize,
		"orderValue": orderValue,
	}).Debug("Order split into chunks")

	return chunks
}

// countOrders counts orders by side
func (p *Planner) countOrders(orders []contracts.Order, side contracts.OrderSide) int {
	count := 0
	for _, order := range orders {
		if order.Side == side {
			count++
		}
	}
	return count
}

// DefaultExecutionConfig returns default configuration
func DefaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		OrderType:      contracts.OrderTypeLimit, // 기본: 지정가
		SlippageBps:    10,                       // 0.1% 슬리피지
		MaxOrderSize:   50_000_000,               // 5천만원
		SplitThreshold: 100_000_000,              // 1억원 이상 분할
	}
}
