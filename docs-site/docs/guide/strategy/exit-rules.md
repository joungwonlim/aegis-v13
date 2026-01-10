# 청산규칙 (Exit Rules)

> ATR 기반 동적 청산 전략 v1.2.0

---

## 개요

청산규칙은 포지션의 익절(Take Profit)과 손절(Stop Loss)을 자동으로 관리하는 시스템입니다.
ATR(Average True Range) 기반의 동적 트리거를 사용하여 종목별 변동성에 맞는 청산 전략을 적용합니다.

### 핵심 특징

- **ATR 기반 동적 트리거**: 종목별 변동성에 따라 익절/손절 가격 자동 조정
- **분할 청산**: 단계별로 포지션의 일부만 청산하여 수익 극대화
- **상태 머신**: 포지션 상태에 따른 적절한 청산 규칙 적용
- **보호 메커니즘**: Stop Floor, HWM Trailing으로 수익 보호

---

## 익절 규칙 (Take Profit)

### 3단계 분할 익절

| 단계 | ATR 배수 | 트리거 범위 | 매도 비율 | 설명 |
|------|----------|------------|----------|------|
| **TP1** | × 1.5 | 6% ~ 8% | 25% | 1차 익절 |
| **TP2** | × 2.5 | 10% ~ 12% | 25% | 2차 익절 |
| **TP3** | × 3.5 | 15% ~ 18% | 20% | 3차 익절 |

### ATR 기반 트리거 계산

```
트리거 가격 = 진입가 × (1 + min(max(ATR% × 배수, 최소%), 최대%))
```

**예시** (진입가 10,000원, ATR 2%):
- TP1: 10,000 × (1 + min(max(2% × 1.5, 6%), 8%)) = 10,600원 (+6%)
- TP2: 10,000 × (1 + min(max(2% × 2.5, 10%), 12%)) = 11,000원 (+10%)
- TP3: 10,000 × (1 + min(max(2% × 3.5, 15%), 18%)) = 11,500원 (+15%)

### 분할 익절 후 잔량

```
초기 100주 기준:
TP1 후: 75주 (25% 매도)
TP2 후: 50주 (추가 25% 매도)
TP3 후: 30주 (추가 20% 매도) → HWM Trailing 시작
```

---

## 손절 규칙 (Stop Loss)

### 2단계 손절

| 단계 | 트리거 | 매도 비율 | 설명 |
|------|--------|----------|------|
| **1차 손절** | -3% | 50% | 진입가 대비 -3% 하락 시 |
| **2차 손절** | -5% | 전량 | 진입가 대비 -5% 하락 시 |
| **Hard Stop** | -7% | 전량 | 어떤 상황에서도 발동 |

### 손절 우선순위

```
2차 손절 (-5%) > 1차 손절 (-3%) > Stop Floor > HWM Trail > Take Profit
```

손절 조건이 익절 조건보다 **항상 우선** 체크됩니다.

---

## 보호 메커니즘

### 1. Stop Floor (손익분기점 보호)

**활성화 조건**: TP1 완료 후

```
Stop Floor 가격 = 진입가 × (1 + 0.6%)
```

TP1 이후에는 최소한 손익분기점 + 0.6% 버퍼를 보장합니다.
현재가가 Stop Floor 아래로 내려가면 잔량 전량 청산합니다.

**예시** (진입가 10,000원):
- TP1 완료 후 Stop Floor = 10,060원
- 현재가가 10,060원 아래로 하락 시 전량 청산

### 2. HWM Trailing (최고가 추적 손절)

**활성화 조건**: TP3 완료 후

```
Trailing Stop 가격 = 최고가 × (1 - min(max(ATR% × 2.0, 3%), 5%))
```

TP3 이후 잔여 30%는 최고가를 추적하며, 최고가 대비 일정 비율 하락 시 청산합니다.

**특징**:
- Trailing Stop은 **올라가기만** 함 (내려가지 않음)
- 최고가가 갱신될 때마다 Trailing Stop도 상향 조정
- 수익을 보호하면서 추가 상승 여력을 남김

**예시** (진입가 10,000원, ATR 2%, 현재 최고가 12,000원):
- Trail 거리 = max(2% × 2.0, 3%) = 4%
- Trailing Stop = 12,000 × (1 - 4%) = 11,520원

---

## 상태 머신

포지션은 다음 상태를 순차적으로 거칩니다:

```
S0_OPEN → S1_TP1_DONE → S2_TP2_DONE → S3_TP3_DONE → S4_EXITING → S5_CLOSED
```

