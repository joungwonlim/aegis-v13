---
sidebar_position: 2
title: Signals API
description: S2 팩터 시그널 API
---

# Signals API

> S2 팩터 시그널 조회 및 관리 API

---

## 엔드포인트 목록

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/signals/latest` | 최신 시그널 조회 |
| `GET` | `/signals/:date` | 특정 날짜 시그널 조회 |
| `GET` | `/signals/:date/:code` | 종목별 시그널 상세 |
| `POST` | `/signals/calculate` | 시그널 계산 실행 |

---

## GET /signals/latest

최신 시그널 세트 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     http://localhost:8089/api/v1/signals/latest
```

### Response

```json
{
  "success": true,
  "data": {
    "date": "2024-01-15",
    "total_stocks": 1850,
    "signals": {
      "005930": {
        "code": "005930",
        "name": "삼성전자",
        "factors": {
          "momentum": 0.65,
          "technical": 0.42,
          "value": -0.15,
          "quality": 0.78,
          "flow": 0.89,
          "event": 0.25
        },
        "total_score": 0.615,
        "updated_at": "2024-01-15T09:00:00Z"
      },
      ...
    }
  }
}
```

---

## GET /signals/:date

특정 날짜의 시그널 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8089/api/v1/signals/2024-01-15?limit=100"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `limit` | int | No | 반환 개수 (기본: 100, 최대: 1000) |
| `offset` | int | No | 시작 위치 (pagination) |
| `min_score` | float | No | 최소 total_score 필터 |

### Response

```json
{
  "success": true,
  "data": {
    "date": "2024-01-15",
    "signals": [ ... ],
    "pagination": {
      "limit": 100,
      "offset": 0,
      "total": 1850,
      "hasNext": true
    }
  }
}
```

---

## GET /signals/:date/:code

종목별 시그널 상세 정보

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     http://localhost:8089/api/v1/signals/2024-01-15/005930
```

### Response

```json
{
  "success": true,
  "data": {
    "code": "005930",
    "name": "삼성전자",
    "date": "2024-01-15",
    "factors": {
      "momentum": {
        "score": 0.65,
        "details": {
          "return_1m": 0.08,
          "return_3m": 0.15,
          "volume_rate": 1.25
        }
      },
      "technical": {
        "score": 0.42,
        "details": {
          "rsi": 62,
          "macd": 0.003,
          "bollinger_position": 0.65
        }
      },
      "value": {
        "score": -0.15,
        "details": {
          "per": 18.5,
          "pbr": 1.8,
          "psr": 2.2
        }
      },
      "quality": {
        "score": 0.78,
        "details": {
          "roe": 0.22,
          "roa": 0.12,
          "debt_ratio": 0.45
        }
      },
      "flow": {
        "score": 0.89,
        "details": {
          "foreign_net_5d": 15000000000,
          "foreign_net_20d": 45000000000,
          "inst_net_5d": 8000000000,
          "inst_net_20d": 25000000000
        }
      },
      "event": {
        "score": 0.25,
        "details": {
          "recent_events": [
            {
              "type": "dividend_increase",
              "date": "2024-01-10",
              "impact": 1.5
            }
          ]
        }
      }
    },
    "total_score": 0.615,
    "weights": {
      "momentum": 0.20,
      "technical": 0.20,
      "value": 0.15,
      "quality": 0.15,
      "flow": 0.25,
      "event": 0.05
    }
  }
}
```

---

## POST /signals/calculate

시그널 계산 실행 (수동 트리거)

### Request

```bash
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "date": "2024-01-15",
       "universe": ["005930", "000660", "035420"]
     }' \
     http://localhost:8089/api/v1/signals/calculate
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `date` | string | Yes | 계산 날짜 (YYYY-MM-DD) |
| `universe` | []string | No | 대상 종목 (미지정 시 전체 universe) |
| `factors` | []string | No | 계산 팩터 (미지정 시 전체) |

### Response

```json
{
  "success": true,
  "data": {
    "job_id": "signal_calc_20240115_143052",
    "status": "RUNNING",
    "progress": {
      "total": 1850,
      "completed": 0,
      "failed": 0
    },
    "estimated_duration": "5m"
  }
}
```

### 계산 상태 확인

```bash
GET /signals/jobs/signal_calc_20240115_143052
```

응답:

```json
{
  "success": true,
  "data": {
    "job_id": "signal_calc_20240115_143052",
    "status": "COMPLETED",
    "progress": {
      "total": 1850,
      "completed": 1845,
      "failed": 5
    },
    "duration": "4m32s",
    "result": {
      "signals_generated": 1845,
      "errors": [
        {
          "code": "000000",
          "reason": "No price data"
        }
      ]
    }
  }
}
```

---

## 시그널 점수 해석

### Momentum (0.65)
- **0.5 ~ 1.0**: 강한 상승세
- **0.0 ~ 0.5**: 약한 상승세
- **-0.5 ~ 0.0**: 약한 하락세
- **-1.0 ~ -0.5**: 강한 하락세

### Flow / 수급 (0.89)
- **0.7 ~ 1.0**: 외국인+기관 강한 매수
- **0.3 ~ 0.7**: 중간 매수세
- **-0.3 ~ 0.3**: 중립
- **-1.0 ~ -0.3**: 매도세

### Total Score (0.615)
- **0.6 ~ 1.0**: 매수 후보 (강력 추천)
- **0.3 ~ 0.6**: 매수 고려
- **-0.3 ~ 0.3**: 중립 (관망)
- **-1.0 ~ -0.3**: 매도 고려

---

## 에러 코드

| 코드 | 설명 | 해결 |
|------|------|------|
| `SIGNALS_NOT_FOUND` | 해당 날짜 시그널 없음 | 계산 실행 또는 다른 날짜 시도 |
| `STOCK_NOT_FOUND` | 종목 코드 없음 | 유효한 종목 코드 확인 |
| `CALCULATION_FAILED` | 시그널 계산 실패 | 로그 확인 후 재시도 |
| `INVALID_DATE` | 잘못된 날짜 형식 | YYYY-MM-DD 형식 사용 |

---

## 사용 예시

### Python

```python
import requests

API_KEY = "YOUR_KEY"
BASE_URL = "http://localhost:8089/api/v1"

headers = {"Authorization": f"Bearer {API_KEY}"}

# 최신 시그널 조회
response = requests.get(f"{BASE_URL}/signals/latest", headers=headers)
signals = response.json()["data"]["signals"]

# 특정 종목 상세
response = requests.get(
    f"{BASE_URL}/signals/2024-01-15/005930",
    headers=headers
)
samsung = response.json()["data"]
print(f"삼성전자 Total Score: {samsung['total_score']}")
```

### Go

```go
client := aegis.NewClient("YOUR_KEY")

// 최신 시그널
signals, err := client.Signals.GetLatest(ctx)
if err != nil {
    log.Fatal(err)
}

// 종목 필터링 (total_score >= 0.6)
for code, signal := range signals.Signals {
    if signal.TotalScore >= 0.6 {
        fmt.Printf("%s: %.2f\n", code, signal.TotalScore)
    }
}
```

---

**Prev**: [API Overview](./overview.md)
**Next**: [Selection API](./selection-api.md)
