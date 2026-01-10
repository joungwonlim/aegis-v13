// Package execution - exit_rules.go
// v1.2.0: ATR 기반 동적 청산 전략
// - 익절: ATR 기반 3단계 (TP1:+6%/25%, TP2:+10%/25%, TP3:+15%/20%)
// - 손절: 1차 -3% (50%), 2차 -5% (전량)
// - Stop Floor: TP1 이후 손익분기점 + 0.6% 보호
// - HWM Trailing: 잔여 30%는 최고가 기준 트레일링
package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// =============================================================================
// ATR Provider Interface
// =============================================================================

// ATRProvider ATR 조회 인터페이스
type ATRProvider interface {
	GetATR(ctx context.Context, code string, period int) (float64, error)
}

// DBATRProvider DB에서 ATR 계산
type DBATRProvider struct {
	pool *pgxpool.Pool
}

// NewDBATRProvider 새 DB ATR Provider 생성
func NewDBATRProvider(pool *pgxpool.Pool) *DBATRProvider {
	return &DBATRProvider{pool: pool}
}

// GetATR ATR14 계산 (True Range의 14일 SMA)
func (p *DBATRProvider) GetATR(ctx context.Context, code string, period int) (float64, error) {
	query := `
		WITH daily_data AS (
			SELECT
				trade_date,
				high,
				low,
				close,
				LAG(close) OVER (ORDER BY trade_date) as prev_close
			FROM data.daily_prices
			WHERE stock_code = $1
			ORDER BY trade_date DESC
			LIMIT $2 + 1
		),
		true_ranges AS (
			SELECT
				GREATEST(
					high - low,
					ABS(high - COALESCE(prev_close, close)),
					ABS(low - COALESCE(prev_close, close))
				) as true_range
			FROM daily_data
			WHERE prev_close IS NOT NULL
			LIMIT $2
		)
		SELECT COALESCE(AVG(true_range), 0) FROM true_ranges
	`

	var atr float64
	err := p.pool.QueryRow(ctx, query, code, period).Scan(&atr)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate ATR for %s: %w", code, err)
	}
	return atr, nil
}

// =============================================================================
// Price Provider Interface
// =============================================================================

// PriceProvider 현재가 조회 인터페이스
type PriceProvider interface {
	GetCurrentPrice(ctx context.Context, code string) (int64, error)
}

// =============================================================================
// Exit Notifier Interface
// =============================================================================

// ExitNotifier 청산 알림 인터페이스
type ExitNotifier interface {
	NotifyExitSignal(ctx context.Context, signal *contracts.ExitSignal) error
}

// =============================================================================
// Position Monitor
// ⭐ SSOT: 포지션 모니터링 및 청산 신호 생성은 여기서만
// =============================================================================

// PositionMonitor 포지션 모니터링 및 자동 청산
type PositionMonitor struct {
	config      *contracts.ExitRulesConfig
	priceFunc   PriceProvider
	atrProvider ATRProvider
	notifier    ExitNotifier
	pool        *pgxpool.Pool
	logger      *logger.Logger

	positions     map[string]*contracts.MonitoredPosition
	recentSignals []*contracts.ExitSignal
	mu            sync.RWMutex
	stopCh        chan struct{}
	isRunning     bool
	autoSell      bool
}

// NewPositionMonitor 새 포지션 모니터 생성
func NewPositionMonitor(
	config *contracts.ExitRulesConfig,
	priceFunc PriceProvider,
	atrProvider ATRProvider,
	pool *pgxpool.Pool,
	logger *logger.Logger,
) *PositionMonitor {
	if config == nil {
		config = contracts.DefaultExitRulesConfig()
	}
	return &PositionMonitor{
		config:        config,
		priceFunc:     priceFunc,
		atrProvider:   atrProvider,
		pool:          pool,
		logger:        logger,
		positions:     make(map[string]*contracts.MonitoredPosition),
		recentSignals: make([]*contracts.ExitSignal, 0, 50),
		stopCh:        make(chan struct{}),
		autoSell:      true,
	}
}

