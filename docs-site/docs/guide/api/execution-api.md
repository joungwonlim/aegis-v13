---
sidebar_position: 5
title: Execution API
description: S6 주문 실행 API
---

# Execution API

> S6 주문 실행 및 모니터링 API

---

## 엔드포인트 목록

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `POST` | `/execution/plan` | 실행 계획 생성 |
| `POST` | `/execution/orders` | 주문 제출 |
| `GET` | `/execution/orders/:id` | 주문 상태 조회 |
| `DELETE` | `/execution/orders/:id` | 주문 취소 |
| `GET` | `/execution/orders` | 주문 목록 조회 |
| `GET` | `/execution/balance` | 잔고 조회 |

---

## POST /execution/plan

실행 계획 생성 (주문 시뮬레이션)

### Request

```bash
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "date": "2024-01-15",
       "target_portfolio_id": "port_20240115_001",
       "config": {
         "order_type": "LIMIT",
         "slippage_bps": 10,
         "max_order_size": 50000000
       }
     }' \
     http://localhost:8080/api/v1/execution/plan
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `date` | string | Yes | 실행 날짜 |
| `target_portfolio_id` | string | Yes | 목표 포트폴리오 ID |
| `config` | object | No | 실행 설정 |

### Config Object

| 필드 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| `order_type` | string | "LIMIT" | 주문 유형: "LIMIT", "MARKET" |
| `slippage_bps` | int | 10 | 슬리피지 (10 = 0.1%) |
| `max_order_size` | int64 | 50000000 | 최대 주문 금액 (5천만원) |
| `split_threshold` | int64 | 100000000 | 분할 주문 기준 (1억원) |

### Response

```json
{
  "success": true,
  "data": {
    "plan_id": "exec_plan_20240115_143052",
    "total_orders": 13,
    "orders": [
      {
        "code": "005930",
        "name": "삼성전자",
        "side": "BUY",
        "quantity": 12,
        "price": 73300,
        "order_type": "LIMIT",
        "estimated_value": 879600,
        "priority": 1,
        "reason": "Increase position"
      },
      {
        "code": "035720",
        "name": "카카오",
        "side": "SELL",
        "quantity": 85,
        "price": 49950,
        "order_type": "LIMIT",
        "estimated_value": 4245750,
        "priority": 2,
        "reason": "Exit position"
      },
      ...
    ],
    "summary": {
      "buy_orders": 8,
      "sell_orders": 5,
      "total_buy_value": 15000000,
      "total_sell_value": 12000000,
      "net_cash_flow": -3000000
    }
  }
}
```

---

## POST /execution/orders

주문 제출 (실제 실행)

### Request

```bash
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "orders": [
         {
           "code": "005930",
           "side": "BUY",
           "quantity": 12,
           "price": 73300,
           "order_type": "LIMIT"
         },
         {
           "code": "035720",
           "side": "SELL",
           "quantity": 85,
           "price": 49950,
           "order_type": "LIMIT"
         }
       ],
       "dry_run": false
     }' \
     http://localhost:8080/api/v1/execution/orders
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `orders` | []object | Yes | 주문 리스트 |
| `dry_run` | bool | No | 시뮬레이션 모드 (기본: false) |

### Order Object

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `code` | string | Yes | 종목 코드 |
| `side` | string | Yes | 주문 방향: "BUY", "SELL" |
| `quantity` | int | Yes | 수량 |
| `price` | int | Yes | 가격 (0 = 시장가) |
| `order_type` | string | Yes | "LIMIT", "MARKET" |

### Response

```json
{
  "success": true,
  "data": {
    "batch_id": "batch_20240115_143052",
    "submitted": 13,
    "failed": 0,
    "orders": [
      {
        "order_id": "ORD-20240115-001",
        "code": "005930",
        "side": "BUY",
        "quantity": 12,
        "price": 73300,
        "status": "SUBMITTED",
        "submitted_at": "2024-01-15T14:30:52Z"
      },
      ...
    ]
  }
}
```

---

## GET /execution/orders/:id

