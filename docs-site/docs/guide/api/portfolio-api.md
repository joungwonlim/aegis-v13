---
sidebar_position: 4
title: Portfolio API
description: S5 포트폴리오 구성 API
---

# Portfolio API

> S5 포트폴리오 구성 및 관리 API

---

## 엔드포인트 목록

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/portfolio/target` | 목표 포트폴리오 조회 |
| `GET` | `/portfolio/holdings` | 현재 보유 종목 조회 |
| `GET` | `/portfolio/diff` | 리밸런싱 차이 계산 |
| `POST` | `/portfolio/construct` | 포트폴리오 구성 실행 |

---

## GET /portfolio/target

목표 포트폴리오 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/portfolio/target?date=2024-01-15"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `date` | string | No | 날짜 (기본: 최신) |

### Response

```json
{
  "success": true,
  "data": {
    "date": "2024-01-15",
    "total_positions": 20,
    "total_weight": 0.95,
    "cash_reserve": 0.05,
    "positions": [
      {
        "code": "005930",
        "name": "삼성전자",
        "weight": 0.08,
        "target_qty": 15,
        "action": "BUY",
        "reason": "Rank 1, Strong flow signal (0.92)"
      },
      {
        "code": "000660",
        "name": "SK하이닉스",
        "weight": 0.075,
        "target_qty": 8,
        "action": "BUY",
        "reason": "Rank 2, High momentum (0.78)"
      },
      ...
    ],
    "config": {
      "max_positions": 20,
      "max_weight": 0.15,
      "min_weight": 0.03,
      "weighting_mode": "equal"
    }
  }
}
```

---

## GET /portfolio/holdings

현재 보유 종목 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/portfolio/holdings?date=2024-01-15"
```

### Response

```json
{
  "success": true,
  "data": {
    "date": "2024-01-15",
    "total_value": 100000000,
    "cash": 5000000,
    "holdings": [
      {
        "code": "005930",
        "name": "삼성전자",
        "quantity": 138,
        "avg_price": 72500,
        "current_price": 73200,
        "market_value": 10101600,
        "weight": 0.101,
        "unrealized_pnl": 96600,
        "unrealized_pnl_pct": 0.0097
      },
      ...
    ]
  }
}
```

---

## GET /portfolio/diff

리밸런싱 차이 계산

목표 포트폴리오와 현재 보유 종목의 차이를 계산

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/portfolio/diff?date=2024-01-15"
```

### Response

```json
{
  "success": true,
  "data": {
    "date": "2024-01-15",
    "turnover": 0.32,
    "actions": [
      {
        "code": "005930",
        "name": "삼성전자",
        "action": "BUY",
        "current_qty": 138,
        "target_qty": 150,
        "diff_qty": 12,
        "diff_value": 878400,
        "reason": "Increase weight 10.1% → 15%"
      },
      {
        "code": "035720",
        "name": "카카오",
        "action": "SELL",
        "current_qty": 85,
        "target_qty": 0,
        "diff_qty": -85,
        "diff_value": -4250000,
        "reason": "Exit: Dropped from top 20"
      },
      {
        "code": "000660",
        "name": "SK하이닉스",
        "action": "HOLD",
        "current_qty": 80,
        "target_qty": 80,
        "diff_qty": 0,
        "diff_value": 0,
        "reason": "Maintain position"
      }
    ],
    "summary": {
      "buy_count": 8,
      "sell_count": 5,
      "hold_count": 7,
      "total_buy_value": 15000000,
      "total_sell_value": 12000000
    }
  }
}
```

---

## POST /portfolio/construct

포트폴리오 구성 실행

### Request

```bash
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "date": "2024-01-15",
       "config": {
         "max_positions": 20,
         "max_weight": 0.15,
         "min_weight": 0.03,
         "cash_reserve": 0.05,
         "weighting_mode": "equal"
       },
       "constraints": {
         "max_sector_weight": 0.30,
         "blacklist": ["000000", "111111"]
       }
     }' \
     http://localhost:8080/api/v1/portfolio/construct
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `date` | string | Yes | 구성 날짜 |
| `config` | object | No | 포트폴리오 설정 |
| `constraints` | object | No | 제약조건 |

### Config Object

| 필드 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| `max_positions` | int | 20 | 최대 종목 수 |
| `max_weight` | float | 0.15 | 종목당 최대 비중 (15%) |
| `min_weight` | float | 0.03 | 종목당 최소 비중 (3%) |
| `cash_reserve` | float | 0.05 | 현금 보유 비중 (5%) |
| `weighting_mode` | string | "equal" | 비중 방식: "equal", "score_based", "risk_parity" |

### Constraints Object