// SetNotifier 청산 알림 설정
func (pm *PositionMonitor) SetNotifier(notifier ExitNotifier) {
	pm.notifier = notifier
}

// SetAutoSell 자동 매도 설정
func (pm *PositionMonitor) SetAutoSell(enabled bool) {
	pm.autoSell = enabled
	pm.logger.WithFields(map[string]interface{}{
		"auto_sell": enabled,
	}).Info("Auto-sell setting changed")
}

// =============================================================================
// Position Management
// =============================================================================

// AddPosition 모니터링할 포지션 추가
func (pm *PositionMonitor) AddPosition(ctx context.Context, pos *contracts.MonitoredPosition) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 기본 초기화
	pos.ReferencePrice = pos.EntryPrice
	pos.HighestPrice = pos.EntryPrice
	pos.HighestPriceTime = pos.EntryTime
	if pos.RemainingQuantity == 0 {
		pos.RemainingQuantity = pos.InitialQuantity
	}
	if pos.State == "" {
		pos.State = contracts.PositionStateOpen
	}
	pos.LastUpdated = time.Now()

	// ATR 기반 트리거 가격 계산
	if pm.config.UseATRBased {
		if pos.ATRPercent == 0 && pm.atrProvider != nil {
			atr, err := pm.atrProvider.GetATR(ctx, pos.Code, 14)
			if err != nil {
				pm.logger.WithFields(map[string]interface{}{
					"code":  pos.Code,
					"error": err.Error(),
				}).Warn("Failed to get ATR, using default 2%")
				pos.ATRPercent = 2.0
			} else if pos.EntryPrice > 0 {
				pos.ATRPercent = (atr / float64(pos.EntryPrice)) * 100
			}
		}
		pm.calculateTriggerPrices(pos)
	}

	pm.positions[pos.Code] = pos

	pm.logger.WithFields(map[string]interface{}{
		"code":        pos.Code,
		"entry_price": pos.EntryPrice,
		"quantity":    pos.InitialQuantity,
		"atr_percent": pos.ATRPercent,
		"tp1_trigger": pos.TP1TriggerPrice,
		"tp2_trigger": pos.TP2TriggerPrice,
		"tp3_trigger": pos.TP3TriggerPrice,
	}).Info("Position added for monitoring")

	return nil
}

// calculateTriggerPrices ATR 기반 트리거 가격 계산
func (pm *PositionMonitor) calculateTriggerPrices(pos *contracts.MonitoredPosition) {
	atr := pos.ATRPercent / 100

	// TP1: ATR * 1.5, clamp [6%, 8%]
	tp1Pct := clamp(atr*pm.config.TP1ATRMultiplier, pm.config.TP1MinPercent/100, pm.config.TP1MaxPercent/100)
	pos.TP1TriggerPrice = int64(float64(pos.EntryPrice) * (1 + tp1Pct))

	// TP2: ATR * 2.5, clamp [10%, 12%]
	tp2Pct := clamp(atr*pm.config.TP2ATRMultiplier, pm.config.TP2MinPercent/100, pm.config.TP2MaxPercent/100)
	pos.TP2TriggerPrice = int64(float64(pos.EntryPrice) * (1 + tp2Pct))

	// TP3: ATR * 3.5, clamp [15%, 18%]
	tp3Pct := clamp(atr*pm.config.TP3ATRMultiplier, pm.config.TP3MinPercent/100, pm.config.TP3MaxPercent/100)
	pos.TP3TriggerPrice = int64(float64(pos.EntryPrice) * (1 + tp3Pct))
}

// clamp 값을 min/max 범위로 제한
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// RemovePosition 포지션 제거
func (pm *PositionMonitor) RemovePosition(code string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.positions, code)
	pm.logger.WithFields(map[string]interface{}{
		"code": code,
	}).Info("Position removed from monitoring")
}

// GetPositions 모니터링 중인 포지션 목록
func (pm *PositionMonitor) GetPositions() []*contracts.MonitoredPosition {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*contracts.MonitoredPosition, 0, len(pm.positions))
	for _, p := range pm.positions {
		result = append(result, p)
	}
	return result
}

