---
sidebar_position: 7
title: Error Codes
description: API 에러 코드 전체 목록
---

# Error Codes

> Aegis v13 API 에러 코드 상세 가이드

---

## 에러 응답 형식

```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Missing required parameter: date",
    "details": {
      "parameter": "date",
      "expected": "YYYY-MM-DD"
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## HTTP Status Codes

| Status | 의미 | 사용 |
|--------|------|------|
| 200 | OK | 성공 |
| 400 | Bad Request | 잘못된 요청 파라미터 |
| 401 | Unauthorized | 인증 실패 |
| 403 | Forbidden | 권한 없음 |
| 404 | Not Found | 리소스 없음 |
| 409 | Conflict | 충돌 (중복 데이터) |
| 429 | Too Many Requests | Rate limit 초과 |
| 500 | Internal Server Error | 서버 내부 오류 |
| 503 | Service Unavailable | 서비스 일시 중단 |

---

## 공통 에러 (Common)

### INVALID_REQUEST (400)

잘못된 요청 파라미터

```json
{
  "code": "INVALID_REQUEST",
  "message": "Missing required parameter: date"
}
```

**원인**:
- 필수 파라미터 누락
- 잘못된 파라미터 형식
- 파라미터 범위 초과

**해결**:
- API 문서에서 필수 파라미터 확인
- 파라미터 형식 검증 (날짜, 숫자 등)

---

### UNAUTHORIZED (401)

인증 실패

```json
{
  "code": "UNAUTHORIZED",
  "message": "Invalid or expired API key"
}
```

**원인**:
- API 키 누락
- 잘못된 API 키
- 만료된 API 키

**해결**:
- `Authorization: Bearer YOUR_KEY` 헤더 확인
- API 키 갱신

---

### FORBIDDEN (403)

권한 없음

```json
{
  "code": "FORBIDDEN",
  "message": "Insufficient permissions for this operation"
}
```

**원인**:
- API 키 권한 부족
- 리소스 접근 권한 없음

**해결**:
- 계정 권한 확인
- 필요 권한 요청

---

### NOT_FOUND (404)

리소스 없음

```json
{
  "code": "NOT_FOUND",
  "message": "Resource not found"
}
```

**원인**:
- 잘못된 엔드포인트 경로
- 존재하지 않는 리소스 ID

**해결**:
- 엔드포인트 경로 확인
- 리소스 ID 검증

---

### RATE_LIMIT_EXCEEDED (429)

Rate limit 초과

```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Too many requests. Retry after 60 seconds",
  "details": {
    "limit": 100,
    "window": "1m",
    "retry_after": 60
  }
}
```

**원인**:
- 1분당 요청 제한 초과

**해결**:
- `retry_after` 초 대기 후 재시도
- 요청 빈도 조절

---

### INTERNAL_ERROR (500)

서버 내부 오류

```json
{
  "code": "INTERNAL_ERROR",
  "message": "An unexpected error occurred",
  "details": {
    "request_id": "req_123456"
  }
}
```

**원인**:
- 서버 내부 버그
- 데이터베이스 오류

**해결**:
- `request_id`와 함께 문의
- 잠시 후 재시도

---

## Signals 에러

### SIGNALS_NOT_FOUND (404)

시그널 데이터 없음

```json
{
  "code": "SIGNALS_NOT_FOUND",
  "message": "No signals found for date 2024-01-15"
}
```

**해결**:
- 시그널 계산 실행: `POST /signals/calculate`
- 다른 날짜 시도

---

### STOCK_NOT_FOUND (404)

종목 코드 없음

```json
{
  "code": "STOCK_NOT_FOUND",
  "message": "Stock code 999999 not found"
}
```

**해결**:
- 유효한 종목 코드 사용
- Universe에 포함된 종목 확인

---

### CALCULATION_FAILED (500)

시그널 계산 실패

```json
{
  "code": "CALCULATION_FAILED",
  "message": "Failed to calculate signals for 123 stocks",
  "details": {
    "failed_codes": ["000000", "111111"]
  }
}
```

**해결**:
- 로그 확인
- 개별 종목 데이터 확인
- 재시도

---

## Selection 에러

### NO_SIGNALS_AVAILABLE (400)

시그널 데이터 없음

```json
{
  "code": "NO_SIGNALS_AVAILABLE",
  "message": "No signals available for date 2024-01-15"
}
```

**해결**:
- 시그널 먼저 계산: `POST /signals/calculate`

---

### INVALID_WEIGHTS (400)

잘못된 가중치

```json
{
  "code": "INVALID_WEIGHTS",
  "message": "Weight sum must equal 1.0, got 0.95"
}
```

**해결**:
```json
{
  "weights": {
    "momentum": 0.20,
    "technical": 0.20,
    "value": 0.15,
    "quality": 0.15,
    "flow": 0.25,
    "event": 0.05
  }
}
// Sum = 1.0 ✅
```

---

### INVALID_FILTER (400)

잘못된 필터 값

```json
{
  "code": "INVALID_FILTER",
  "message": "min_momentum must be between -1.0 and 1.0"
}
```

**해결**:
- 필터 범위 확인
- 유효한 값 사용

---

## Portfolio 에러

### NO_RANKED_STOCKS (400)

랭킹 데이터 없음

```json
{
  "code": "NO_RANKED_STOCKS",
  "message": "No ranked stocks available for date 2024-01-15"
}
```

**해결**:
- 랭킹 먼저 실행: `POST /selection/rank`

---

### INSUFFICIENT_STOCKS (400)

종목 수 부족

```json
{
  "code": "INSUFFICIENT_STOCKS",
  "message": "Only 5 stocks passed filters, need at least 10"
}
```

**해결**:
- `min_weight` 낮추기
- 스크리닝 필터 완화

---

### CONSTRAINT_VIOLATION (400)

제약조건 위반

```json
{
  "code": "CONSTRAINT_VIOLATION",
  "message": "Sector weight exceeds max_sector_weight (0.35 > 0.30)"
}
```

**해결**:
- `max_sector_weight` 증가
- 종목 제외 또는 조정

---

## Execution 에러

### INSUFFICIENT_FUNDS (400)

잔고 부족

```json
{
  "code": "INSUFFICIENT_FUNDS",
  "message": "Insufficient funds for order. Required: 5000000, Available: 3000000"
}
```

**해결**:
- 매도 주문 먼저 실행
- 주문 금액 조정

---

### ORDER_REJECTED (400)

증권사 거부

```json
{
  "code": "ORDER_REJECTED",
  "message": "Order rejected by broker",
  "details": {
    "broker_code": "BR-101",
    "broker_message": "Invalid price tick"
  }
}
```

**원인**:
- 잘못된 호가 단위
- 가격 범위 초과
- 시장가 불가 종목

**해결**:
- 호가 단위 확인 (50원, 100원, 500원 등)
- 가격 범위 조정

---

### INVALID_PRICE (400)

잘못된 가격

```json
{
  "code": "INVALID_PRICE",
  "message": "Price 73250 does not match tick size 500"
}
```

**해결**:
```
가격대별 호가 단위:
< 1,000원: 1원
1,000 ~ 5,000원: 5원
5,000 ~ 10,000원: 10원
10,000 ~ 50,000원: 50원
50,000 ~ 100,000원: 100원
100,000 ~ 500,000원: 500원
> 500,000원: 1,000원
```

---

### MARKET_CLOSED (400)

장 마감

```json
{
  "code": "MARKET_CLOSED",
  "message": "Market is closed. Trading hours: 09:00 - 15:30 KST"
}
```

**해결**:
- 장 시간 확인 (09:00 ~ 15:30 KST)
- 다음 장 시작까지 대기

---

### STOCK_SUSPENDED (400)

거래정지 종목

```json
{
  "code": "STOCK_SUSPENDED",
  "message": "Stock 000000 is suspended from trading"
}
```

**해결**:
- 거래정지 사유 확인
- 해당 종목 제외

---

## Audit 에러

### NO_DATA_FOR_PERIOD (404)

기간 데이터 없음

```json
{
  "code": "NO_DATA_FOR_PERIOD",
  "message": "No data available for period 3M"
}
```

**해결**:
- 기간 조정 (더 짧은 기간)
- 데이터 수집 확인

---

### INSUFFICIENT_SNAPSHOTS (400)

스냅샷 부족

```json
{
  "code": "INSUFFICIENT_SNAPSHOTS",
  "message": "Need at least 20 snapshots for analysis, got 8"
}
```

**해결**:
- 더 많은 데이터 축적 대기
- 더 짧은 기간 선택

---

## 에러 처리 예시

### Python

```python
import requests
import time

