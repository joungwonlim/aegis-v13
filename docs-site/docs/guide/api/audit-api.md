---
sidebar_position: 6
title: Audit API
description: S7 성과 분석 API
---

# Audit API

> S7 성과 분석 및 귀인 분석 API

---

## 엔드포인트 목록

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `GET` | `/audit/performance` | 성과 분석 조회 |
| `GET` | `/audit/attribution` | 귀인 분석 조회 |
| `GET` | `/audit/snapshots` | 일별 스냅샷 조회 |
| `GET` | `/audit/equity-curve` | 자산 곡선 조회 |
| `POST` | `/audit/snapshot` | 스냅샷 저장 |

---

## GET /audit/performance

성과 분석 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/audit/performance?period=3M"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `period` | string | No | 기간: "1M", "3M", "6M", "1Y", "YTD" (기본: "1M") |

### Response

```json
{
  "success": true,
  "data": {
    "period": "3M",
    "start_date": "2023-10-15",
    "end_date": "2024-01-15",
    "returns": {
      "total_return": 0.152,
      "annual_return": 0.698,
      "daily_avg": 0.0015,
      "monthly_avg": 0.048
    },
    "risk": {
      "volatility": 0.185,
      "sharpe": 3.61,
      "sortino": 5.23,
      "max_drawdown": -0.082,
      "max_drawdown_duration": 12
    },
    "trading": {
      "win_rate": 0.65,
      "avg_win": 0.042,
      "avg_loss": -0.025,
      "profit_factor": 2.73,
      "total_trades": 145,
      "winning_trades": 94,
      "losing_trades": 51
    },
    "benchmark": {
      "name": "KOSPI",
      "return": 0.089,
      "alpha": 0.063,
      "beta": 0.92,
      "correlation": 0.78
    }
  }
}
```

---

## GET /audit/attribution

귀인 분석 조회 (팩터별 기여도)

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/audit/attribution?period=1M"
```

### Response

```json
{
  "success": true,
  "data": {
    "period": "1M",
    "start_date": "2023-12-15",
    "end_date": "2024-01-15",
    "total_return": 0.048,
    "attributions": [
      {
        "factor": "flow",
        "contribution": 0.018,
        "contribution_pct": 37.5,
        "exposure": 0.72,
        "return_pct": 2.5
      },
      {
        "factor": "momentum",
        "contribution": 0.012,
        "contribution_pct": 25.0,
        "exposure": 0.65,
        "return_pct": 1.85
      },
      {
        "factor": "technical",
        "contribution": 0.009,
        "contribution_pct": 18.75,
        "exposure": 0.58,
        "return_pct": 1.55
      },
      {
        "factor": "value",
        "contribution": 0.005,
        "contribution_pct": 10.42,
        "exposure": 0.42,
        "return_pct": 1.19
      },
      {
        "factor": "quality",
        "contribution": 0.003,
        "contribution_pct": 6.25,
        "exposure": 0.38,
        "return_pct": 0.79
      },
      {
        "factor": "event",
        "contribution": 0.001,
        "contribution_pct": 2.08,
        "exposure": 0.15,
        "return_pct": 0.67
      }
    ],
    "top_contributors": [
      {
        "code": "005930",
        "name": "삼성전자",
        "contribution": 0.008,
        "main_factor": "flow"
      },
      {
        "code": "000660",
        "name": "SK하이닉스",
        "contribution": 0.006,
        "main_factor": "momentum"
      }
    ],
    "bottom_contributors": [
      {
        "code": "035720",
        "name": "카카오",
        "contribution": -0.003,
        "main_factor": "value"
      }
    ]
  }
}
```

---

## GET /audit/snapshots

일별 포트폴리오 스냅샷 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/audit/snapshots?start_date=2024-01-01&end_date=2024-01-15"
```

### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| `start_date` | string | Yes | 시작 날짜 (YYYY-MM-DD) |
| `end_date` | string | Yes | 종료 날짜 (YYYY-MM-DD) |