// GetRecentSignals 최근 청산 신호 조회
func (pm *PositionMonitor) GetRecentSignals(limit int) []*contracts.ExitSignal {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if limit <= 0 || limit > len(pm.recentSignals) {
		limit = len(pm.recentSignals)
	}

	result := make([]*contracts.ExitSignal, limit)
	for i := 0; i < limit; i++ {
		result[i] = pm.recentSignals[len(pm.recentSignals)-1-i]
	}
	return result
}

// =============================================================================
// Price Monitoring & Exit Signal Generation
// =============================================================================

// CheckPosition 개별 포지션 체크 및 청산 신호 생성
func (pm *PositionMonitor) CheckPosition(ctx context.Context, pos *contracts.MonitoredPosition) ([]*contracts.ExitSignal, error) {
	// 현재가 조회
	currentPrice, err := pm.priceFunc.GetCurrentPrice(ctx, pos.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get price for %s: %w", pos.Code, err)
	}

	// 포지션 상태 업데이트
	pos.CurrentPrice = currentPrice
	pos.LastUpdated = time.Now()

	// 최고가 업데이트 (HWM)
	if currentPrice > pos.HighestPrice {
		pos.HighestPrice = currentPrice
		pos.HighestPriceTime = time.Now()

		// TP3 이후: HWM 트레일링 스탑 갱신
		if pos.State == contracts.PositionStateTP3Done {
			pm.updateTrailStopPrice(pos)
		}
	}

	// 손익률 계산
	pos.UnrealizedPnL = float64(currentPrice-pos.EntryPrice) / float64(pos.EntryPrice) * 100
	pos.RefPnL = float64(currentPrice-pos.ReferencePrice) / float64(pos.ReferencePrice) * 100

	// ATR 기반 모드
	if pm.config.UseATRBased && pos.TP1TriggerPrice > 0 {
		return pm.checkPositionATRBased(pos, currentPrice)
	}

	return nil, nil
}

// checkPositionATRBased ATR 기반 청산 조건 체크
func (pm *PositionMonitor) checkPositionATRBased(pos *contracts.MonitoredPosition, currentPrice int64) ([]*contracts.ExitSignal, error) {
	// ==========================================================================
	// 청산 조건 체크 (우선순위: 2차손절 > 1차손절 > StopFloor > HWMTrail > TP)
	// ==========================================================================

	// 1. 2차 손절: 진입가 대비 -5% (전량 청산)
	secondStopPrice := int64(float64(pos.EntryPrice) * (1 + pm.config.SecondStopPercent/100))
	if currentPrice <= secondStopPrice {
		return []*contracts.ExitSignal{pm.createSecondStopSignal(pos, currentPrice)}, nil
	}

	// 2. 1차 손절: 진입가 대비 -3% (50% 청산) - 아직 1차 손절 안했을 때만
	if !pos.FirstStopTriggered {
		firstStopPrice := int64(float64(pos.EntryPrice) * (1 + pm.config.FirstStopPercent/100))
		if currentPrice <= firstStopPrice {
			return []*contracts.ExitSignal{pm.createFirstStopSignal(pos, currentPrice)}, nil
		}
	}

	// 3. Stop Floor: TP1 이후, 손익분기점+0.6% 아래로 내려가면 전량 청산
	if pos.State >= contracts.PositionStateTP1Done && pos.StopFloorPrice > 0 {
		if currentPrice <= pos.StopFloorPrice {
			return []*contracts.ExitSignal{pm.createStopFloorSignal(pos, currentPrice)}, nil
		}
	}

	// 4. HWM Trailing Stop: TP3 이후, 최고가 대비 하락 시 잔량 청산
	if pos.State == contracts.PositionStateTP3Done && pos.TrailStopPrice > 0 {
		if currentPrice <= pos.TrailStopPrice {
			return []*contracts.ExitSignal{pm.createHWMTrailSignal(pos, currentPrice)}, nil
		}
	}

	// 5. Take Profit 체크 (상태에 따라)
	var signals []*contracts.ExitSignal

	switch pos.State {
	case contracts.PositionStateOpen:
		if currentPrice >= pos.TP1TriggerPrice {
			signals = append(signals, pm.createTP1Signal(pos, currentPrice))
		}

	case contracts.PositionStateTP1Done:
		if currentPrice >= pos.TP2TriggerPrice {
			signals = append(signals, pm.createTP2Signal(pos, currentPrice))
		}

	case contracts.PositionStateTP2Done:
		if currentPrice >= pos.TP3TriggerPrice {
			signals = append(signals, pm.createTP3Signal(pos, currentPrice))
		}
	}

	return signals, nil
}

