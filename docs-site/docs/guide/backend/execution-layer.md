---
sidebar_position: 6
title: Execution Layer
description: S6 주문 실행
---

# Execution Layer

> S6: 주문 실행

---

## 책임

목표 포트폴리오를 실제 주문으로 변환하고 체결 관리

---

## 폴더 구조

```
internal/execution/
├── planner.go      # ExecutionPlanner 구현
├── broker.go       # 증권사 연동 (KIS)
└── monitor.go      # 체결 모니터링
```

---

## Execution Planner

### 인터페이스

```go
type ExecutionPlanner interface {
    Plan(ctx context.Context, target *TargetPortfolio) ([]Order, error)
}
```

### 구현

```go
// internal/execution/planner.go

type planner struct {
    broker Broker
    config ExecutionConfig
}

type ExecutionConfig struct {
    OrderType      string  `yaml:"order_type"`       // "LIMIT", "MARKET"
    SlippageBps    int     `yaml:"slippage_bps"`     // 슬리피지 (10 = 0.1%)
    MaxOrderSize   int64   `yaml:"max_order_size"`   // 최대 주문 금액
    SplitThreshold int64   `yaml:"split_threshold"`  // 분할 주문 기준
}

func (p *planner) Plan(ctx context.Context, target *contracts.TargetPortfolio) ([]contracts.Order, error) {
    orders := make([]contracts.Order, 0)

    // 1. 매도 주문 먼저 (자금 확보)
    for _, pos := range target.Positions {
        if pos.Action == contracts.ActionSell {
            order := p.createSellOrder(pos)
            orders = append(orders, order)
        }
    }

    // 2. 매수 주문
    for _, pos := range target.Positions {
        if pos.Action == contracts.ActionBuy {
            order := p.createBuyOrder(pos)
            orders = append(orders, order)
        }
    }

    return orders, nil
}

func (p *planner) createBuyOrder(pos contracts.TargetPosition) contracts.Order {
    price := p.getTargetPrice(pos.Code, "BUY")

    return contracts.Order{
        Code:      pos.Code,
        Side:      "BUY",
        Quantity:  pos.TargetQty,
        Price:     price,
        OrderType: p.config.OrderType,
        Status:    "PENDING",
    }
}

func (p *planner) getTargetPrice(code string, side string) int {
    currentPrice := p.broker.GetCurrentPrice(code)

    if p.config.OrderType == "MARKET" {
        return 0 // 시장가
    }

    // 지정가: 슬리피지 적용
    slippage := float64(p.config.SlippageBps) / 10000

    if side == "BUY" {
        return int(float64(currentPrice) * (1 + slippage))
    }
    return int(float64(currentPrice) * (1 - slippage))
}
```

---

## Broker Interface

```go
// internal/execution/broker.go

type Broker interface {
    // 현재가 조회
    GetCurrentPrice(code string) int

    // 주문 제출
    SubmitOrder(ctx context.Context, order *Order) (*OrderResult, error)

    // 주문 취소
    CancelOrder(ctx context.Context, orderID string) error

    // 주문 상태 조회
    GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)

    // 잔고 조회
    GetBalance(ctx context.Context) (*Balance, error)
}
```

---

## KIS Broker 구현

```go
// internal/external/kis/broker.go

type kisBroker struct {
    client  *KISClient
    account string
}

func (b *kisBroker) SubmitOrder(ctx context.Context, order *contracts.Order) (*contracts.OrderResult, error) {
    // KIS API 호출
    req := &KISOrderRequest{
        AccountNo: b.account,
        StockCode: order.Code,
        OrderQty:  order.Quantity,
        OrderPrice: order.Price,
        OrderType: b.convertOrderType(order.OrderType),
        Side:      b.convertSide(order.Side),
    }

    resp, err := b.client.PlaceOrder(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("KIS order failed: %w", err)
    }

    return &contracts.OrderResult{
        OrderID:   resp.OrderNo,
        Status:    "SUBMITTED",
        Timestamp: time.Now(),
    }, nil
}
```

---

## 주문 분할

대량 주문 시 분할 처리:

```go
func (p *planner) splitOrder(order contracts.Order) []contracts.Order {
    if order.Quantity * order.Price < p.config.SplitThreshold {
        return []contracts.Order{order}
    }

    // 분할
    chunks := make([]contracts.Order, 0)
    remaining := order.Quantity
    chunkSize := p.config.MaxOrderSize / order.Price

    for remaining > 0 {
        qty := min(remaining, chunkSize)
        chunk := order
        chunk.Quantity = qty
        chunks = append(chunks, chunk)
        remaining -= qty
    }

    return chunks
}
```

---

## 설정 예시 (YAML)

```yaml
# config/execution.yaml

execution:
  order_type: "LIMIT"        # LIMIT, MARKET
  slippage_bps: 10           # 0.1%
  max_order_size: 50000000   # 5천만원
  split_threshold: 100000000 # 1억원 이상 분할

  # 주문 시간
  order_time:
    start: "09:00"
    end: "15:20"

  # 재시도
  retry:
    max_attempts: 3
    delay_seconds: 5
```

---

## DB 스키마

```sql
-- execution.orders: 주문 내역
CREATE TABLE execution.orders (
    id           SERIAL PRIMARY KEY,
    order_id     VARCHAR(50) UNIQUE,  -- 증권사 주문번호
    code         VARCHAR(10) NOT NULL,
    side         VARCHAR(4) NOT NULL,  -- BUY, SELL
    quantity     INT NOT NULL,
    price        INT,
    order_type   VARCHAR(10),          -- LIMIT, MARKET
    status       VARCHAR(20),          -- PENDING, FILLED, PARTIAL, REJECTED
    filled_qty   INT DEFAULT 0,
    filled_price INT,
    reason       TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

-- execution.executions: 체결 내역
CREATE TABLE execution.executions (
    id          SERIAL PRIMARY KEY,
    order_id    VARCHAR(50) REFERENCES execution.orders(order_id),
    exec_qty    INT NOT NULL,
    exec_price  INT NOT NULL,
    exec_time   TIMESTAMPTZ NOT NULL,
    fee         DECIMAL(12,2),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_status ON execution.orders(status);
CREATE INDEX idx_orders_date ON execution.orders(created_at);
```

---

**Prev**: [Portfolio Layer](./portfolio-layer.md)
**Next**: [Audit Layer](./audit-layer.md)