### Response

```json
{
  "success": true,
  "data": {
    "snapshots": [
      {
        "date": "2024-01-15",
        "total_value": 105230000,
        "cash": 5230000,
        "daily_return": 0.0023,
        "cum_return": 1.152,
        "positions": [
          {
            "code": "005930",
            "name": "삼성전자",
            "quantity": 138,
            "price": 73200,
            "value": 10101600,
            "weight": 0.096,
            "daily_pnl": 96600
          },
          ...
        ]
      },
      ...
    ]
  }
}
```

---

## GET /audit/equity-curve

자산 곡선 데이터 조회

### Request

```bash
curl -H "Authorization: Bearer YOUR_KEY" \
     "http://localhost:8080/api/v1/audit/equity-curve?start_date=2023-10-01&end_date=2024-01-15"
```

### Response

```json
{
  "success": true,
  "data": {
    "start_date": "2023-10-01",
    "end_date": "2024-01-15",
    "initial_value": 100000000,
    "final_value": 115200000,
    "total_return": 0.152,
    "data_points": [
      {
        "date": "2023-10-01",
        "value": 100000000,
        "return": 0.0,
        "cum_return": 1.0
      },
      {
        "date": "2023-10-02",
        "value": 100150000,
        "return": 0.0015,
        "cum_return": 1.0015
      },
      ...
    ],
    "benchmark": [
      {
        "date": "2023-10-01",
        "value": 100000000,
        "return": 0.0,
        "cum_return": 1.0
      },
      ...
    ]
  }
}
```

---

## POST /audit/snapshot

일별 스냅샷 저장 (내부용)

### Request

```bash
curl -X POST \
     -H "Authorization: Bearer YOUR_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "date": "2024-01-15",
       "total_value": 105230000,
       "cash": 5230000,
       "positions": [ ... ]
     }' \
     http://localhost:8080/api/v1/audit/snapshot
```

### Response

```json
{
  "success": true,
  "data": {
    "message": "Snapshot saved successfully",
    "date": "2024-01-15"
  }
}
```

---

## 성과 지표 해석

### Sharpe Ratio (샤프 지수)

위험 대비 수익률 측정

```
Sharpe = (Annual Return - Risk Free Rate) / Volatility

해석:
> 3.0: 매우 우수
2.0 ~ 3.0: 우수
1.0 ~ 2.0: 양호
< 1.0: 개선 필요
```

### Sortino Ratio (소르티노 지수)

하락 위험 대비 수익률 (Sharpe보다 엄격)

```
Sortino = (Annual Return - Risk Free Rate) / Downside Volatility

해석:
> 4.0: 매우 우수
3.0 ~ 4.0: 우수
2.0 ~ 3.0: 양호
< 2.0: 개선 필요
```

### Max Drawdown (최대 낙폭)

```
MDD = (Trough - Peak) / Peak

예시:
Peak: 110,000,000원
Trough: 101,000,000원
MDD = -8.18%

해석:
< -5%: 매우 안정적
-5% ~ -10%: 안정적
-10% ~ -20%: 보통
> -20%: 위험
```

### Win Rate (승률)

```
Win Rate = Winning Trades / Total Trades

해석:
> 70%: 매우 높음
60% ~ 70%: 높음
50% ~ 60%: 보통
< 50%: 낮음
```

### Profit Factor (이익 팩터)

```
Profit Factor = Total Wins / Total Losses

해석:
> 3.0: 매우 우수
2.0 ~ 3.0: 우수
1.5 ~ 2.0: 양호
< 1.5: 개선 필요
```

---

## 귀인 분석 해석

### Contribution (기여도)

```
Total Return = 4.8%

flow: 1.8% (37.5%)        ⭐ 가장 큰 기여
momentum: 1.2% (25.0%)
technical: 0.9% (18.75%)
value: 0.5% (10.42%)
quality: 0.3% (6.25%)
event: 0.1% (2.08%)
```

### Exposure (노출도)

