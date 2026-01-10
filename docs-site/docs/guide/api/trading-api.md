---
sidebar_position: 6
title: Trading API
description: KIS 직접 거래 API (잔고, 주문, 실시간)
---

# Trading API

> KIS (한국투자증권) Open API를 직접 사용하는 거래 API

---

## Overview

Execution API가 전략 기반 주문 계획/실행을 다룬다면, Trading API는 KIS API를 직접 래핑하여 실시간 거래 기능을 제공합니다.

| 구분 | Execution API | Trading API |
|------|---------------|-------------|
| 목적 | 전략 기반 리밸런싱 | 직접 거래 |
| 주문 방식 | 배치 (계획 → 실행) | 개별 주문 |
| 실시간 | 미지원 | WebSocket 지원 |

---

## 엔드포인트 목록

### 잔고/포지션

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/api/trading/balance` | 잔고 및 보유종목 |
| `GET` | `/api/trading/positions` | 보유종목만 조회 |

### 주문

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/api/trading/orders` | 주문 목록 조회 |
| `GET` | `/api/trading/orders/unfilled` | 미체결 주문 |
| `GET` | `/api/trading/orders/filled` | 체결 주문 |
| `POST` | `/api/trading/orders` | 주문 실행 |
| `DELETE` | `/api/trading/orders` | 주문 취소 |

### 시세

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/api/trading/price` | 현재가 조회 |

### WebSocket

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/api/trading/ws/status` | WS 연결 상태 |
| `POST` | `/api/trading/ws/subscribe` | 실시간 구독 |
| `POST` | `/api/trading/ws/unsubscribe` | 구독 해제 |

---

## GET /api/trading/balance

잔고 및 보유종목 조회

### Request

```bash
curl http://localhost:8080/api/trading/balance
```

### Response

```json
{
  "balance": {
    "total_deposit": 10000000,
    "available_cash": 5000000,
    "total_purchase": 5000000,
    "total_evaluation": 5230000,
    "total_profit_loss": 230000,
    "profit_loss_rate": 4.6,
    "total_asset": 10230000
  },
  "positions": [
    {
      "stock_code": "005930",
      "stock_name": "삼성전자",
      "quantity": 10,
      "available_quantity": 10,
      "avg_buy_price": 72000,
      "current_price": 73500,
      "eval_amount": 735000,
      "profit_loss": 15000,
      "profit_loss_rate": 2.08
    }
  ]
}
```

### Balance 필드 설명

| 필드 | 타입 | 설명 |
|------|------|------|
| `total_deposit` | int64 | 예수금 총액 |
| `available_cash` | int64 | 출금 가능 금액 |
| `total_purchase` | int64 | 매입 금액 합계 |
| `total_evaluation` | int64 | 평가 금액 합계 |
| `total_profit_loss` | int64 | 평가 손익 합계 |
| `profit_loss_rate` | float64 | 수익률 (%) |
| `total_asset` | int64 | 총 자산 |

---

## GET /api/trading/positions

보유종목만 조회

### Request

```bash
curl http://localhost:8080/api/trading/positions
```

### Response

```json
{
  "positions": [
    {
      "stock_code": "005930",
      "stock_name": "삼성전자",
      "quantity": 10,
      "available_quantity": 10,
      "avg_buy_price": 72000,
      "current_price": 73500,
      "eval_amount": 735000,
      "profit_loss": 15000,
      "profit_loss_rate": 2.08
    }
  ],
  "count": 1
}
```

---

## GET /api/trading/orders

기간별 주문 목록 조회

### Request

```bash
curl "http://localhost:8080/api/trading/orders?start_date=20240101&end_date=20240131"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `start_date` | string | No | 시작일 (YYYYMMDD) |
| `end_date` | string | No | 종료일 (YYYYMMDD) |

### Response

```json
{
  "orders": [
    {
      "order_no": "0001234567",
      "stock_code": "005930",
      "stock_name": "삼성전자",
      "order_side": "buy",
      "order_price": 72000,
      "order_quantity": 10,
      "executed_quantity": 10,
      "executed_price": 72000,
      "remaining_qty": 0,
      "status": "filled",
      "order_time": "093015",
      "order_date": "20240115"
    }
  ],
  "count": 1
}
```

---

## GET /api/trading/orders/unfilled

미체결 주문 조회

### Request

```bash
curl http://localhost:8080/api/trading/orders/unfilled
```

### Response

```json
{
  "orders": [
    {
      "order_no": "0001234568",
      "stock_code": "000660",
      "stock_name": "SK하이닉스",
      "order_side": "buy",
      "order_price": 130000,
      "order_quantity": 5,
      "executed_quantity": 0,
      "remaining_qty": 5,
      "status": "pending"
    }
  ],
  "count": 1
}
```

---

## GET /api/trading/orders/filled

체결 완료 주문 조회

### Request

```bash
curl http://localhost:8080/api/trading/orders/filled
```

### Response

```json
{
  "orders": [
    {
      "order_no": "0001234567",
      "stock_code": "005930",
      "stock_name": "삼성전자",
      "order_side": "buy",
      "executed_quantity": 10,
      "executed_price": 72000,
      "status": "filled"
    }
  ],
  "count": 1
}
```

---

## POST /api/trading/orders

주문 실행

### Request

```bash
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{
       "stock_code": "005930",
       "side": "buy",
       "type": "limit",
       "quantity": 10,
       "price": 72000
     }' \
     http://localhost:8080/api/trading/orders
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `stock_code` | string | Yes | 종목 코드 (6자리) |
| `side` | string | Yes | `buy` 또는 `sell` |
| `type` | string | Yes | `limit` (지정가) 또는 `market` (시장가) |
| `quantity` | int64 | Yes | 주문 수량 |
| `price` | int64 | Yes | 주문 가격 (시장가는 0) |