// updateTrailStopPrice HWM 트레일링 스탑 가격 갱신
func (pm *PositionMonitor) updateTrailStopPrice(pos *contracts.MonitoredPosition) {
	atr := pos.ATRPercent / 100
	trailPct := clamp(atr*pm.config.TrailATRMultiplier, pm.config.TrailMinPercent/100, pm.config.TrailMaxPercent/100)
	newTrailStop := int64(float64(pos.HighestPrice) * (1 - trailPct))

	// 트레일링 스탑은 올라가기만 함
	if newTrailStop > pos.TrailStopPrice {
		pos.TrailStopPrice = newTrailStop
		pm.logger.WithFields(map[string]interface{}{
			"code":        pos.Code,
			"hwm":         pos.HighestPrice,
			"trail_stop":  pos.TrailStopPrice,
			"trail_pct":   trailPct * 100,
		}).Debug("HWM Trail updated")
	}
}

// =============================================================================
// Signal Creators
// =============================================================================

func (pm *PositionMonitor) createTP1Signal(pos *contracts.MonitoredPosition, currentPrice int64) *contracts.ExitSignal {
	sellQty := int(float64(pos.InitialQuantity) * pm.config.TP1SellPercent / 100)
	if sellQty < 1 {
		sellQty = 1
	}
	if sellQty > pos.RemainingQuantity {
		sellQty = pos.RemainingQuantity
	}

	tp1Pct := float64(pos.TP1TriggerPrice-pos.EntryPrice) / float64(pos.EntryPrice) * 100

	return &contracts.ExitSignal{
		PositionID:   pos.ID,
		Code:         pos.Code,
		Name:         pos.Name,
		Reason:       contracts.ExitReasonTP1,
		CurrentPrice: currentPrice,
		EntryPrice:   pos.EntryPrice,
		PnLPercent:   pos.UnrealizedPnL,
		SellQuantity: sellQty,
		IsPartial:    pos.RemainingQuantity-sellQty > 0,
		Message:      fmt.Sprintf("TP1: +%.1f%% (%d -> %d), %d주 매도 (25%%)", tp1Pct, pos.EntryPrice, currentPrice, sellQty),
		TriggeredAt:  time.Now(),
	}
}

func (pm *PositionMonitor) createTP2Signal(pos *contracts.MonitoredPosition, currentPrice int64) *contracts.ExitSignal {
	sellQty := int(float64(pos.InitialQuantity) * pm.config.TP2SellPercent / 100)
	if sellQty < 1 {
		sellQty = 1
	}
	if sellQty > pos.RemainingQuantity {
		sellQty = pos.RemainingQuantity
	}

	tp2Pct := float64(pos.TP2TriggerPrice-pos.EntryPrice) / float64(pos.EntryPrice) * 100

	return &contracts.ExitSignal{
		PositionID:   pos.ID,
		Code:         pos.Code,
		Name:         pos.Name,
		Reason:       contracts.ExitReasonTP2,
		CurrentPrice: currentPrice,
		EntryPrice:   pos.EntryPrice,
		PnLPercent:   pos.UnrealizedPnL,
		SellQuantity: sellQty,
		IsPartial:    pos.RemainingQuantity-sellQty > 0,
		Message:      fmt.Sprintf("TP2: +%.1f%% (%d -> %d), %d주 매도 (25%%)", tp2Pct, pos.EntryPrice, currentPrice, sellQty),
		TriggeredAt:  time.Now(),
	}
}

