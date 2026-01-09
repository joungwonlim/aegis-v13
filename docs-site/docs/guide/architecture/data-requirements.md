---
sidebar_position: 4
title: Data Requirements
description: 기관급 퀀트 시스템 데이터 요구사항
---

# Data Requirements

> 기관급 퀀트 시스템에 필요한 데이터 정의

---

## 데이터 분류

```
┌─────────────────────────────────────────────────────────────────┐
│                        DATA CATEGORIES                          │
├─────────────────────────────────────────────────────────────────┤
│  Reference     │  Market      │  Fundamental  │  Alternative   │
│  (기준정보)     │  (시장)       │  (재무)        │  (대체)        │
├─────────────────────────────────────────────────────────────────┤
│  종목 마스터    │  가격        │  재무제표      │  투자자 수급   │
│  섹터/업종     │  거래량      │  밸류에이션    │  공시/이벤트   │
│  지수 구성     │  시가총액    │  성장성        │  뉴스/센티먼트 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 1. Reference Data (기준정보)

### 1.1 종목 마스터

| 필드 | 설명 | 필수 | 소스 |
|------|------|------|------|
| `code` | 종목코드 (6자리) | ✅ | KRX |
| `name` | 종목명 | ✅ | KRX |
| `name_en` | 영문명 | - | KRX |
| `market` | 시장구분 (KOSPI/KOSDAQ) | ✅ | KRX |
| `sector` | 업종 | ✅ | KRX |
| `industry` | 세부업종 | - | KRX |
| `listing_date` | 상장일 | ✅ | KRX |
| `fiscal_month` | 결산월 | ✅ | DART |
| `is_halted` | 거래정지 여부 | ✅ | KRX |
| `is_admin` | 관리종목 여부 | ✅ | KRX |
| `is_spac` | SPAC 여부 | ✅ | KRX |

```go
type Stock struct {
    Code        string    `json:"code"`
    Name        string    `json:"name"`
    Market      string    `json:"market"`      // KOSPI, KOSDAQ
    Sector      string    `json:"sector"`
    ListingDate time.Time `json:"listing_date"`
    FiscalMonth int       `json:"fiscal_month"` // 12 = 12월 결산
    Status      StockStatus
}

