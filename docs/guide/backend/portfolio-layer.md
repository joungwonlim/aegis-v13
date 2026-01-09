# Portfolio Layer

> S5: 포트폴리오 구성

---

## 책임

랭킹 결과를 바탕으로 **목표 포트폴리오** 산출

---

## 폴더 구조

```
internal/portfolio/
├── constructor.go   # PortfolioConstructor 구현
├── rebalancer.go    # 리밸런싱 로직
└── constraints.go   # 제약조건 검증
```

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

type constructor struct {
    config     PortfolioConfig
    current    CurrentPortfolio  // 현재 보유
    constraints Constraints
}

type PortfolioConfig struct {
    MaxPositions   int     `yaml:"max_positions"`    // 최대 종목 수
    MaxWeight      float64 `yaml:"max_weight"`       // 최대 비중 (20%)
    MinWeight      float64 `yaml:"min_weight"`       // 최소 비중 (2%)
    CashReserve    float64 `yaml:"cash_reserve"`     // 현금 보유 (5%)
    TurnoverLimit  float64 `yaml:"turnover_limit"`   // 회전율 제한 (30%)
}

func (c *constructor) Construct(ctx context.Context, ranked []contracts.RankedStock) (*contracts.TargetPortfolio, error) {
    target := &contracts.TargetPortfolio{
        Date:      time.Now(),
        Positions: make([]contracts.TargetPosition, 0),
        Cash:      c.config.CashReserve,
    }

    // 1. Top N 선택
    topN := c.selectTopN(ranked)

    // 2. 비중 계산 (점수 기반 가중)
    weights := c.calculateWeights(topN)

    // 3. 제약조건 적용
    weights = c.applyConstraints(weights)

    // 4. 현재 포트폴리오와 비교하여 Action 결정
    for code, weight := range weights {
        action := c.determineAction(code, weight)

        target.Positions = append(target.Positions, contracts.TargetPosition{
            Code:      code,
            Name:      c.getStockName(code),
            Weight:    weight,
            TargetQty: c.calculateQuantity(code, weight),
            Action:    action,
            Reason:    c.getActionReason(code, action),
        })
    }

    // 5. 매도 종목 추가 (현재 보유 중 목표에 없는 것)
    target.Positions = append(target.Positions, c.getSellPositions(weights)...)

    return target, nil
}

func (c *constructor) determineAction(code string, targetWeight float64) contracts.Action {
    currentWeight := c.current.GetWeight(code)

    if currentWeight == 0 {
        return contracts.ActionBuy
    }
    if targetWeight == 0 {
        return contracts.ActionSell
    }

    // 비중 변화가 미미하면 Hold
    diff := math.Abs(targetWeight - currentWeight)
    if diff < 0.02 { // 2% 미만
        return contracts.ActionHold
    }

    if targetWeight > currentWeight {
        return contracts.ActionBuy
    }
    return contracts.ActionSell
}
```

---

## Weight 계산 방식

### Option 1: Equal Weight (동일 비중)

```go
func (c *constructor) equalWeight(n int) map[string]float64 {
    available := 1.0 - c.config.CashReserve
    weight := available / float64(n)

    weights := make(map[string]float64)
    for _, stock := range topN {
        weights[stock.Code] = weight
    }
    return weights
}
```

### Option 2: Score-Based (점수 비례)

```go
func (c *constructor) scoreBasedWeight(ranked []contracts.RankedStock) map[string]float64 {
    available := 1.0 - c.config.CashReserve

    // 점수 합계
    var totalScore float64
    for _, s := range ranked {
        totalScore += s.TotalScore
    }

    weights := make(map[string]float64)
    for _, s := range ranked {
        weights[s.Code] = (s.TotalScore / totalScore) * available
    }
    return weights
}
```

### Option 3: Risk Parity (리스크 패리티)

```go
func (c *constructor) riskParityWeight(ranked []contracts.RankedStock) map[string]float64 {
    // 변동성 역수 비례
    available := 1.0 - c.config.CashReserve

    var totalInvVol float64
    for _, s := range ranked {
        vol := c.getVolatility(s.Code)
        totalInvVol += 1.0 / vol
    }

    weights := make(map[string]float64)
    for _, s := range ranked {
        vol := c.getVolatility(s.Code)
        weights[s.Code] = (1.0 / vol / totalInvVol) * available
    }
    return weights
}
```

---

## 제약조건

```go
// internal/portfolio/constraints.go

type Constraints struct {
    MaxSectorWeight float64            // 섹터당 최대 비중
    MaxWeight       float64            // 종목당 최대 비중
    MinWeight       float64            // 종목당 최소 비중
    BlackList       []string           // 매매 금지 종목
}

func (c *constructor) applyConstraints(weights map[string]float64) map[string]float64 {
    result := make(map[string]float64)

    for code, weight := range weights {
        // Max weight 제한
        if weight > c.constraints.MaxWeight {
            weight = c.constraints.MaxWeight
        }

        // Min weight 제한
        if weight < c.constraints.MinWeight {
            continue // 제외
        }

        // 블랙리스트 체크
        if c.isBlackListed(code) {
            continue
        }

        result[code] = weight
    }

    // 비중 합계 재조정
    return c.normalizeWeights(result)
}
```

---

## 설정 예시 (YAML)

```yaml
# config/portfolio.yaml

portfolio:
  max_positions: 20
  max_weight: 0.15       # 15%
  min_weight: 0.03       # 3%
  cash_reserve: 0.05     # 5%
  turnover_limit: 0.30   # 30%

  weighting: "score_based"  # equal, score_based, risk_parity

  constraints:
    max_sector_weight: 0.30   # 섹터당 30%
    black_list: []
```

---

## DB 스키마

```sql
-- portfolio.targets: 목표 포트폴리오
CREATE TABLE portfolio.targets (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL,
    positions   JSONB NOT NULL,     -- TargetPosition[]
    cash        DECIMAL(5,4),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- portfolio.holdings: 현재 보유
CREATE TABLE portfolio.holdings (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(10) NOT NULL,
    name        VARCHAR(100),
    quantity    INT NOT NULL,
    avg_price   DECIMAL(12,2),
    weight      DECIMAL(5,4),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
```

---

**Prev**: [Selection Layer](./selection-layer.md)
**Next**: [Execution Layer](./execution-layer.md)