func (pm *PositionMonitor) createTP3Signal(pos *contracts.MonitoredPosition, currentPrice int64) *contracts.ExitSignal {
	sellQty := int(float64(pos.InitialQuantity) * pm.config.TP3SellPercent / 100)
	if sellQty < 1 {
		sellQty = 1
	}
	if sellQty > pos.RemainingQuantity {
		sellQty = pos.RemainingQuantity
	}

	tp3Pct := float64(pos.TP3TriggerPrice-pos.EntryPrice) / float64(pos.EntryPrice) * 100

	return &contracts.ExitSignal{
		PositionID:   pos.ID,
		Code:         pos.Code,
		Name:         pos.Name,
		Reason:       contracts.ExitReasonTP3,
		CurrentPrice: currentPrice,
		EntryPrice:   pos.EntryPrice,
		PnLPercent:   pos.UnrealizedPnL,
		SellQuantity: sellQty,
		IsPartial:    pos.RemainingQuantity-sellQty > 0,
		Message:      fmt.Sprintf("TP3: +%.1f%% (%d -> %d), %d주 매도 (20%%), HWM Trailing 시작", tp3Pct, pos.EntryPrice, currentPrice, sellQty),
		TriggeredAt:  time.Now(),
	}
}

func (pm *PositionMonitor) createFirstStopSignal(pos *contracts.MonitoredPosition, currentPrice int64) *contracts.ExitSignal {
	sellQty := int(float64(pos.RemainingQuantity) * pm.config.FirstStopSellPercent / 100)
	if sellQty < 1 {
		sellQty = 1
	}
	if sellQty > pos.RemainingQuantity {
		sellQty = pos.RemainingQuantity
	}

	return &contracts.ExitSignal{
		PositionID:   pos.ID,
		Code:         pos.Code,
		Name:         pos.Name,
		Reason:       contracts.ExitReasonFirstStop,
		CurrentPrice: currentPrice,
		EntryPrice:   pos.EntryPrice,
		PnLPercent:   pos.UnrealizedPnL,
		SellQuantity: sellQty,
		IsPartial:    pos.RemainingQuantity-sellQty > 0,
		Message:      fmt.Sprintf("1차 손절: %.1f%% (%d -> %d), %d주 매도 (50%%)", pm.config.FirstStopPercent, pos.EntryPrice, currentPrice, sellQty),
		TriggeredAt:  time.Now(),
	}
}

func (pm *PositionMonitor) createSecondStopSignal(pos *contracts.MonitoredPosition, currentPrice int64) *contracts.ExitSignal {
	return &contracts.ExitSignal{
		PositionID:   pos.ID,
		Code:         pos.Code,
		Name:         pos.Name,
		Reason:       contracts.ExitReasonSecondStop,
		CurrentPrice: currentPrice,
		EntryPrice:   pos.EntryPrice,
		PnLPercent:   pos.UnrealizedPnL,
		SellQuantity: pos.RemainingQuantity,
		IsPartial:    false,
		Message:      fmt.Sprintf("2차 손절: %.1f%% (%d -> %d), 잔량 %d주 전량 매도", pm.config.SecondStopPercent, pos.EntryPrice, currentPrice, pos.RemainingQuantity),
		TriggeredAt:  time.Now(),
	}
}

func (pm *PositionMonitor) createStopFloorSignal(pos *contracts.MonitoredPosition, currentPrice int64) *contracts.ExitSignal {
	return &contracts.ExitSignal{
		PositionID:   pos.ID,
		Code:         pos.Code,
		Name:         pos.Name,
		Reason:       contracts.ExitReasonStopFloor,
		CurrentPrice: currentPrice,
		EntryPrice:   pos.EntryPrice,
		PnLPercent:   pos.UnrealizedPnL,
		SellQuantity: pos.RemainingQuantity,
		IsPartial:    false,
		Message:      fmt.Sprintf("Stop Floor: 바닥가 %d원 이탈 (현재 %d원), 잔량 %d주 매도", pos.StopFloorPrice, currentPrice, pos.RemainingQuantity),
		TriggeredAt:  time.Now(),
	}
}