type StockStatus struct {
    IsHalted bool `json:"is_halted"` // 거래정지
    IsAdmin  bool `json:"is_admin"`  // 관리종목
    IsSPAC   bool `json:"is_spac"`   // SPAC
}
```

**커버리지 목표**: 100%
**갱신 주기**: Daily

---

### 1.2 지수 구성종목

| 지수 | 설명 | 용도 |
|------|------|------|
| KOSPI200 | 코스피 200 | 대형주 Universe |
| KOSDAQ150 | 코스닥 150 | 중형주 Universe |
| KRX300 | KRX 300 | 통합 Universe |

```go
type IndexConstituent struct {
    IndexCode   string    `json:"index_code"`  // KOSPI200
    StockCode   string    `json:"stock_code"`
    Weight      float64   `json:"weight"`      // 비중
    EffectiveAt time.Time `json:"effective_at"`
}
```

**커버리지 목표**: 100%
**갱신 주기**: 정기변경 시 (분기)

---

## 2. Market Data (시장 데이터)

### 2.1 가격 데이터 (OHLCV)

| 필드 | 설명 | 필수 | 단위 |
|------|------|------|------|
| `open` | 시가 | ✅ | 원 |
| `high` | 고가 | ✅ | 원 |
| `low` | 저가 | ✅ | 원 |
| `close` | 종가 | ✅ | 원 |
| `volume` | 거래량 | ✅ | 주 |
| `value` | 거래대금 | ✅ | 원 |
| `adj_close` | 수정종가 | ✅ | 원 |
| `adj_factor` | 수정계수 | ✅ | - |

```go
type OHLCV struct {
    StockCode string    `json:"stock_code"`
    Date      time.Time `json:"date"`
    Open      int64     `json:"open"`
    High      int64     `json:"high"`
    Low       int64     `json:"low"`
    Close     int64     `json:"close"`
    Volume    int64     `json:"volume"`
    Value     int64     `json:"value"`       // 거래대금
    AdjClose  float64   `json:"adj_close"`   // 수정종가
    AdjFactor float64   `json:"adj_factor"`  // 수정계수 (액면분할, 배당 등)
}
```

**커버리지 목표**: 100%
**갱신 주기**: Daily (장 마감 후)
**보관 기간**: 10년+

---

### 2.2 시가총액

| 필드 | 설명 | 필수 |
|------|------|------|
| `market_cap` | 시가총액 | ✅ |
| `shares_outstanding` | 상장주식수 | ✅ |
| `shares_float` | 유동주식수 | - |
| `float_ratio` | 유동비율 | - |

```go
type MarketCap struct {
    StockCode         string    `json:"stock_code"`
    Date              time.Time `json:"date"`
    MarketCap         int64     `json:"market_cap"`          // 시가총액 (원)
    SharesOutstanding int64     `json:"shares_outstanding"`  // 상장주식수
    SharesFloat       int64     `json:"shares_float"`        // 유동주식수
    FloatRatio        float64   `json:"float_ratio"`         // 유동비율
}
```

**커버리지 목표**: 95%+
**갱신 주기**: Daily

---

### 2.3 시장 지표

| 지표 | 설명 | 용도 |
|------|------|------|
| KOSPI | 코스피 지수 | 시장 상황 판단 |
| KOSDAQ | 코스닥 지수 | 시장 상황 판단 |
| USD/KRW | 원/달러 환율 | 외국인 수급 예측 |
| VIX | 변동성 지수 | 리스크 관리 |

```go
type MarketIndex struct {
    IndexCode string    `json:"index_code"`
    Date      time.Time `json:"date"`
    Close     float64   `json:"close"`
    Change    float64   `json:"change"`
    ChangeP   float64   `json:"change_p"`   // 등락률
    Volume    int64     `json:"volume"`
}
```

**커버리지 목표**: 100%
**갱신 주기**: Daily

---

## 3. Fundamental Data (재무 데이터)

### 3.1 재무제표

| 항목 | 설명 | 주기 |
|------|------|------|
| 매출액 | Revenue | 분기 |
| 영업이익 | Operating Income | 분기 |
| 당기순이익 | Net Income | 분기 |
| 자산총계 | Total Assets | 분기 |
| 부채총계 | Total Liabilities | 분기 |
| 자본총계 | Total Equity | 분기 |
| 영업현금흐름 | Operating Cash Flow | 분기 |

```go
type FinancialStatement struct {
    StockCode   string    `json:"stock_code"`
    FiscalYear  int       `json:"fiscal_year"`   // 2024
    FiscalQtr   int       `json:"fiscal_qtr"`    // 1,2,3,4
    ReportType  string    `json:"report_type"`   // annual, quarterly

    // 손익계산서
    Revenue         int64 `json:"revenue"`
    OperatingIncome int64 `json:"operating_income"`
    NetIncome       int64 `json:"net_income"`

    // 재무상태표
    TotalAssets      int64 `json:"total_assets"`
    TotalLiabilities int64 `json:"total_liabilities"`
    TotalEquity      int64 `json:"total_equity"`

    // 현금흐름표
    OperatingCF int64 `json:"operating_cf"`
    InvestingCF int64 `json:"investing_cf"`
    FinancingCF int64 `json:"financing_cf"`
}
```

**커버리지 목표**: 80%+
**갱신 주기**: 분기 (실적발표 후)

---

### 3.2 밸류에이션 지표

| 지표 | 산식 | 용도 |
|------|------|------|
| PER | 주가 / EPS | 가치 평가 |
| PBR | 주가 / BPS | 가치 평가 |
| PSR | 시총 / 매출 | 성장주 평가 |
| EV/EBITDA | EV / EBITDA | 기업가치 평가 |
| DIV | 배당수익률 | 배당주 평가 |

```go
type Valuation struct {
    StockCode string    `json:"stock_code"`
    Date      time.Time `json:"date"`

    PER       float64   `json:"per"`        // Price/Earnings
    PBR       float64   `json:"pbr"`        // Price/Book
    PSR       float64   `json:"psr"`        // Price/Sales
    PCR       float64   `json:"pcr"`        // Price/Cash Flow
    EV_EBITDA float64   `json:"ev_ebitda"`  // EV/EBITDA
    DivYield  float64   `json:"div_yield"`  // 배당수익률

    // 기초 데이터
    EPS       float64   `json:"eps"`        // 주당순이익
    BPS       float64   `json:"bps"`        // 주당순자산
    DPS       float64   `json:"dps"`        // 주당배당금
}
```

**커버리지 목표**: 80%+
**갱신 주기**: Daily (종가 기준 재계산)

---

### 3.3 퀄리티/성장 지표

| 지표 | 산식 | 용도 |
|------|------|------|
| ROE | 순이익 / 자본 | 수익성 |
| ROA | 순이익 / 자산 | 효율성 |
| 부채비율 | 부채 / 자본 | 안정성 |
| 매출성장률 | YoY Revenue Growth | 성장성 |
| 이익성장률 | YoY Earnings Growth | 성장성 |

```go
type QualityMetrics struct {
    StockCode string    `json:"stock_code"`
    Date      time.Time `json:"date"`

    // 수익성
    ROE       float64   `json:"roe"`
    ROA       float64   `json:"roa"`
    OPM       float64   `json:"opm"`        // 영업이익률
    NPM       float64   `json:"npm"`        // 순이익률

    // 안정성
    DebtRatio float64   `json:"debt_ratio"` // 부채비율
    CurrentR  float64   `json:"current_r"`  // 유동비율

    // 성장성
    RevenueGrowth  float64 `json:"revenue_growth"`  // 매출성장률 YoY
    EarningsGrowth float64 `json:"earnings_growth"` // 이익성장률 YoY
}
```

**커버리지 목표**: 80%+
**갱신 주기**: 분기

---

## 4. Alternative Data (대체 데이터)

### 4.1 투자자별 매매동향 (수급)

| 투자자 | 코드 | 중요도 |
|--------|------|--------|
| 외국인 | `foreign` | ⭐⭐⭐ |
| 기관계 | `institution` | ⭐⭐⭐ |
| 개인 | `individual` | ⭐⭐ |
| 금융투자 | `financial` | ⭐ |
| 보험 | `insurance` | ⭐ |
| 투신 | `trust` | ⭐ |
| 연기금 | `pension` | ⭐⭐ |

```go
type InvestorFlow struct {
    StockCode string    `json:"stock_code"`
    Date      time.Time `json:"date"`

    // 순매수 금액 (백만원)
    Foreign     int64 `json:"foreign"`      // 외국인
    Institution int64 `json:"institution"`  // 기관계
    Individual  int64 `json:"individual"`   // 개인

    // 기관 상세
    Financial   int64 `json:"financial"`    // 금융투자
    Insurance   int64 `json:"insurance"`    // 보험
    Trust       int64 `json:"trust"`        // 투신
    Pension     int64 `json:"pension"`      // 연기금
    Bank        int64 `json:"bank"`         // 은행
    OtherInst   int64 `json:"other_inst"`   // 기타법인

    // 순매수 수량 (주)
    ForeignQty     int64 `json:"foreign_qty"`
    InstitutionQty int64 `json:"institution_qty"`
    IndividualQty  int64 `json:"individual_qty"`
}
```

**커버리지 목표**: 80%+
**갱신 주기**: Daily
**중요**: 외국인/기관 수급은 핵심 시그널

---

### 4.2 공시 데이터

| 공시 유형 | 코드 | 중요도 |
|----------|------|--------|
| 실적공시 | `earnings` | ⭐⭐⭐ |
| 주요사항보고 | `material` | ⭐⭐⭐ |
| 지분공시 | `stake` | ⭐⭐ |
| 자사주 | `treasury` | ⭐⭐ |
| 배당 | `dividend` | ⭐⭐ |
| 유상증자 | `rights` | ⭐⭐⭐ |
| 무상증자 | `bonus` | ⭐ |

```go
type Disclosure struct {
    StockCode    string    `json:"stock_code"`
    DisclosureID string    `json:"disclosure_id"` // DART 고유번호
    Date         time.Time `json:"date"`
    Title        string    `json:"title"`
    Type         string    `json:"type"`          // earnings, material, stake...
    Importance   int       `json:"importance"`    // 1=높음, 2=중간, 3=낮음
    URL          string    `json:"url"`

    // 파싱된 주요 내용 (유형별로 다름)
    ParsedData   map[string]interface{} `json:"parsed_data"`
}
```

**커버리지 목표**: 70%+
**갱신 주기**: Real-time / Daily
**소스**: DART OpenAPI

---

## 5. 데이터 소스 맵핑

| 데이터 | Primary Source | Backup Source | API |
|--------|---------------|---------------|-----|
| 종목 마스터 | KRX | - | KRX OpenAPI |
| 가격/거래량 | Naver Finance | KIS | Naver Chart API |
| 시가총액 | Naver Finance | KRX | Naver |
| 재무제표 | DART | Naver Finance | DART OpenAPI |
| 밸류에이션 | Naver Finance | 직접계산 | Naver |
| 투자자 수급 | Naver Finance | KRX | Naver |
| 공시 | DART | - | DART OpenAPI |
| 시장지표 | KRX | Yahoo | KRX/Yahoo API |

---

## 6. 데이터 품질 기준

### 커버리지 요구사항

| 데이터 | 필수 | 최소 커버리지 | 목표 커버리지 |
|--------|------|--------------|--------------|
| 가격 | ✅ | 100% | 100% |
| 거래량 | ✅ | 100% | 100% |
| 시가총액 | ✅ | 90% | 95%+ |
| 재무제표 | ⚠️ | 70% | 80%+ |
| 밸류에이션 | ⚠️ | 70% | 80%+ |
| 투자자 수급 | ⚠️ | 70% | 80%+ |
| 공시 | ⚠️ | 60% | 70%+ |

### 품질 검증 룰

```go
type DataQualityRules struct {
    // 가격 검증
    MaxPriceChange float64 `yaml:"max_price_change"` // 30% (상한가)
    MinPrice       int64   `yaml:"min_price"`        // 100원 이상

    // 거래량 검증
    MaxVolumeSpike float64 `yaml:"max_volume_spike"` // 전일 대비 50배

    // 재무 검증
    MaxPER         float64 `yaml:"max_per"`          // 200 이상 이상치
    MinPBR         float64 `yaml:"min_pbr"`          // 0 미만 이상치
}
```

---

## 7. 갱신 스케줄

| 데이터 | 갱신 시점 | 주기 |
|--------|----------|------|
| 가격/거래량 | 장 마감 후 (15:40~) | Daily |
| 시가총액 | 장 마감 후 | Daily |
| 투자자 수급 | 장 마감 후 (16:00~) | Daily |
| 재무제표 | 실적발표 후 | Quarterly |
| 공시 | 실시간 | Real-time |
| 종목 마스터 | 영업일 시작 전 | Daily |
| 지수 구성 | 정기변경 시 | Quarterly |

```yaml
# 수집 스케줄 예시
schedule:
  daily:
    - name: prices
      time: "15:45"
      priority: 1
    - name: market_cap
      time: "15:50"
      priority: 2
    - name: investor_flow
      time: "16:05"
      priority: 3

  quarterly:
    - name: financials
      trigger: "earnings_release"
    - name: index_constituents
      trigger: "rebalance_date"
```

---

## 8. 데이터 파이프라인 활용

### S2 Signals에서 활용

| 시그널 유형 | 사용 데이터 |
|------------|------------|
| **Momentum** | 가격, 거래량 |
| **Value** | 밸류에이션 (PER, PBR, PSR) |
| **Quality** | 재무제표 (ROE, 부채비율) |
| **Growth** | 재무제표 (매출/이익 성장률) |
| **Flow** | 투자자 수급 (외국인/기관) |
| **Event** | 공시 |

### 시그널-데이터 의존성

```
┌──────────────────────────────────────────────────────────┐
│                    S2 SIGNALS                            │
├──────────────────────────────────────────────────────────┤
│  Momentum ←── 가격, 거래량                               │
│  Value    ←── PER, PBR, PSR, EV/EBITDA                 │
│  Quality  ←── ROE, ROA, 부채비율                        │
│  Growth   ←── 매출성장률, 이익성장률                     │
│  Flow     ←── 외국인 순매수, 기관 순매수                 │
│  Event    ←── 공시 (실적, 지분, 자사주 등)              │
└──────────────────────────────────────────────────────────┘
```

---

**Prev**: [Contracts](./contracts.md)
**Next**: [Backend Overview](../backend/folder-structure.md)
