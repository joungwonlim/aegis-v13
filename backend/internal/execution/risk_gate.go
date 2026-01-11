package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/internal/risk"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// =============================================================================
// RiskGate - S6 사전 리스크 게이트
// =============================================================================

// GateMode 게이트 동작 모드
type GateMode string

const (
	GateModeShadow  GateMode = "shadow"  // 로깅만, 실제 차단 안함
	GateModeEnforce GateMode = "enforce" // 실제 차단
	GateModeOff     GateMode = "off"     // 비활성화
)

// RiskGate S6 리스크 게이트
// ⭐ SSOT: 주문 전 리스크 체크는 여기서만
type RiskGate struct {
	engine     *risk.Engine
	repo       *Repository
	priceRepo  PriceRepository
	logger     *logger.Logger
	mode       GateMode
	runID      string
}

// PriceRepository 가격 조회 인터페이스 (의존성 역전)
type PriceRepository interface {
	GetHistoricalReturns(ctx context.Context, codes []string, days int) (map[string][]float64, error)
}

// RiskGateConfig 리스크 게이트 설정
type RiskGateConfig struct {
	Mode         GateMode
	LookbackDays int // 과거 데이터 일수 (기본: 200)
}

// DefaultRiskGateConfig 기본 설정
func DefaultRiskGateConfig() RiskGateConfig {
	return RiskGateConfig{
		Mode:         GateModeShadow, // 기본: Shadow 모드
		LookbackDays: 200,
	}
}

// NewRiskGate 새 리스크 게이트 생성
func NewRiskGate(
	engine *risk.Engine,
	repo *Repository,
	priceRepo PriceRepository,
	logger *logger.Logger,
	config RiskGateConfig,
) *RiskGate {
	return &RiskGate{
		engine:    engine,
		repo:      repo,
		priceRepo: priceRepo,
		logger:    logger,
		mode:      config.Mode,
		runID:     fmt.Sprintf("gate_%s", time.Now().Format("20060102_150405")),
	}
}

// =============================================================================
// Gate Check
// =============================================================================

// GateCheckInput 게이트 체크 입력
type GateCheckInput struct {
	Orders          []contracts.Order
	CurrentHoldings []risk.Holding
	TargetHoldings  []risk.Holding
}

// GateCheckResult 게이트 체크 결과
type GateCheckResult struct {
	Passed       bool                 `json:"passed"`
	Mode         GateMode             `json:"mode"`
	WouldBlock   bool                 `json:"would_block"`   // Shadow 모드에서 차단됐을지 여부
	RiskCheck    *risk.RiskCheckResult `json:"risk_check"`
	BlockedOrders []string            `json:"blocked_orders"` // 차단된 주문 코드
	Message      string               `json:"message"`
	CheckedAt    time.Time            `json:"checked_at"`
	RunID        string               `json:"run_id"`
}

// Check 주문 전 리스크 체크 실행
func (g *RiskGate) Check(ctx context.Context, input GateCheckInput) (*GateCheckResult, error) {
	result := &GateCheckResult{
		Mode:      g.mode,
		CheckedAt: time.Now(),
		RunID:     g.runID,
	}

	// 게이트가 꺼져있으면 통과
	if g.mode == GateModeOff {
		result.Passed = true
		result.Message = "Risk gate is disabled"
		return result, nil
	}

	// 1. 종목 코드 추출
	codes := make([]string, 0)
	for _, h := range input.TargetHoldings {
		codes = append(codes, h.Code)
	}

	// 2. 과거 수익률 조회
	historicalReturns, err := g.priceRepo.GetHistoricalReturns(ctx, codes, 200)
	if err != nil {
		g.logger.WithFields(map[string]interface{}{
			"error": err,
		}).Warn("Failed to get historical returns, passing gate")
		result.Passed = true
		result.Message = "Historical data unavailable, gate passed"
		return result, nil
	}

	// 3. 리스크 한도 체크
	riskCheck, err := g.engine.CheckRiskLimits(ctx, input.TargetHoldings, historicalReturns)
	if err != nil {
		return nil, fmt.Errorf("risk check failed: %w", err)
	}
	result.RiskCheck = riskCheck

	// 4. 결과 판정
	if riskCheck.Passed {
		result.Passed = true
		result.WouldBlock = false
		result.Message = "All risk checks passed"
	} else {
		result.WouldBlock = true
		result.BlockedOrders = g.getBlockedOrderCodes(input.Orders, riskCheck)
		result.Message = g.buildBlockMessage(riskCheck)

		// Shadow 모드면 통과, Enforce 모드면 차단
		if g.mode == GateModeShadow {
			result.Passed = true
			g.logShadowBlock(ctx, result, riskCheck)
		} else {
			result.Passed = false
		}
	}

	// 5. 결과 저장 (Shadow 모드용 분석)
	if err := g.saveGateResult(ctx, result); err != nil {
		g.logger.WithFields(map[string]interface{}{
			"error": err,
		}).Warn("Failed to save gate result")
	}

	return result, nil
}

// =============================================================================
// Shadow Mode Logging
// =============================================================================

