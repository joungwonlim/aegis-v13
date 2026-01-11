package contracts

// Pipeline Stage 정의 (SSOT)
// 모든 로그, 스냅샷, DB row에서 이 상수를 사용해야 함
//
// 파이프라인 흐름:
//   S0 → S1 → S2 → S3 → S4 → S5 → S6 → S7
//   Data  Universe  Signals  Screener  Ranker  Portfolio  Execution  Audit

// Stage represents a pipeline stage
type Stage string

const (
	// StageDataQuality S0: 데이터 수집 및 품질 검증
	// 책임: 외부 데이터 수집, 품질 게이트, 커버리지 검증
	// 위치: internal/s0_data/
	StageDataQuality Stage = "S0_DATA_QUALITY"

	// StageUniverse S1: 투자 가능 종목 (Universe)
	// 책임: 거래 가능 종목 필터링, 유동성/시총 기준 적용
	// 위치: internal/s1_universe/
	StageUniverse Stage = "S1_UNIVERSE"

	// StageSignals S2: 팩터/시그널 계산
	// 책임: 6개 팩터 계산 (Momentum, Technical, Value, Quality, Flow, Event)
	// 위치: internal/s2_signals/
	StageSignals Stage = "S2_SIGNALS"

	// StageScreener S3: Hard Cut 필터링
	// 책임: 재무 지표 기반 부실 종목 제거 (PER/PBR/ROE)
	// 위치: internal/selection/screener.go
	StageScreener Stage = "S3_SCREENER"

	// StageRanker S4: 종합 점수 산출 및 순위 부여
	// 책임: 가중치 적용, 점수 계산, Top N 선별
	// 위치: internal/selection/ranker.go
	StageRanker Stage = "S4_RANKER"

	// StagePortfolio S5: 포트폴리오 구성
	// 책임: 목표 비중 결정, 제약 조건 적용, 리밸런싱
	// 위치: internal/portfolio/
	StagePortfolio Stage = "S5_PORTFOLIO"

	// StageExecution S6: 주문 실행
	// 책임: 주문 생성, 분할 주문, 브로커 연동, 리스크 게이트
	// 위치: internal/execution/
	StageExecution Stage = "S6_EXECUTION"

	// StageAudit S7: 성과 분석 및 감사
	// 책임: 수익률 계산, 귀인분석, 리스크 리포트, 의사결정 추적
	// 위치: internal/audit/
	StageAudit Stage = "S7_AUDIT"
)

// String returns the stage name
func (s Stage) String() string {
	return string(s)
}

// ShortName returns abbreviated stage name (e.g., "S0", "S1")
func (s Stage) ShortName() string {
	switch s {
	case StageDataQuality:
		return "S0"
	case StageUniverse:
		return "S1"
	case StageSignals:
		return "S2"
	case StageScreener:
		return "S3"
	case StageRanker:
		return "S4"
	case StagePortfolio:
		return "S5"
	case StageExecution:
		return "S6"
	case StageAudit:
		return "S7"
	default:
		return "UNKNOWN"
	}
}

// Description returns Korean description of the stage
func (s Stage) Description() string {
	switch s {
	case StageDataQuality:
		return "데이터 수집/품질 검증"
	case StageUniverse:
		return "투자 가능 종목"
	case StageSignals:
		return "팩터/시그널 계산"
	case StageScreener:
		return "Hard Cut 필터링"
	case StageRanker:
		return "종합 점수/순위"
	case StagePortfolio:
		return "포트폴리오 구성"
	case StageExecution:
		return "주문 실행"
	case StageAudit:
		return "성과 분석/감사"
	default:
		return "알 수 없음"
	}
}

// AllStages returns all pipeline stages in order
func AllStages() []Stage {
	return []Stage{
		StageDataQuality,
		StageUniverse,
		StageSignals,
		StageScreener,
		StageRanker,
		StagePortfolio,
		StageExecution,
		StageAudit,
	}
}

// IsValidStage checks if a stage string is valid
func IsValidStage(s string) bool {
	for _, stage := range AllStages() {
		if string(stage) == s {
			return true
		}
	}
	return false
}

// PipelineResult represents the result of a pipeline stage execution
type PipelineResult struct {
	Stage       Stage                  `json:"stage"`
	Success     bool                   `json:"success"`
	InputCount  int                    `json:"input_count"`
	OutputCount int                    `json:"output_count"`
	Duration    int64                  `json:"duration_ms"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PipelineSnapshot records the state at each pipeline stage for reproducibility
type PipelineSnapshot struct {
	RunID     string                    `json:"run_id"`
	Stage     Stage                     `json:"stage"`
	Timestamp int64                     `json:"timestamp"`
	Inputs    map[string]interface{}    `json:"inputs"`
	Outputs   map[string]interface{}    `json:"outputs"`
	Config    map[string]interface{}    `json:"config"`
	Results   map[string]PipelineResult `json:"results,omitempty"`
}