| 필드 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| `max_sector_weight` | float | 0.30 | 섹터당 최대 비중 (30%) |
| `blacklist` | []string | [] | 제외 종목 리스트 |

### Response

```json
{
  "success": true,
  "data": {
    "job_id": "portfolio_construct_20240115_143052",
    "status": "COMPLETED",
    "result": {
      "total_positions": 20,
      "total_weight": 0.95,
      "cash_reserve": 0.05,
      "positions": [ ... ]
    }
  }
}
```

---

## Weighting Modes

### 1. Equal Weight (equal)

모든 종목에 동일한 비중 할당

```
available = 1.0 - cash_reserve
weight_per_stock = available / total_stocks

예: 20개 종목, 5% 현금보유
→ 각 종목 4.75% (0.95 / 20)
```

### 2. Score-Based (score_based)

시그널 점수에 비례하여 비중 할당

```
normalized_score = (total_score + 1.0) / 2.0  // -1~1 → 0~1
weight = (normalized_score / sum_of_scores) * available

예:
종목 A: score 0.8 → normalized 0.9
종목 B: score 0.6 → normalized 0.8
→ A가 B보다 높은 비중
```

### 3. Risk Parity (risk_parity)

변동성 역비례 비중 할당 (향후 구현 예정)

```
weight_i = (1/volatility_i) / sum(1/volatility)

변동성 낮은 종목에 더 높은 비중
```

---

## 제약조건 적용

### Max Weight Constraint

```
IF weight > max_weight:
    weight = max_weight
```

### Min Weight Constraint

```
IF weight < min_weight:
    exclude from portfolio
```

### Sector Weight Constraint

```
sector_total = sum(weights in sector)
IF sector_total > max_sector_weight:
    scale down proportionally
```

---

## 에러 코드

| 코드 | 설명 | 해결 |
|------|------|------|
| `NO_RANKED_STOCKS` | 랭킹 데이터 없음 | 랭킹 먼저 실행 |
| `INVALID_WEIGHTS` | 잘못된 가중치 설정 | 0.0 ~ 1.0 범위 확인 |
| `INSUFFICIENT_STOCKS` | 종목 수 부족 | min_weight 또는 필터 조정 |
| `CONSTRAINT_VIOLATION` | 제약조건 위반 | 제약조건 확인 |

---

## 사용 예시

### Python: 포트폴리오 구성 및 리밸런싱

```python
import requests

API_KEY = "YOUR_KEY"
BASE_URL = "http://localhost:8080/api/v1"
headers = {"Authorization": f"Bearer {API_KEY}"}

# 1. 포트폴리오 구성
construct_payload = {
    "date": "2024-01-15",
    "config": {
        "max_positions": 20,
        "weighting_mode": "score_based",
        "cash_reserve": 0.05
    }
}

response = requests.post(
    f"{BASE_URL}/portfolio/construct",
    json=construct_payload,
    headers=headers
)
target = response.json()["data"]["result"]
print(f"Target Portfolio: {target['total_positions']} positions")

# 2. 현재 보유 종목 조회
response = requests.get(
    f"{BASE_URL}/portfolio/holdings?date=2024-01-15",
    headers=headers
)
holdings = response.json()["data"]["holdings"]

# 3. 리밸런싱 차이 계산
response = requests.get(
    f"{BASE_URL}/portfolio/diff?date=2024-01-15",
    headers=headers
)
diff = response.json()["data"]
print(f"Turnover: {diff['turnover']:.2%}")

# 4. 매수/매도 액션 추출
buy_actions = [a for a in diff["actions"] if a["action"] == "BUY"]
sell_actions = [a for a in diff["actions"] if a["action"] == "SELL"]

print(f"BUY: {len(buy_actions)} stocks")
print(f"SELL: {len(sell_actions)} stocks")
```

### Go: 포트폴리오 비교

```go
client := aegis.NewClient("YOUR_KEY")

// Target portfolio
target, err := client.Portfolio.GetTarget(ctx, "2024-01-15")
if err != nil {
    log.Fatal(err)
}

// Current holdings
holdings, err := client.Portfolio.GetHoldings(ctx, "2024-01-15")
if err != nil {
    log.Fatal(err)
}

// Calculate rebalancing needs
diff, err := client.Portfolio.GetDiff(ctx, "2024-01-15")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Turnover: %.2f%%\n", diff.Turnover*100)
for _, action := range diff.Actions {
    if action.Action == "BUY" {
        fmt.Printf("BUY %s: %d shares\n", action.Code, action.DiffQty)
    }
}
```

---

**Prev**: [Selection API](./selection-api.md)
**Next**: [Execution API](./execution-api.md)