주문 상태 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     http://localhost:8080/api/v1/execution/orders/ORD-20240115-001
```

### Response

```json
{
  "success": true,
  "data": {
    "order_id": "ORD-20240115-001",
    "code": "005930",
    "name": "삼성전자",
    "side": "BUY",
    "quantity": 12,
    "price": 73300,
    "order_type": "LIMIT",
    "status": "PARTIALLY_FILLED",
    "filled_quantity": 8,
    "remaining_quantity": 4,
    "avg_filled_price": 73250,
    "submitted_at": "2024-01-15T14:30:52Z",
    "updated_at": "2024-01-15T14:31:15Z",
    "executions": [
      {
        "exec_id": "EXEC-001",
        "quantity": 5,
        "price": 73200,
        "executed_at": "2024-01-15T14:31:05Z"
      },
      {
        "exec_id": "EXEC-002",
        "quantity": 3,
        "price": 73300,
        "executed_at": "2024-01-15T14:31:15Z"
      }
    ]
  }
}
```

---

## DELETE /execution/orders/:id

주문 취소

### Request

```bash
curl -X DELETE \
     -H "Authorization: Bearer YOUR_KEY" \
     http://localhost:8080/api/v1/execution/orders/ORD-20240115-001
```

### Response

```json
{
  "success": true,
  "data": {
    "order_id": "ORD-20240115-001",
    "status": "CANCELLED",
    "cancelled_at": "2024-01-15T14:32:00Z",
    "message": "Order cancelled successfully"
  }
}
```

---

## GET /execution/orders

주문 목록 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/execution/orders?date=2024-01-15&status=FILLED"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `date` | string | No | 날짜 (기본: 오늘) |
| `status` | string | No | 상태 필터 |
| `side` | string | No | 방향 필터: "BUY", "SELL" |
| `limit` | int | No | 개수 (기본: 100) |

### Response

```json
{
  "success": true,
  "data": {
    "total": 13,
    "orders": [
      {
        "order_id": "ORD-20240115-001",
        "code": "005930",
        "side": "BUY",
        "quantity": 12,
        "status": "FILLED",
        "avg_filled_price": 73250,
        "submitted_at": "2024-01-15T14:30:52Z"
      },
      ...
    ]
  }
}
```

---

## GET /execution/balance

계좌 잔고 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     http://localhost:8080/api/v1/execution/balance
```

### Response

```json
{
  "success": true,
  "data": {
    "account_no": "12345678-01",
    "total_value": 105230000,
    "cash": 5230000,
    "stock_value": 100000000,
    "buying_power": 8000000,
    "holdings": [
      {
        "code": "005930",
        "name": "삼성전자",
        "quantity": 138,
        "avg_price": 72500,
        "current_price": 73200,
        "market_value": 10101600,
        "pnl": 96600,
        "pnl_pct": 0.97
      },
      ...
    ],
    "updated_at": "2024-01-15T14:35:00Z"
  }
}
```

---

## 주문 상태 (Status)

| 상태 | 설명 |
|------|------|
| `PENDING` | 대기 중 |
| `SUBMITTED` | 제출됨 |
| `ACCEPTED` | 증권사 접수 |
| `PARTIALLY_FILLED` | 부분 체결 |
| `FILLED` | 전량 체결 |
| `CANCELLED` | 취소됨 |
| `REJECTED` | 거부됨 |
| `EXPIRED` | 만료됨 |
| `FAILED` | 실패 |

---

## 주문 유형 (Order Type)

### LIMIT (지정가)

```json
{
  "order_type": "LIMIT",
  "price": 73300
}
```

- 지정한 가격에만 체결
- 슬리피지 반영:
  - 매수: `current_price * (1 + slippage_bps/10000)`
  - 매도: `current_price * (1 - slippage_bps/10000)`

### MARKET (시장가)

```json
{
  "order_type": "MARKET",
  "price": 0
}
```

- 현재가에 즉시 체결
- 빠른 체결, 슬리피지 위험 있음

---

## 주문 분할

대량 주문 시 자동 분할:

### 분할 기준

