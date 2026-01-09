---
sidebar_position: 1
title: API Overview
description: Aegis v13 REST API 개요
---

# API Overview

> Aegis v13 Backend REST API 문서

---

## Base URL

```
http://localhost:8080/api/v1
```

프로덕션: `https://api.aegis.trade/v1`

---

## 인증

현재 버전은 **API 키 인증** 사용:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
     https://api.aegis.trade/v1/signals/latest
```

---

## 응답 형식

### 성공 응답

```json
{
  "success": true,
  "data": { ... },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 에러 응답

```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Missing required parameter: date",
    "details": { ... }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## 주요 엔드포인트

| 리소스 | 엔드포인트 | 설명 |
|--------|-----------|------|
| **Signals** | `/signals` | 팩터 시그널 조회 |
| **Selection** | `/selection` | 스크리닝/랭킹 결과 |
| **Portfolio** | `/portfolio` | 포트폴리오 구성 |
| **Execution** | `/execution` | 주문 실행/모니터링 |
| **Audit** | `/audit` | 성과 분석 |

---

## Rate Limiting

- **기본**: 100 requests/minute
- **Burst**: 200 requests/minute

Rate limit 초과 시 `429 Too Many Requests` 반환

---

## Pagination

리스트 조회 시 pagination 사용:

```bash
GET /api/v1/signals?page=1&limit=50
```

응답:

```json
{
  "success": true,
  "data": {
    "items": [ ... ],
    "pagination": {
      "page": 1,
      "limit": 50,
      "total": 500,
      "hasNext": true
    }
  }
}
```

---

## 날짜 형식

- **ISO 8601**: `2024-01-15T10:30:00Z`
- **Date only**: `2024-01-15`

쿼리 파라미터에서 날짜 사용 시:

```bash
GET /api/v1/signals?date=2024-01-15
```

---

## HTTP 메서드

| 메서드 | 용도 | 예시 |
|--------|------|------|
| `GET` | 조회 | `GET /signals/latest` |
| `POST` | 생성/실행 | `POST /execution/orders` |
| `PUT` | 수정 | `PUT /portfolio/config` |
| `DELETE` | 삭제 | `DELETE /execution/orders/123` |

---

## 에러 코드

| 코드 | HTTP Status | 설명 |
|------|-------------|------|
| `INVALID_REQUEST` | 400 | 잘못된 요청 |
| `UNAUTHORIZED` | 401 | 인증 실패 |
| `FORBIDDEN` | 403 | 권한 없음 |
| `NOT_FOUND` | 404 | 리소스 없음 |
| `CONFLICT` | 409 | 충돌 (중복 데이터 등) |
| `INTERNAL_ERROR` | 500 | 서버 에러 |
| `SERVICE_UNAVAILABLE` | 503 | 서비스 일시 중단 |

자세한 에러 코드: [Error Codes](./errors.md)

---

## 다음 단계

각 리소스별 상세 API 문서:

- [Signals API](./signals-api.md)
- [Selection API](./selection-api.md)
- [Portfolio API](./portfolio-api.md)
- [Execution API](./execution-api.md)
- [Audit API](./audit-api.md)

---

## SDK 및 도구

### cURL 예시

```bash
# 최신 시그널 조회
curl -H "Authorization: Bearer YOUR_KEY" \
     http://localhost:8080/api/v1/signals/latest

# 포트폴리오 구성
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{"date": "2024-01-15", "top_n": 30}' \
     http://localhost:8080/api/v1/portfolio/construct
```

### HTTP 클라이언트 (Go)

```go
import "github.com/wonny/aegis/v13/pkg/client"

client := client.New("YOUR_API_KEY")
signals, err := client.Signals.GetLatest(ctx)
```

---

## 웹훅 (Webhook)

특정 이벤트 발생 시 콜백 수신:

```json
{
  "event": "ORDER_FILLED",
  "data": {
    "order_id": "ORD-123",
    "code": "005930",
    "quantity": 10,
    "filled_price": 72300
  },
  "timestamp": "2024-01-15T14:30:00Z"
}
```

웹훅 설정: `POST /webhooks`

---

## Versioning

API 버전은 URL에 포함:

- `v1`: 현재 안정 버전
- `v2`: 개발 중 (Beta)

Breaking changes 발생 시 새 버전 릴리스

---

## 지원 및 문의

- **Documentation**: https://joungwonlim.github.io/aegis-v13/
- **Issues**: https://github.com/wonny/aegis/issues
- **Email**: support@aegis.trade