def api_call_with_retry(url, headers, max_retries=3):
    for attempt in range(max_retries):
        try:
            response = requests.get(url, headers=headers)
            data = response.json()

            if data["success"]:
                return data["data"]

            # 에러 처리
            error = data["error"]
            code = error["code"]

            if code == "RATE_LIMIT_EXCEEDED":
                retry_after = error["details"]["retry_after"]
                print(f"Rate limited. Waiting {retry_after}s...")
                time.sleep(retry_after)
                continue

            elif code in ["SIGNALS_NOT_FOUND", "NO_RANKED_STOCKS"]:
                print(f"Data not ready: {error['message']}")
                return None

            elif code == "UNAUTHORIZED":
                raise ValueError("Invalid API key")

            else:
                print(f"Error {code}: {error['message']}")
                return None

        except requests.exceptions.RequestException as e:
            print(f"Network error: {e}")
            if attempt < max_retries - 1:
                time.sleep(2 ** attempt)  # Exponential backoff
            else:
                raise

    return None
```

### Go

```go
func callAPIWithRetry(url string, maxRetries int) (interface{}, error) {
    client := &http.Client{Timeout: 30 * time.Second}

    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := client.Get(url)
        if err != nil {
            if attempt < maxRetries-1 {
                time.Sleep(time.Duration(1<<attempt) * time.Second)
                continue
            }
            return nil, err
        }
        defer resp.Body.Close()

        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)

        if result["success"].(bool) {
            return result["data"], nil
        }

        errObj := result["error"].(map[string]interface{})
        code := errObj["code"].(string)

        switch code {
        case "RATE_LIMIT_EXCEEDED":
            retryAfter := int(errObj["details"].(map[string]interface{})["retry_after"].(float64))
            time.Sleep(time.Duration(retryAfter) * time.Second)
            continue

        case "SIGNALS_NOT_FOUND", "NO_RANKED_STOCKS":
            return nil, fmt.Errorf("data not ready: %s", errObj["message"])

        case "UNAUTHORIZED":
            return nil, fmt.Errorf("invalid API key")

        default:
            return nil, fmt.Errorf("%s: %s", code, errObj["message"])
        }
    }

    return nil, fmt.Errorf("max retries exceeded")
}
```

---

## 에러 디버깅 팁

### 1. Request ID 활용

모든 에러 응답에 `request_id` 포함:

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "details": {
      "request_id": "req_abc123"
    }
  }
}
```

문의 시 `request_id` 제공하면 빠른 디버깅 가능

### 2. 상세 로그

```python
import logging

logging.basicConfig(level=logging.DEBUG)

response = requests.get(url, headers=headers)
logging.debug(f"Request: {url}")
logging.debug(f"Response: {response.text}")
```

### 3. 에러 모니터링

```python
error_counts = {}

def track_error(error_code):
    error_counts[error_code] = error_counts.get(error_code, 0) + 1

    if error_counts[error_code] > 10:
        alert(f"High error rate for {error_code}")
```

---

**Prev**: [Audit API](./audit-api.md)
**Next**: [API Overview](./overview.md)
