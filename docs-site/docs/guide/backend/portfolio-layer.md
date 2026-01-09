---
sidebar_position: 5
title: Portfolio Layer
description: S5 포트폴리오 구성
---

# Portfolio Layer

> S5: 포트폴리오 구성

---

## 책임

랭킹 결과를 바탕으로 **목표 포트폴리오** 산출

---

## 구현 상태 (2026-01-10)

| 컴포넌트 | 상태 | 파일 |
|---------|------|------|
| **Constructor** | ✅ 완료 | `internal/portfolio/constructor.go` |
| **Constraints** | ✅ 완료 | `internal/portfolio/constraints.go` |
| **Repository** | ✅ 완료 | `internal/portfolio/repository.go` |
| Rebalancer | ⏳ TODO | `internal/portfolio/rebalancer.go` |

:::tip YAML SSOT
포트폴리오 제약조건과 비중 배분은 `backend/config/strategy/korea_equity_v13.yaml`의 `portfolio` 섹션에서 관리됩니다.
:::

---

## 프로세스 흐름

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Portfolio Construction Pipeline                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              입력: RankedStock[]                             │
│                           (S4에서 점수순 정렬된 종목)                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌───────────────────┐       ┌───────────────────┐       ┌───────────────────┐
│  1. Top-N 선정    │──────▶│  2. Tier 비중     │──────▶│  3. 제약조건 적용  │
├───────────────────┤       ├───────────────────┤       ├───────────────────┤
│ target: 20종목    │       │ Tier 1: 5종목×5%  │       │ position_max: 10% │
│ min: 15종목       │       │ Tier 2: 10종목×4.5%│       │ position_min: 4%  │
│ max: 20종목       │       │ Tier 3: 5종목×4%  │       │ sector_max: 25%   │
│                   │       │ → 합계: 90%       │       │ cash: 10%         │
└───────────────────┘       └───────────────────┘       └───────────────────┘
                                                                │
                                                                ▼
                                                ┌───────────────────────────┐
                                                │   출력: TargetPortfolio    │
                                                │  (종목, 비중, 매매액션)     │
                                                └───────────────────────────┘
```

---

## 폴더 구조

```
internal/portfolio/
├── constructor.go   # Portfolio Constructor 구현
├── constraints.go   # 제약조건 검증
└── repository.go    # DB 접근 (SSOT)
```

> `rebalancer.go`는 향후 구현 예정

---

## Portfolio Constructor

### 인터페이스

```go
type PortfolioConstructor interface {
    Construct(ctx context.Context, ranked []RankedStock) (*TargetPortfolio, error)
}
```

### 구현

```go
// internal/portfolio/constructor.go

type Constructor struct {
    config      PortfolioConfig
    constraints Constraints
    logger      *logger.Logger
}

type PortfolioConfig struct {
    MaxPositions  int     // 최대 종목 수
    MaxWeight     float64 // 종목당 최대 비중 (0.0 ~ 1.0)
    MinWeight     float64 // 종목당 최소 비중 (0.0 ~ 1.0)
    CashReserve   float64 // 현금 보유 비중 (0.0 ~ 1.0)
    TurnoverLimit float64 // 회전율 제한 (0.0 ~ 1.0)
    WeightingMode string  // "equal", "score_based", "risk_parity"
}

func (c *Constructor) Construct(ctx context.Context, ranked []contracts.RankedStock) (*contracts.TargetPortfolio, error) {
    target := &contracts.TargetPortfolio{
        Date:      time.Now(),
        Positions: make([]contracts.TargetPosition, 0),
        Cash:      c.config.CashReserve,
    }

    // 1. Select top N stocks
    topN := c.selectTopN(ranked)
    if len(topN) == 0 {
        c.logger.Warn("No stocks selected for portfolio")
        return target, nil
    }

    // 2. Calculate weights
    weights := c.calculateWeights(topN)

    // 3. Apply constraints
    weights = c.applyConstraints(weights)

    // 4. Create target positions
    for code, weight := range weights {
        // Find stock info from ranked
        var stock *contracts.RankedStock
        for i := range topN {
            if topN[i].Code == code {
                stock = &topN[i]
                break
            }
        }

        if stock == nil {
            continue
        }

        target.Positions = append(target.Positions, contracts.TargetPosition{
            Code:      code,
            Name:      stock.Name,
            Weight:    weight,
            TargetQty: 0, // 계산 필요 (가격 정보 필요)
            Action:    contracts.ActionBuy,
            Reason:    c.getActionReason(stock),
        })
    }

    c.logger.WithFields(map[string]interface{}{
        "positions":    len(target.Positions),
        "total_weight": target.TotalWeight(),
        "cash":         target.Cash,
    }).Info("Portfolio constructed")

    return target, nil
}