| 상태 | 설명 | 활성화된 규칙 |
|------|------|--------------|
| `S0_OPEN` | 진입 완료, 익절 없음 | 1차/2차 손절, TP1 체크 |
| `S1_TP1_DONE` | TP1 완료 | Stop Floor 활성화, TP2 체크 |
| `S2_TP2_DONE` | TP2 완료 | Stop Floor 유지, TP3 체크 |
| `S3_TP3_DONE` | TP3 완료 | HWM Trailing 활성화 |
| `S4_EXITING` | 청산 진행 중 | - |
| `S5_CLOSED` | 완전 청산 | - |

---

## 청산 사유 (Exit Reason)

| 사유 | 코드 | 설명 |
|------|------|------|
| 1차 익절 | `TP1` | ATR 기반 1차 목표가 도달 |
| 2차 익절 | `TP2` | ATR 기반 2차 목표가 도달 |
| 3차 익절 | `TP3` | ATR 기반 3차 목표가 도달 |
| 1차 손절 | `FIRST_STOP` | 진입가 대비 -3% 하락 |
| 2차 손절 | `SECOND_STOP` | 진입가 대비 -5% 하락 |
| 하드 스탑 | `HARD_STOP` | 진입가 대비 -7% 하락 |
| 스탑 플로어 | `STOP_FLOOR` | 손익분기점+0.6% 이탈 |
| HWM 트레일 | `HWM_TRAIL` | 최고가 대비 트레일링 스탑 이탈 |
| 수동 청산 | `MANUAL` | 사용자 수동 청산 |

---

## 기본 설정값

```go
ExitRulesConfig{
    // ATR 기반 활성화
    UseATRBased: true,

    // TP1: ATR × 1.5, clamp [6%, 8%], 25% 매도
    TP1ATRMultiplier: 1.5,
    TP1MinPercent:    6.0,
    TP1MaxPercent:    8.0,
    TP1SellPercent:   25.0,

    // TP2: ATR × 2.5, clamp [10%, 12%], 25% 매도
    TP2ATRMultiplier: 2.5,
    TP2MinPercent:    10.0,
    TP2MaxPercent:    12.0,
    TP2SellPercent:   25.0,

    // TP3: ATR × 3.5, clamp [15%, 18%], 20% 매도
    TP3ATRMultiplier: 3.5,
    TP3MinPercent:    15.0,
    TP3MaxPercent:    18.0,
    TP3SellPercent:   20.0,

    // 손절
    FirstStopPercent:     -3.0,
    FirstStopSellPercent: 50.0,
    SecondStopPercent:    -5.0,
    HardStopPercent:      -7.0,

    // Stop Floor
    StopFloorBuffer: 0.6,

    // HWM Trailing
    TrailATRMultiplier: 2.0,
    TrailMinPercent:    3.0,
    TrailMaxPercent:    5.0,

    // 모니터링 주기
    CheckIntervalSeconds: 30,
}
```

---

## 사용 예시

```go
import (
    "github.com/wonny/aegis/v13/backend/internal/contracts"
    "github.com/wonny/aegis/v13/backend/internal/execution"
)

// 모니터 생성
monitor := execution.NewPositionMonitor(
    contracts.DefaultExitRulesConfig(),
    priceProvider,    // PriceProvider 인터페이스 구현체
    atrProvider,      // ATRProvider 인터페이스 구현체
    db.Pool,
    logger,
)

// 알림 설정 (선택)
monitor.SetNotifier(myNotifier)

// 포지션 추가
monitor.AddPosition(ctx, &contracts.MonitoredPosition{
    ID:              "pos-001",
    Code:            "005930",
    Name:            "삼성전자",
    EntryPrice:      70000,
    InitialQuantity: 100,
    EntryTime:       time.Now(),
})

// 백그라운드 모니터링 시작
monitor.Start(ctx)

// 수동 체크
signals, err := monitor.CheckAllPositions(ctx)
for _, signal := range signals {
    fmt.Printf("Exit signal: %s %s @ %d\n", signal.Code, signal.Reason, signal.CurrentPrice)
}
```

---

## 소스 코드 위치

| 파일 | 설명 |
|------|------|
| `internal/contracts/exit.go` | 타입 정의 (ExitRulesConfig, ExitSignal, MonitoredPosition 등) |
| `internal/execution/exit_rules.go` | PositionMonitor 구현체 |

---

## 참고

- v10의 청산규칙을 v13 아키텍처에 맞게 단순화하여 구현
- ATR 계산은 14일 True Range의 단순이동평균(SMA) 사용
- 모든 가격은 `int64` (원 단위)로 처리