func (pm *PositionMonitor) createHWMTrailSignal(pos *contracts.MonitoredPosition, currentPrice int64) *contracts.ExitSignal {
	return &contracts.ExitSignal{
		PositionID:   pos.ID,
		Code:         pos.Code,
		Name:         pos.Name,
		Reason:       contracts.ExitReasonHWMTrail,
		CurrentPrice: currentPrice,
		EntryPrice:   pos.EntryPrice,
		PnLPercent:   pos.UnrealizedPnL,
		SellQuantity: pos.RemainingQuantity,
		IsPartial:    false,
		Message:      fmt.Sprintf("HWM Trail: 최고가 %d -> 트레일 %d원 이탈 (현재 %d원), 잔량 %d주 매도", pos.HighestPrice, pos.TrailStopPrice, currentPrice, pos.RemainingQuantity),
		TriggeredAt:  time.Now(),
	}
}

// =============================================================================
// Batch Operations
// =============================================================================

// CheckAllPositions 모든 포지션 체크
func (pm *PositionMonitor) CheckAllPositions(ctx context.Context) ([]*contracts.ExitSignal, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var allSignals []*contracts.ExitSignal

	for _, pos := range pm.positions {
		signals, err := pm.CheckPosition(ctx, pos)
		if err != nil {
			pm.logger.WithFields(map[string]interface{}{
				"code":  pos.Code,
				"error": err.Error(),
			}).Warn("Error checking position")
			continue
		}

		for _, signal := range signals {
			allSignals = append(allSignals, signal)
			pm.logger.WithFields(map[string]interface{}{
				"code":     signal.Code,
				"reason":   signal.Reason,
				"quantity": signal.SellQuantity,
				"pnl":      signal.PnLPercent,
				"state":    pos.State,
			}).Info("Exit signal generated")
		}
	}

	return allSignals, nil
}

// =============================================================================
// Background Monitoring
// =============================================================================

// Start 백그라운드 모니터링 시작
func (pm *PositionMonitor) Start(ctx context.Context) error {
	if pm.isRunning {
		return fmt.Errorf("monitor already running")
	}

	pm.isRunning = true
	interval := time.Duration(pm.config.CheckIntervalSeconds) * time.Second

	pm.logger.WithFields(map[string]interface{}{
		"interval":     interval,
		"use_atr":      pm.config.UseATRBased,
		"tp1_range":    fmt.Sprintf("%.0f%%-%.0f%%", pm.config.TP1MinPercent, pm.config.TP1MaxPercent),
		"tp2_range":    fmt.Sprintf("%.0f%%-%.0f%%", pm.config.TP2MinPercent, pm.config.TP2MaxPercent),
		"tp3_range":    fmt.Sprintf("%.0f%%-%.0f%%", pm.config.TP3MinPercent, pm.config.TP3MaxPercent),
		"first_stop":   pm.config.FirstStopPercent,
		"second_stop":  pm.config.SecondStopPercent,
		"stop_floor":   pm.config.StopFloorBuffer,
		"trail_range":  fmt.Sprintf("%.0f%%-%.0f%%", pm.config.TrailMinPercent, pm.config.TrailMaxPercent),
	}).Info("Starting position monitor")

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				pm.isRunning = false
				pm.logger.Info("Position monitor stopped: context cancelled")
				return
			case <-pm.stopCh:
				pm.isRunning = false
				pm.logger.Info("Position monitor stopped")
				return
			case <-ticker.C:
				signals, err := pm.CheckAllPositions(ctx)
				if err != nil {
					pm.logger.WithFields(map[string]interface{}{
						"error": err.Error(),
					}).Error("Error checking positions")
					continue
				}

				for _, signal := range signals {
					pm.executeExit(ctx, signal)
				}
			}
		}
	}()

	return nil
}

// Stop 모니터링 중지
func (pm *PositionMonitor) Stop() {
	if pm.isRunning {
		close(pm.stopCh)
	}
}

// IsRunning 실행 상태 확인
func (pm *PositionMonitor) IsRunning() bool {
	return pm.isRunning
}