func DefaultPortfolioConfig() PortfolioConfig {
    return PortfolioConfig{
        MaxPositions:  20,    // 최대 20 종목
        MaxWeight:     0.15,  // 최대 15%
        MinWeight:     0.03,  // 최소 3%
        CashReserve:   0.05,  // 5% 현금 보유
        TurnoverLimit: 0.30,  // 30% 회전율 제한
        WeightingMode: "equal", // 기본: 동일 비중
    }
}
```

---

## Weight 계산 방식

### Option 1: Equal Weight (동일 비중)

```go
func (c *Constructor) equalWeight(stocks []contracts.RankedStock) map[string]float64 {
    available := 1.0 - c.config.CashReserve
    weight := available / float64(len(stocks))

    weights := make(map[string]float64)
    for _, stock := range stocks {
        weights[stock.Code] = weight
    }

    return weights
}
```

### Option 2: Score-Based (점수 비례)

```go
func (c *Constructor) scoreBasedWeight(stocks []contracts.RankedStock) map[string]float64 {
    available := 1.0 - c.config.CashReserve

    // Calculate total score (only positive scores)
    var totalScore float64
    for _, stock := range stocks {
        // Normalize score to 0 ~ 1 (from -1 ~ 1)
        normalizedScore := (stock.TotalScore + 1.0) / 2.0
        if normalizedScore > 0 {
            totalScore += normalizedScore
        }
    }

    if totalScore == 0 {
        // Fallback to equal weight
        return c.equalWeight(stocks)
    }

    weights := make(map[string]float64)
    for _, stock := range stocks {
        normalizedScore := (stock.TotalScore + 1.0) / 2.0
        if normalizedScore > 0 {
            weights[stock.Code] = (normalizedScore / totalScore) * available
        }
    }

    return weights
}
```

### Option 3: Risk Parity (리스크 패리티)

```go
func (c *Constructor) riskParityWeight(stocks []contracts.RankedStock) map[string]float64 {
    // TODO: Implement risk parity (변동성 필요)
    c.logger.Warn("Risk parity not implemented, using equal weight")
    return c.equalWeight(stocks)
}
```

> Risk Parity는 변동성 데이터가 필요하므로 향후 구현 예정

---

## 제약조건

```go
// internal/portfolio/constraints.go

type Constraints struct {
    MaxSectorWeight float64  // 섹터당 최대 비중 (0.0 ~ 1.0)
    MaxWeight       float64  // 종목당 최대 비중 (0.0 ~ 1.0)
    MinWeight       float64  // 종목당 최소 비중 (0.0 ~ 1.0)
    BlackList       []string // 제외 종목 리스트
}

func (c *Constraints) IsBlackListed(code string) bool {
    return slices.Contains(c.BlackList, code)
}

func DefaultConstraints() Constraints {
    return Constraints{
        MaxSectorWeight: 0.30, // 섹터당 최대 30%
        MaxWeight:       0.15, // 종목당 최대 15%
        MinWeight:       0.03, // 종목당 최소 3%
        BlackList:       []string{},
    }
}

// applyConstraints applies portfolio constraints to weights
func (c *Constructor) applyConstraints(weights map[string]float64) map[string]float64 {
    result := make(map[string]float64)

    for code, weight := range weights {
        // Apply max weight constraint
        if weight > c.constraints.MaxWeight {
            weight = c.constraints.MaxWeight
        }

        // Apply min weight constraint
        if weight < c.constraints.MinWeight {
            continue // 제외
        }

        // Check blacklist
        if c.constraints.IsBlackListed(code) {
            continue
        }

        result[code] = weight
    }

    // Normalize weights to sum to (1.0 - CashReserve)
    return c.normalizeWeights(result)
}

// normalizeWeights normalizes weights to sum to target total
func (c *Constructor) normalizeWeights(weights map[string]float64) map[string]float64 {
    targetTotal := 1.0 - c.config.CashReserve

    // Calculate current total
    var currentTotal float64
    for _, weight := range weights {
        currentTotal += weight
    }

    if currentTotal == 0 {
        return weights
    }

    // Scale weights
    factor := targetTotal / currentTotal
    normalized := make(map[string]float64)
    for code, weight := range weights {
        normalized[code] = weight * factor
    }

    return normalized
}
```

---

## Portfolio Repository

```go
// internal/portfolio/repository.go

type Repository struct {
    pool *pgxpool.Pool
}

