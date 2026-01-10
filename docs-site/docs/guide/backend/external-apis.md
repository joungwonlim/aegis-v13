---
sidebar_position: 10
title: External APIs
description: KIS, DART 등 외부 API 연동
---

# External APIs

> 외부 API 연동 계층: KIS (한국투자증권), DART (전자공시)

---

## Overview

| 패키지 | 역할 | 위치 |
|--------|------|------|
| **kis** | 주식 주문, 잔고, 실시간 시세 | `internal/external/kis/` |
| **dart** | 전자공시 수집 | `internal/external/dart/` |
| **naver** | 네이버 금융 데이터 | `internal/external/naver/` |

---

## KIS (한국투자증권) API

### 파일 구조

```
internal/external/kis/
├── client.go      # HTTP 클라이언트, 토큰 관리
├── types.go       # 타입 정의
├── balance.go     # 잔고/보유종목 조회
├── orders.go      # 주문 조회/실행/취소
└── websocket.go   # 실시간 시세 WebSocket
```

### 환경 변수

```bash
# KIS API 설정
KIS_APP_KEY=your-app-key
KIS_APP_SECRET=your-app-secret
KIS_ACCOUNT_NO=12345678-01
KIS_BASE_URL=https://openapi.koreainvestment.com:9443
KIS_IS_VIRTUAL=false  # true: 모의투자, false: 실전
KIS_HTS_ID=your-hts-id  # 체결통보 구독용
```

### 주요 기능

#### 1. 토큰 관리

```go
// 자동 토큰 갱신
client := kis.NewClient(cfg.KIS, httpClient, log)

// 내부적으로 토큰 캐싱 및 만료 전 자동 갱신
price, err := client.GetCurrentPrice(ctx, "005930")
```

#### 2. 잔고 조회

```go
// 잔고 및 보유종목 조회
balance, positions, err := client.GetBalance(ctx)

// Balance 구조체
type Balance struct {
    TotalDeposit     int64   // 예수금
    AvailableCash    int64   // 출금가능금액
    TotalPurchase    int64   // 매입금액합계
    TotalEvaluation  int64   // 평가금액합계
    TotalProfitLoss  int64   // 평가손익합계
    ProfitLossRate   float64 // 수익률
    TotalAsset       int64   // 총자산
}

// Position 구조체
type Position struct {
    StockCode         string  // 종목코드
    StockName         string  // 종목명
    Quantity          int64   // 보유수량
    AvailableQuantity int64   // 매도가능수량
    AvgBuyPrice       int64   // 평균매입가
    CurrentPrice      int64   // 현재가
    EvalAmount        int64   // 평가금액
    ProfitLoss        int64   // 평가손익
    ProfitLossRate    float64 // 수익률
}
```

#### 3. 주문 조회

```go
// 전체 주문 조회
orders, err := client.GetOrders(ctx, "20240101", "20240131")

// 미체결 주문만 조회
unfilledOrders, err := client.GetUnfilledOrders(ctx)

// 체결 주문만 조회
filledOrders, err := client.GetFilledOrders(ctx)

// Order 구조체
type Order struct {
    OrderNo          string      // 주문번호
    StockCode        string      // 종목코드
    StockName        string      // 종목명
    OrderSide        OrderSide   // buy/sell
    OrderPrice       int64       // 주문가격
    OrderQuantity    int64       // 주문수량
    ExecutedQuantity int64       // 체결수량
    ExecutedPrice    int64       // 체결가격
    RemainingQty     int64       // 잔여수량
    Status           OrderStatus // pending/partial/filled/cancelled
    OrderTime        string      // 주문시간 (HHMMSS)
    OrderDate        string      // 주문일자 (YYYYMMDD)
}
```

#### 4. 주문 실행

```go
// 매수 주문
result, err := client.PlaceOrder(ctx, kis.PlaceOrderRequest{
    StockCode: "005930",
    Side:      kis.OrderSideBuy,
    Type:      kis.OrderTypeLimit,  // 지정가
    Quantity:  10,
    Price:     72000,
})

// 시장가 매도
result, err := client.PlaceOrder(ctx, kis.PlaceOrderRequest{
    StockCode: "005930",
    Side:      kis.OrderSideSell,
    Type:      kis.OrderTypeMarket,  // 시장가
    Quantity:  10,
    Price:     0,  // 시장가는 0
})

// 주문 취소
result, err := client.CancelOrder(ctx, "0001234567")
```

#### 5. 실시간 WebSocket

```go
// WebSocket 클라이언트 생성
wsClient := kis.NewWSClient(cfg.KIS, log)

// HTS ID 설정 (체결통보용)
wsClient.SetHtsID("MYHTSPD01")

// 콜백 설정
wsClient.OnTick(func(tick *kis.TickData) {
    fmt.Printf("[%s] %d원 (%+d, %.2f%%)\n",
        tick.Symbol, tick.Price, tick.Change, tick.ChangeRate)
})

wsClient.OnExecution(func(exec *kis.ExecutionNotice) {
    fmt.Printf("[체결] %s %s %d주 @ %d원\n",
        exec.StockCode, exec.OrderSide, exec.ExecutedQty, exec.ExecutedPrice)
})

wsClient.OnError(func(err error) {
    log.Error("WebSocket error: %v", err)
})

// 연결
err := wsClient.Connect(ctx)

// 종목 구독 (최대 41개)
err := wsClient.Subscribe("005930", "000660", "035720")

// 체결통보 구독
err := wsClient.SubscribeExecution()

// 구독 해제
err := wsClient.Unsubscribe("005930")

// 연결 종료
wsClient.Disconnect()

// 재연결
err := wsClient.Reconnect(ctx)
```

### TR ID 참조

| TR ID | 용도 | 비고 |
|-------|------|------|
| `TTTC8434R` | 잔고 조회 (실전) | |
| `VTTC8434R` | 잔고 조회 (모의) | |
| `TTTC8001R` | 주문 조회 (실전) | |
| `VTTC8001R` | 주문 조회 (모의) | |
| `TTTC0802U` | 매수 (실전) | |
| `VTTC0802U` | 매수 (모의) | |
| `TTTC0801U` | 매도 (실전) | |
| `VTTC0801U` | 매도 (모의) | |
| `TTTC0803U` | 취소 (실전) | |
| `VTTC0803U` | 취소 (모의) | |
| `H0STCNT0` | 실시간 체결가 | WebSocket |
| `H0STCNI0` | 체결통보 (실전) | WebSocket |
| `H0STCNI9` | 체결통보 (모의) | WebSocket |

### WebSocket URL

| 환경 | URL |
|------|-----|
| 실전 | `ws://ops.koreainvestment.com:21000/` |
| 모의 | `ws://ops.koreainvestment.com:31000/` |

### 제한 사항

- 세션당 최대 **41개** 종목 구독
- 체결통보 구독에는 **HTS ID** 필요
- 체결통보 데이터는 **AES-256-CBC** 암호화

---

## DART API

### 환경 변수

```bash
DART_API_KEY=your-dart-api-key
DART_BASE_URL=https://opendart.fss.or.kr/api
```

### 주요 기능

공시 데이터 수집 (예정)

---

## Naver Finance

### 환경 변수

```bash
NAVER_BASE_URL=https://finance.naver.com
```

### 주요 기능

시세 및 종목 정보 수집 (예정)

---

**Prev**: [Forecast Layer](./forecast-layer.md)