// logShadowBlock Shadow 모드에서 차단 이벤트 로깅
func (g *RiskGate) logShadowBlock(ctx context.Context, result *GateCheckResult, riskCheck *risk.RiskCheckResult) {
	fields := map[string]interface{}{
		"run_id":         g.runID,
		"mode":           "shadow",
		"would_block":    true,
		"violation_count": len(riskCheck.Violations),
	}

	// 위반 상세 정보 추가
	for i, v := range riskCheck.Violations {
		fields[fmt.Sprintf("violation_%d_type", i)] = v.Type
		fields[fmt.Sprintf("violation_%d_limit", i)] = v.Limit
		fields[fmt.Sprintf("violation_%d_actual", i)] = v.Actual
		fields[fmt.Sprintf("violation_%d_severity", i)] = v.Severity
	}

	// 리스크 메트릭 추가
	fields["var_95"] = riskCheck.Metrics.PortfolioVaR95
	fields["var_99"] = riskCheck.Metrics.PortfolioVaR99
	fields["max_single_exposure"] = riskCheck.Metrics.MaxSingleExposure
	fields["concentration_ratio"] = riskCheck.Metrics.ConcentrationRatio

	g.logger.WithFields(fields).Warn("🚨 SHADOW BLOCK: Would have blocked orders")
}

// =============================================================================
// Helper Methods
// =============================================================================

// getBlockedOrderCodes 차단된 주문 코드 추출
func (g *RiskGate) getBlockedOrderCodes(orders []contracts.Order, riskCheck *risk.RiskCheckResult) []string {
	// 단일 종목 익스포저 초과 시 해당 종목만 차단
	blocked := make([]string, 0)

	for _, v := range riskCheck.Violations {
		if v.Type == "SINGLE_EXPOSURE_LIMIT" {
			// 가장 큰 비중 종목을 차단 대상으로
			for _, order := range orders {
				if order.Side == contracts.OrderSideBuy {
					blocked = append(blocked, order.Code)
					break
				}
			}
		}
	}

	// 포트폴리오 전체 VaR 초과 시 모든 매수 주문 차단
	for _, v := range riskCheck.Violations {
		if v.Type == "VAR_95_LIMIT" || v.Type == "VAR_99_LIMIT" {
			for _, order := range orders {
				if order.Side == contracts.OrderSideBuy {
					blocked = append(blocked, order.Code)
				}
			}
			break
		}
	}

	return blocked
}

// buildBlockMessage 차단 메시지 생성
func (g *RiskGate) buildBlockMessage(riskCheck *risk.RiskCheckResult) string {
	if len(riskCheck.Violations) == 0 {
		return "No violations"
	}

	msg := fmt.Sprintf("Risk limit violations (%d): ", len(riskCheck.Violations))
	for i, v := range riskCheck.Violations {
		if i > 0 {
			msg += ", "
		}
		msg += fmt.Sprintf("%s (%.2f%% > %.2f%%)", v.Type, v.Actual*100, v.Limit*100)
	}
	return msg
}

// =============================================================================
// Persistence
// =============================================================================

// GateEvent 게이트 이벤트 (DB 저장용)
type GateEvent struct {
	ID            int64     `json:"id"`
	RunID         string    `json:"run_id"`
	Mode          GateMode  `json:"mode"`
	Passed        bool      `json:"passed"`
	WouldBlock    bool      `json:"would_block"`
	ViolationCount int      `json:"violation_count"`
	VaR95         float64   `json:"var_95"`
	VaR99         float64   `json:"var_99"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
}

// saveGateResult 게이트 결과 저장
func (g *RiskGate) saveGateResult(ctx context.Context, result *GateCheckResult) error {
	if g.repo == nil {
		return nil // Repository가 없으면 저장 생략
	}

	event := GateEvent{
		RunID:      result.RunID,
		Mode:       result.Mode,
		Passed:     result.Passed,
		WouldBlock: result.WouldBlock,
		Message:    result.Message,
		CreatedAt:  result.CheckedAt,
	}

	if result.RiskCheck != nil {
		event.ViolationCount = len(result.RiskCheck.Violations)
		event.VaR95 = result.RiskCheck.Metrics.PortfolioVaR95
		event.VaR99 = result.RiskCheck.Metrics.PortfolioVaR99
	}

	return g.repo.SaveGateEvent(ctx, &event)
}

// =============================================================================
// Mode Management
// =============================================================================

// SetMode 게이트 모드 변경
func (g *RiskGate) SetMode(mode GateMode) {
	g.mode = mode
	g.logger.WithFields(map[string]interface{}{
		"mode": mode,
	}).Info("Risk gate mode changed")
}

// GetMode 현재 게이트 모드 조회
func (g *RiskGate) GetMode() GateMode {
	return g.mode
}

// IsEnabled 게이트 활성화 여부
func (g *RiskGate) IsEnabled() bool {
	return g.mode != GateModeOff
}

// IsShadowMode Shadow 모드 여부
func (g *RiskGate) IsShadowMode() bool {
	return g.mode == GateModeShadow
}