func (r *Repository) SaveTargetPortfolio(ctx context.Context, target *contracts.TargetPortfolio) error {
    tx, err := r.pool.Begin(ctx)
    defer tx.Rollback(ctx)

    // Delete existing positions for the date
    _, err = tx.Exec(ctx, "DELETE FROM portfolio.target_positions WHERE target_date = $1", target.Date)

    // Insert new positions
    query := `
        INSERT INTO portfolio.target_positions (
            target_date, stock_code, stock_name, weight, target_qty, action, reason
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

    for _, pos := range target.Positions {
        _, err := tx.Exec(ctx, query,
            target.Date, pos.Code, pos.Name, pos.Weight, pos.TargetQty, pos.Action, pos.Reason,
        )
    }

    // Save portfolio summary
    summaryQuery := `
        INSERT INTO portfolio.portfolio_snapshots (
            snapshot_date, total_positions, total_weight, cash_reserve, created_at
        ) VALUES ($1, $2, $3, $4, NOW())
        ON CONFLICT (snapshot_date) DO UPDATE SET
            total_positions = EXCLUDED.total_positions,
            total_weight = EXCLUDED.total_weight,
            cash_reserve = EXCLUDED.cash_reserve,
            created_at = NOW()
    `

    _, err = tx.Exec(ctx, summaryQuery,
        target.Date, len(target.Positions), target.TotalWeight(), target.Cash,
    )

    return tx.Commit(ctx)
}

func (r *Repository) GetTargetPortfolio(ctx context.Context, date time.Time) (*contracts.TargetPortfolio, error) {
    query := `
        SELECT stock_code, stock_name, weight, target_qty, action, reason
        FROM portfolio.target_positions
        WHERE target_date = $1
        ORDER BY weight DESC
    `

    rows, err := r.pool.Query(ctx, query, date)
    defer rows.Close()

    portfolio := &contracts.TargetPortfolio{
        Date:      date,
        Positions: make([]contracts.TargetPosition, 0),
    }

    for rows.Next() {
        var pos contracts.TargetPosition
        err := rows.Scan(&pos.Code, &pos.Name, &pos.Weight, &pos.TargetQty, &pos.Action, &pos.Reason)
        portfolio.Positions = append(portfolio.Positions, pos)
    }

    return portfolio, nil
}

func (r *Repository) SaveHoldings(ctx context.Context, date time.Time, holdings []Holding) error {
    // 현재 보유 종목 저장
}

func (r *Repository) GetCurrentHoldings(ctx context.Context, date time.Time) ([]Holding, error) {
    // 현재 보유 종목 조회
}

func (r *Repository) SaveRebalanceLog(ctx context.Context, log *RebalanceLog) error {
    // 리밸런싱 로그 저장
}
```

---

## DB 스키마

```sql
-- portfolio.target_positions: 목표 포트폴리오 포지션
CREATE TABLE portfolio.target_positions (
    id          SERIAL PRIMARY KEY,
    target_date DATE NOT NULL,
    stock_code  VARCHAR(10) NOT NULL,
    stock_name  VARCHAR(100),
    weight      DECIMAL(8,4),       -- 목표 비중
    target_qty  INT,                -- 목표 수량
    action      VARCHAR(10),        -- BUY, SELL, HOLD
    reason      TEXT,               -- 액션 사유
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(target_date, stock_code)
);

-- portfolio.portfolio_snapshots: 포트폴리오 스냅샷
CREATE TABLE portfolio.portfolio_snapshots (
    id              SERIAL PRIMARY KEY,
    snapshot_date   DATE NOT NULL UNIQUE,
    total_positions INT,
    total_weight    DECIMAL(8,4),
    cash_reserve    DECIMAL(8,4),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- portfolio.holdings: 현재 보유 종목
CREATE TABLE portfolio.holdings (
    id                  SERIAL PRIMARY KEY,
    holding_date        DATE NOT NULL,
    stock_code          VARCHAR(10) NOT NULL,
    stock_name          VARCHAR(100),
    quantity            INT NOT NULL,
    avg_price           DECIMAL(12,2),
    current_price       DECIMAL(12,2),
    market_value        DECIMAL(15,2),
    weight              DECIMAL(8,4),
    unrealized_pnl      DECIMAL(15,2),
    unrealized_pnl_pct  DECIMAL(8,4),
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(holding_date, stock_code)
);

-- portfolio.rebalance_logs: 리밸런싱 로그
CREATE TABLE portfolio.rebalance_logs (
    id                SERIAL PRIMARY KEY,
    rebalance_date    DATE NOT NULL,
    total_orders      INT,
    executed_orders   INT,
    failed_orders     INT,
    turnover          DECIMAL(8,4),     -- 회전율
    execution_time_ms BIGINT,
    metadata          JSONB,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_target_positions_date ON portfolio.target_positions(target_date);
CREATE INDEX idx_holdings_date ON portfolio.holdings(holding_date, stock_code);
CREATE INDEX idx_rebalance_logs_date ON portfolio.rebalance_logs(rebalance_date);
```

---

**Prev**: [Selection Layer](./selection-layer.md)
**Next**: [Execution Layer](./execution-layer.md)
