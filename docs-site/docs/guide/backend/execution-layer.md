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

:::tip ⭐ P0 핵심 책임: 수량 계산
Portfolio(S5)는 **목표 금액(TargetValue)**만 산출합니다.
Execution(S6)은 **현재가를 조회하여 수량(Qty)을 계산**합니다.

```
Qty = TargetValue / CurrentPrice
```

이렇게 분리하는 이유:
- Portfolio는 가격 변동에 독립적
- 실제 주문 시점의 현재가로 정확한 수량 계산
- 재현성: 같은 TargetValue로 언제든 수량 재계산 가능
:::

---

## 구현 상태 (2026-01-11)

| 컴포넌트 | 상태 | 파일 |
|---------|------|------|
| **ExecutionPlanner** | ✅ 완료 | `internal/execution/planner.go` |
| **Broker Interface** | ✅ 완료 | `internal/execution/broker.go` |
| **ExecutionMonitor** | ✅ 완료 | `internal/execution/monitor.go` |
| **Repository** | ✅ 완료 | `internal/execution/repository.go` |
| **RiskGate (Phase B)** | ✅ 완료 | `internal/execution/risk_gate.go` |
| KIS API 연동 | ⏳ TODO | `internal/external/kis/` |

### 리스크 게이트 (Phase B)

| 컴포넌트 | 상태 | 설명 |
|---------|------|------|
| **RiskGate** | ✅ 완료 | S6 사전 리스크 체크 |
| **Shadow Mode** | ✅ 완료 | 차단 로깅 (would_block) |
| **Gate CLI** | ✅ 완료 | `go run ./cmd/quant gate` |
| **DB Migration** | ✅ 완료 | `migrations/026_risk_gate_events.sql` |

:::tip YAML SSOT
주문 설정과 슬리피지 모델은 `backend/config/strategy/korea_equity_v13.yaml`의 `execution` 섹션에서 관리됩니다.
:::

---

## 프로세스 흐름

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Execution Pipeline                                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           입력: TargetPortfolio                              │
│                        (S5에서 생성된 목표 포트폴리오)                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌───────────────────┐       ┌───────────────────┐       ┌───────────────────┐
│  1. 주문 계획     │──────▶│  2. 주문 분할     │──────▶│  3. 주문 제출     │
│   (Planner)       │       │   (Splitting)     │       │   (Broker)        │
├───────────────────┤       ├───────────────────┤       ├───────────────────┤
│ • 현재 보유 조회  │       │ 주문/ADTV20 ≥ 2% │       │ • order_type:     │
│ • Diff 계산       │       │ → 분할 실행       │       │   LIMIT           │
│ • 매도 우선 생성  │       │ • min_slices: 3   │       │ • 매수: ASK1      │
│ • 매수 주문 생성  │       │ • max_slices: 8   │       │ • 매도: BID1      │
│                   │       │ • interval: 90s   │       │                   │
└───────────────────┘       └───────────────────┘       └───────────────────┘
                                                                │
                                                                ▼
                                                ┌───────────────────────────┐
                                                │  4. 체결 모니터링          │
                                                │      (Monitor)            │
                                                ├───────────────────────────┤
                                                │ • 체결 상태 추적           │
                                                │ • 미체결 주문 처리          │
                                                │ • 슬리피지 기록            │
                                                └───────────────────────────┘
```

---

## 폴더 구조

```
internal/execution/
├── planner.go      # ExecutionPlanner 구현 (수량 계산⭐)
├── broker.go       # 증권사 연동 (KIS)
├── monitor.go      # 체결 모니터링
├── exit_rules.go   # 청산 규칙 (손절/익절/추세)
└── repository.go   # DB 접근 (SSOT)
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
            order, err := p.createSellOrder(ctx, pos)
            if err != nil {
                return nil, err
            }
            orders = append(orders, order)
        }
    }

    // 2. 매수 주문
    for _, pos := range target.Positions {
        if pos.Action == contracts.ActionBuy {
            order, err := p.createBuyOrder(ctx, pos)
            if err != nil {
                return nil, err
            }
            orders = append(orders, order)
        }
    }

    return orders, nil
}

// ⭐ P0 핵심: TargetValue를 현재가로 나눠 수량 계산
func (p *planner) createBuyOrder(ctx context.Context, pos contracts.TargetPosition) (contracts.Order, error) {
    // 1. 현재가 조회
    currentPrice, err := p.broker.GetCurrentPrice(ctx, pos.Code)
    if err != nil {
        return contracts.Order{}, fmt.Errorf("get price for %s: %w", pos.Code, err)
    }

    // 2. 수량 계산: Qty = TargetValue / CurrentPrice
    qty := pos.CalculateQty(int64(currentPrice))
    if qty <= 0 {
        return contracts.Order{}, fmt.Errorf("calculated qty is zero for %s (value=%d, price=%.0f)",
            pos.Code, pos.TargetValue, currentPrice)
    }

    // 3. 주문가 결정 (슬리피지 적용)
    price := p.getTargetPrice(currentPrice, "BUY")

    return contracts.Order{
        Code:      pos.Code,
        Side:      "BUY",
        Quantity:  qty,
        Price:     price,
        OrderType: p.config.OrderType,
        Status:    "PENDING",
    }, nil
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

// ⭐ P0 수정: 모든 메서드에 ctx 추가, 가격은 float64 반환
type Broker interface {
    // 현재가 조회 (float64로 통일, 호출자가 필요시 int64로 변환)
    GetCurrentPrice(ctx context.Context, code string) (float64, error)

    // 주문 제출
    SubmitOrder(ctx context.Context, order *Order) (*OrderResult, error)

    // 주문 취소
    CancelOrder(ctx context.Context, orderID string) error

    // 주문 상태 조회
    GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)

    // 잔고 조회
    GetBalance(ctx context.Context) (*Balance, error)

    // 현재 보유 종목 조회
    GetHoldings(ctx context.Context) ([]Holding, error)
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
// ⭐ P0 수정: Price=0 (시장가) 방어 코드 추가
func (p *planner) splitOrder(order contracts.Order) []contracts.Order {
    // 방어: 시장가 주문(Price=0)은 분할 불가
    if order.Price <= 0 {
        p.logger.Warn("Cannot split market order (price=0), returning as-is",
            "code", order.Code)
        return []contracts.Order{order}
    }

    if order.Quantity * order.Price < p.config.SplitThreshold {
        return []contracts.Order{order}
    }

    // 분할
    chunks := make([]contracts.Order, 0)
    remaining := order.Quantity
    chunkSize := p.config.MaxOrderSize / order.Price

    // 방어: chunkSize가 0이면 분할 불가
    if chunkSize <= 0 {
        p.logger.Warn("ChunkSize is zero, returning order as-is",
            "code", order.Code, "maxOrderSize", p.config.MaxOrderSize)
        return []contracts.Order{order}
    }

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