```
IF order_value > split_threshold:
    split into chunks of max_order_size
```

### 예시

```
종목: 삼성전자
수량: 2000주
가격: 73,000원
총액: 146,000,000원 (1억 4천 6백만원)

split_threshold = 100,000,000원 (1억)
max_order_size = 50,000,000원 (5천만)

→ 3개 주문으로 분할:
  1. 685주 (49,905,000원)
  2. 685주 (49,905,000원)
  3. 630주 (45,990,000원)
```

---

## 에러 코드

| 코드 | 설명 | 해결 |
|------|------|------|
| `INSUFFICIENT_FUNDS` | 잔고 부족 | 매도 먼저 실행 또는 금액 조정 |
| `ORDER_REJECTED` | 증권사 거부 | 주문 조건 확인 |
| `INVALID_PRICE` | 잘못된 가격 | 가격 호가 단위 확인 |
| `MARKET_CLOSED` | 장 마감 | 장 시간 확인 |
| `STOCK_SUSPENDED` | 거래정지 종목 | 종목 상태 확인 |

---

## 사용 예시

### Python: 리밸런싱 실행

```python
import requests

API_KEY = "YOUR_KEY"
BASE_URL = "http://localhost:8080/api/v1"
headers = {"Authorization": f"Bearer {API_KEY}"}

# 1. 실행 계획 생성
plan_payload = {
    "date": "2024-01-15",
    "target_portfolio_id": "port_20240115_001",
    "config": {
        "order_type": "LIMIT",
        "slippage_bps": 10
    }
}

response = requests.post(
    f"{BASE_URL}/execution/plan",
    json=plan_payload,
    headers=headers
)
plan = response.json()["data"]
print(f"Total orders: {plan['total_orders']}")

# 2. 주문 제출 (실제 실행)
orders_payload = {
    "orders": plan["orders"],
    "dry_run": False
}

response = requests.post(
    f"{BASE_URL}/execution/orders",
    json=orders_payload,
    headers=headers
)
result = response.json()["data"]
print(f"Submitted: {result['submitted']}, Failed: {result['failed']}")

# 3. 주문 상태 모니터링
import time

order_ids = [order["order_id"] for order in result["orders"]]
for order_id in order_ids:
    response = requests.get(
        f"{BASE_URL}/execution/orders/{order_id}",
        headers=headers
    )
    order = response.json()["data"]
    print(f"{order['code']} {order['side']}: {order['status']}")

    # FILLED 될 때까지 대기
    while order["status"] not in ["FILLED", "CANCELLED", "REJECTED"]:
        time.sleep(5)
        response = requests.get(
            f"{BASE_URL}/execution/orders/{order_id}",
            headers=headers
        )
        order = response.json()["data"]

print("All orders processed")
```

### Go: 주문 실행 및 모니터링

```go
client := aegis.NewClient("YOUR_KEY")

// 1. 실행 계획
plan, err := client.Execution.CreatePlan(ctx, &ExecutionPlanRequest{
    Date:              "2024-01-15",
    TargetPortfolioID: "port_20240115_001",
    Config: ExecutionConfig{
        OrderType:    "LIMIT",
        SlippageBps:  10,
        MaxOrderSize: 50000000,
    },
})
if err != nil {
    log.Fatal(err)
}

// 2. 주문 제출
result, err := client.Execution.SubmitOrders(ctx, &OrdersRequest{
    Orders: plan.Orders,
    DryRun: false,
})
if err != nil {
    log.Fatal(err)
}

// 3. 모니터링
for _, orderID := range result.OrderIDs {
    go func(id string) {
        for {
            order, err := client.Execution.GetOrder(ctx, id)
            if err != nil {
                log.Println("Error:", err)
                return
            }

            if order.Status == "FILLED" {
                log.Printf("%s filled at %d\n", order.Code, order.AvgFilledPrice)
                return
            }

            time.Sleep(5 * time.Second)
        }
    }(orderID)
}
```

---

**Prev**: [Portfolio API](./portfolio-api.md)
**Next**: [Audit API](./audit-api.md)