### Response

```json
{
  "order_no": "0001234569",
  "message": "주문이 접수되었습니다"
}
```

### 주문 유형

| 유형 | 설명 | price 값 |
|------|------|----------|
| `limit` | 지정가 주문 | 주문 가격 |
| `market` | 시장가 주문 | 0 |

---

## DELETE /api/trading/orders

주문 취소

### Request

```bash
curl -X DELETE \
     "http://localhost:8080/api/trading/orders?order_no=0001234569"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `order_no` | string | Yes | 주문 번호 |

### Response

```json
{
  "order_no": "0001234569",
  "message": "주문이 취소되었습니다"
}
```

---

## GET /api/trading/price

현재가 조회

### Request

```bash
curl "http://localhost:8080/api/trading/price?stock_code=005930"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `stock_code` | string | Yes | 종목 코드 |

### Response

```json
{
  "stock_code": "005930",
  "stock_name": "삼성전자",
  "current_price": 73500,
  "change": 500,
  "change_rate": 0.68,
  "volume": 12345678,
  "high_price": 74000,
  "low_price": 72800,
  "open_price": 73000
}
```

---

## GET /api/trading/ws/status

WebSocket 연결 상태 조회

### Request

```bash
curl http://localhost:8080/api/trading/ws/status
```

### Response

```json
{
  "connected": true,
  "subscriptions": ["005930", "000660", "035720"],
  "count": 3
}
```

---

## POST /api/trading/ws/subscribe

실시간 시세 구독

### Request

```bash
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{
       "symbols": ["005930", "000660", "035720"]
     }' \
     http://localhost:8080/api/trading/ws/subscribe
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `symbols` | []string | Yes | 구독할 종목 코드 리스트 |

### Response

```json
{
  "status": "subscribed",
  "symbols": ["005930", "000660", "035720"],
  "subscriptions": ["005930", "000660", "035720"]
}
```

### 제한 사항

- 세션당 최대 **41개** 종목 구독 가능
- 이미 구독 중인 종목은 중복 추가되지 않음

---

## POST /api/trading/ws/unsubscribe

실시간 시세 구독 해제

### Request

```bash
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{
       "symbols": ["035720"]
     }' \
     http://localhost:8080/api/trading/ws/unsubscribe
```

### Response

```json
{
  "status": "unsubscribed",
  "symbols": ["035720"],
  "subscriptions": ["005930", "000660"]
}
```

---

## 주문 상태 (Status)

| 상태 | 설명 |
|------|------|
| `pending` | 대기 중 |
| `partial` | 부분 체결 |
| `filled` | 전량 체결 |
| `cancelled` | 취소됨 |

---

## 에러 응답

### 형식

```json
{
  "error": "에러 메시지"
}
```

### 에러 코드

| HTTP | 메시지 | 설명 |
|------|--------|------|
| 400 | `stock_code is required` | 종목 코드 누락 |
| 400 | `side must be 'buy' or 'sell'` | 잘못된 주문 방향 |
| 400 | `type must be 'limit' or 'market'` | 잘못된 주문 유형 |
| 400 | `quantity must be positive` | 수량이 0 이하 |
| 400 | `price is required for limit orders` | 지정가 주문 시 가격 누락 |
| 500 | `Failed to retrieve balance` | 잔고 조회 실패 |
| 500 | `Failed to place order` | 주문 실행 실패 |
| 503 | `WebSocket client not initialized` | WS 클라이언트 미설정 |

---

## 환경 설정

Trading API 사용을 위한 환경 변수:

```bash
# KIS API 설정
KIS_APP_KEY=your-app-key
KIS_APP_SECRET=your-app-secret
KIS_ACCOUNT_NO=12345678-01
KIS_BASE_URL=https://openapi.koreainvestment.com:9443
KIS_IS_VIRTUAL=false  # true: 모의투자, false: 실전
KIS_HTS_ID=your-hts-id  # WebSocket 체결통보용
```

---

## 사용 예시

### cURL: 매수 주문

```bash
# 삼성전자 10주 72,000원 지정가 매수
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{
       "stock_code": "005930",
       "side": "buy",
       "type": "limit",
       "quantity": 10,
       "price": 72000
     }' \
     http://localhost:8080/api/trading/orders
```

### cURL: 시장가 매도

```bash
# SK하이닉스 5주 시장가 매도
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{
       "stock_code": "000660",
       "side": "sell",
       "type": "market",
       "quantity": 5,
       "price": 0
     }' \
     http://localhost:8080/api/trading/orders
```

### cURL: 실시간 구독

```bash
# 구독 시작
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"symbols": ["005930", "000660"]}' \
     http://localhost:8080/api/trading/ws/subscribe

# 상태 확인
curl http://localhost:8080/api/trading/ws/status

# 구독 해제
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"symbols": ["000660"]}' \
     http://localhost:8080/api/trading/ws/unsubscribe
```

---

**Prev**: [Execution API](./execution-api.md)
**Next**: [Audit API](./audit-api.md)
