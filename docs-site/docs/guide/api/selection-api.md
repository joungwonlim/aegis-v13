---
sidebar_position: 3
title: Selection API
description: S3-S4 스크리닝/랭킹 API
---

# Selection API

> S3 Screener + S4 Ranker API

---

## 엔드포인트 목록

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/selection/screened` | 스크리닝 결과 조회 |
| `GET` | `/selection/ranked` | 랭킹 결과 조회 |
| `POST` | `/selection/screen` | 스크리닝 실행 |
| `POST` | `/selection/rank` | 랭킹 실행 |

---

## GET /selection/screened

스크리닝 결과 조회 (Hard Cut 통과 종목)

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     http://localhost:8080/api/v1/selection/screened?date=2024-01-15
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
    "input_count": 1850,
    "passed_count": 247,
    "passed_stocks": [
      "005930",
      "000660",
      "035420",
      ...
    ],
    "filters_applied": {
      "min_momentum": -1.0,
      "min_volume": 500000000,
      "max_per": 50,
      "min_roe": 0.05
    },
    "filter_stats": {
      "momentum_filtered": 523,
      "volume_filtered": 680,
      "per_filtered": 320,
      "roe_filtered": 80
    }
  }
}
```

---

## GET /selection/ranked

랭킹 결과 조회 (Top N 종목)

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/selection/ranked?date=2024-01-15&top_n=30"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `date` | string | No | 날짜 (기본: 최신) |
| `top_n` | int | No | 상위 N개 (기본: 30) |

### Response

```json
{
  "success": true,
  "data": {
    "date": "2024-01-15",
    "total_ranked": 247,
    "top_stocks": [
      {
        "rank": 1,
        "code": "005930",
        "name": "삼성전자",
        "total_score": 0.785,
        "scores": {
          "momentum": 0.65,
          "technical": 0.72,
          "value": 0.45,
          "quality": 0.88,
          "flow": 0.92,
          "event": 0.35
        }
      },
      {
        "rank": 2,
        "code": "000660",
        "name": "SK하이닉스",
        "total_score": 0.742,
        "scores": {
          "momentum": 0.78,
          "technical": 0.65,
          "value": 0.32,
          "quality": 0.75,
          "flow": 0.88,
          "event": 0.42
        }
      },
      ...
    ]
  }
}
```

---

## POST /selection/screen

스크리닝 실행

### Request

```bash
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "date": "2024-01-15",
       "filters": {
         "min_momentum": -0.5,
         "min_volume": 1000000000,
         "max_per": 30,
         "min_roe": 0.10
       }
     }' \
     http://localhost:8080/api/v1/selection/screen
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `date` | string | Yes | 스크리닝 날짜 |
| `filters` | object | No | 커스텀 필터 (미지정 시 기본값) |

### Response

```json
{
  "success": true,
  "data": {
    "job_id": "screen_20240115_143052",
    "status": "COMPLETED",
    "result": {
      "passed_count": 156,
      "passed_stocks": [ ... ]
    }
  }
}
```

---

## POST /selection/rank

랭킹 실행

### Request

```bash
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "date": "2024-01-15",
       "weights": {
         "momentum": 0.25,
         "technical": 0.15,
         "value": 0.20,
         "quality": 0.15,
         "flow": 0.20,
         "event": 0.05
       },
       "top_n": 30
     }' \
     http://localhost:8080/api/v1/selection/rank
```

### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `date` | string | Yes | 랭킹 날짜 |
| `weights` | object | No | 팩터 가중치 (합=1.0) |
| `top_n` | int | No | 상위 N개 (기본: 30) |

### Response

```json
{
  "success": true,
  "data": {
    "job_id": "rank_20240115_143052",
    "status": "COMPLETED",
    "result": {
      "total_ranked": 156,
      "top_stocks": [ ... ]
    }
  }
}
```

---

## 스크리닝 필터 기본값

```yaml
min_momentum: -1.0      # 모멘텀 최소값
min_volume: 500000000   # 최소 거래대금 (5억)
max_per: 50             # PER 최대값
min_roe: 0.05           # ROE 최소값 (5%)
exclude_negative_earnings: true  # 적자 종목 제외
```

---

## 랭킹 가중치 (SSOT)

```yaml
momentum: 0.20    # 20%
technical: 0.20   # 20%
value: 0.15       # 15%
quality: 0.15     # 15%
flow: 0.25        # 25% ⭐ 수급 (한국 시장 중요)
event: 0.05       # 5%
```

---

## 에러 코드

| 코드 | 설명 | 해결 |
|------|------|------|
| `NO_SIGNALS_AVAILABLE` | 시그널 데이터 없음 | 시그널 계산 먼저 실행 |
| `INVALID_WEIGHTS` | 가중치 합이 1.0이 아님 | 가중치 합 = 1.0으로 수정 |
| `INVALID_FILTER` | 잘못된 필터 값 | 필터 범위 확인 |

---

**Prev**: [Signals API](./signals-api.md)
**Next**: [Portfolio API](./portfolio-api.md)