// executeExit 청산 실행
func (pm *PositionMonitor) executeExit(ctx context.Context, signal *contracts.ExitSignal) {
	pm.logger.WithFields(map[string]interface{}{
		"code":     signal.Code,
		"reason":   signal.Reason,
		"price":    signal.CurrentPrice,
		"quantity": signal.SellQuantity,
		"pnl":      signal.PnLPercent,
	}).Info("Executing exit")

	// 중복 실행 방지
	pm.mu.RLock()
	now := time.Now()
	for i := len(pm.recentSignals) - 1; i >= 0; i-- {
		recent := pm.recentSignals[i]
		if recent.Code == signal.Code && recent.Reason == signal.Reason {
			timeDiff := now.Sub(recent.TriggeredAt).Seconds()
			if timeDiff < 60 {
				pm.mu.RUnlock()
				pm.logger.WithFields(map[string]interface{}{
					"code":   signal.Code,
					"reason": signal.Reason,
				}).Warn("Duplicate signal detected, skipping")
				return
			}
		}
	}
	pm.mu.RUnlock()

	// 신호 저장
	pm.addSignal(signal)

	// 알림 발송
	if pm.notifier != nil {
		if err := pm.notifier.NotifyExitSignal(ctx, signal); err != nil {
			pm.logger.WithFields(map[string]interface{}{
				"code":  signal.Code,
				"error": err.Error(),
			}).Error("Failed to send notification")
		}
	}

	// 포지션 상태 업데이트
	pm.updatePositionState(signal)
}

// addSignal 청산 신호 추가
func (pm *PositionMonitor) addSignal(signal *contracts.ExitSignal) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.recentSignals) >= 50 {
		pm.recentSignals = pm.recentSignals[1:]
	}
	pm.recentSignals = append(pm.recentSignals, signal)
}

// updatePositionState 포지션 상태 업데이트
func (pm *PositionMonitor) updatePositionState(signal *contracts.ExitSignal) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pos, ok := pm.positions[signal.Code]
	if !ok {
		return
	}

	switch signal.Reason {
	case contracts.ExitReasonTP1:
		pos.State = contracts.PositionStateTP1Done
		pos.TakeProfitCount = 1
		pos.TP1Done = true
		pos.StopFloorPrice = int64(float64(pos.EntryPrice) * (1 + pm.config.StopFloorBuffer/100))
		pm.logger.WithFields(map[string]interface{}{
			"code":        signal.Code,
			"state":       pos.State,
			"stop_floor":  pos.StopFloorPrice,
		}).Info("TP1 done, StopFloor activated")

	case contracts.ExitReasonTP2:
		pos.State = contracts.PositionStateTP2Done
		pos.TakeProfitCount = 2
		pos.TP2Done = true

	case contracts.ExitReasonTP3:
		pos.State = contracts.PositionStateTP3Done
		pos.TakeProfitCount = 3
		pos.TP3Done = true
		pm.updateTrailStopPrice(pos)
		pm.logger.WithFields(map[string]interface{}{
			"code":       signal.Code,
			"state":      pos.State,
			"trail_stop": pos.TrailStopPrice,
		}).Info("TP3 done, HWM Trailing activated")

	case contracts.ExitReasonFirstStop:
		pos.FirstStopTriggered = true

	case contracts.ExitReasonSecondStop, contracts.ExitReasonStopFloor, contracts.ExitReasonHWMTrail:
		pos.State = contracts.PositionStateClosed
	}

	// 잔량 업데이트
	if signal.IsPartial {
		pos.RemainingQuantity -= signal.SellQuantity
		if pos.RemainingQuantity <= 0 {
			delete(pm.positions, signal.Code)
		}
	} else {
		delete(pm.positions, signal.Code)
	}
}

// GetConfig 현재 설정 반환
func (pm *PositionMonitor) GetConfig() *contracts.ExitRulesConfig {
	return pm.config
}

// UpdateConfig 설정 업데이트
func (pm *PositionMonitor) UpdateConfig(config *contracts.ExitRulesConfig) {
	pm.config = config
	pm.logger.Info("Exit rules config updated")
}
