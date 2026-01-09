package contracts

import (
	"context"
	"time"
)

// QualityGate checks data quality (S0)
// ⭐ SSOT: S0 데이터 품질 검증 인터페이스
type QualityGate interface {
	Check(ctx context.Context, date time.Time) (*DataQualitySnapshot, error)
}

// UniverseBuilder creates investable universe (S1)
// ⭐ SSOT: S1 유니버스 생성 인터페이스
type UniverseBuilder interface {
	Build(ctx context.Context, snapshot *DataQualitySnapshot) (*Universe, error)
}

// SignalBuilder generates signals for stocks (S2)
// ⭐ SSOT: S2 시그널 생성 인터페이스
type SignalBuilder interface {
	Build(ctx context.Context, universe *Universe) (*SignalSet, error)
}

// Screener performs initial filtering (S3)
// ⭐ SSOT: S3 스크리닝 인터페이스
type Screener interface {
	Screen(ctx context.Context, signals *SignalSet) ([]string, error)
}

// Ranker ranks stocks by composite score (S4)
// ⭐ SSOT: S4 랭킹 인터페이스
type Ranker interface {
	Rank(ctx context.Context, codes []string, signals *SignalSet) ([]RankedStock, error)
}

// PortfolioConstructor constructs target portfolio (S5)
// ⭐ SSOT: S5 포트폴리오 구성 인터페이스
type PortfolioConstructor interface {
	Construct(ctx context.Context, ranked []RankedStock) (*TargetPortfolio, error)
}

// ExecutionPlanner plans order execution (S6)
// ⭐ SSOT: S6 주문 실행 계획 인터페이스
type ExecutionPlanner interface {
	Plan(ctx context.Context, target *TargetPortfolio) ([]Order, error)
}

// Auditor analyzes performance (S7)
// ⭐ SSOT: S7 성과 분석 인터페이스
type Auditor interface {
	Analyze(ctx context.Context, period string) (*PerformanceReport, error)
}