```
평균 팩터 점수의 절댓값

flow: 0.72    → 수급 신호에 높은 비중
momentum: 0.65
technical: 0.58
```

---

## 에러 코드

| 코드 | 설명 | 해결 |
|------|------|------|
| `NO_DATA_FOR_PERIOD` | 기간 데이터 없음 | 기간 조정 또는 데이터 수집 |
| `INVALID_PERIOD` | 잘못된 기간 형식 | "1M", "3M", "6M", "1Y", "YTD" 사용 |
| `INSUFFICIENT_SNAPSHOTS` | 스냅샷 부족 | 최소 20일 데이터 필요 |

---

## 사용 예시

### Python: 성과 대시보드

```python
import requests
import matplotlib.pyplot as plt

API_KEY = "YOUR_KEY"
BASE_URL = "http://localhost:8080/api/v1"
headers = {"Authorization": f"Bearer {API_KEY}"}

# 1. 성과 분석
response = requests.get(
    f"{BASE_URL}/audit/performance?period=3M",
    headers=headers
)
perf = response.json()["data"]

print(f"Total Return: {perf['returns']['total_return']:.2%}")
print(f"Sharpe Ratio: {perf['risk']['sharpe']:.2f}")
print(f"Max Drawdown: {perf['risk']['max_drawdown']:.2%}")
print(f"Win Rate: {perf['trading']['win_rate']:.2%}")

# 2. 귀인 분석
response = requests.get(
    f"{BASE_URL}/audit/attribution?period=3M",
    headers=headers
)
attr = response.json()["data"]

print("\nFactor Contributions:")
for a in attr["attributions"]:
    print(f"{a['factor']:10s}: {a['contribution']:+.2%} ({a['contribution_pct']:.1f}%)")

# 3. 자산 곡선
response = requests.get(
    f"{BASE_URL}/audit/equity-curve?start_date=2023-10-01&end_date=2024-01-15",
    headers=headers
)
curve = response.json()["data"]

# Plot
dates = [p["date"] for p in curve["data_points"]]
values = [p["value"] for p in curve["data_points"]]
benchmark = [p["value"] for p in curve["benchmark"]]

plt.figure(figsize=(12, 6))
plt.plot(dates, values, label="Portfolio", linewidth=2)
plt.plot(dates, benchmark, label="KOSPI", linewidth=2, alpha=0.7)
plt.xlabel("Date")
plt.ylabel("Value (KRW)")
plt.title("Portfolio vs Benchmark")
plt.legend()
plt.grid(True, alpha=0.3)
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig("equity_curve.png")
```

### Go: 성과 보고서 생성

```go
client := aegis.NewClient("YOUR_KEY")

// 1. 성과 분석
perf, err := client.Audit.GetPerformance(ctx, "3M")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Performance Report (3M)\n")
fmt.Printf("========================\n")
fmt.Printf("Total Return:  %.2f%%\n", perf.Returns.TotalReturn*100)
fmt.Printf("Sharpe Ratio:  %.2f\n", perf.Risk.Sharpe)
fmt.Printf("Max Drawdown:  %.2f%%\n", perf.Risk.MaxDrawdown*100)
fmt.Printf("Win Rate:      %.2f%%\n", perf.Trading.WinRate*100)

// 2. 귀인 분석
attr, err := client.Audit.GetAttribution(ctx, "3M")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("\nFactor Attribution\n")
fmt.Printf("==================\n")
for _, a := range attr.Attributions {
    fmt.Printf("%-10s: %+.2f%% (%.1f%%)\n",
        a.Factor,
        a.Contribution*100,
        a.ContributionPct,
    )
}

// 3. Top/Bottom Contributors
fmt.Printf("\nTop Contributors\n")
for _, c := range attr.TopContributors {
    fmt.Printf("%s (%s): %+.2f%%\n", c.Name, c.Code, c.Contribution*100)
}
```

---

**Prev**: [Execution API](./execution-api.md)
**Next**: [Error Codes](./errors.md)
