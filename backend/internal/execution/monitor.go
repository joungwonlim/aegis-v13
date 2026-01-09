package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Monitor monitors order execution status
// ⭐ SSOT: 체결 모니터링 로직은 여기서만
type Monitor struct {
	broker     Broker
	repository *Repository
	logger     *logger.Logger
}

// MonitorConfig defines monitoring parameters
type MonitorConfig struct {
	PollInterval  time.Duration // 상태 조회 주기
	MaxRetries    int           // 최대 재시도 횟수
	RetryDelay    time.Duration // 재시도 대기 시간
	TimeoutMinutes int           // 타임아웃 (분)
}

// NewMonitor creates a new execution monitor
func NewMonitor(broker Broker, repository *Repository, logger *logger.Logger) *Monitor {
	return &Monitor{
		broker:     broker,
		repository: repository,
		logger:     logger,
	}
}

// MonitorOrders monitors multiple orders until completion
func (m *Monitor) MonitorOrders(ctx context.Context, orders []contracts.Order, config MonitorConfig) ([]ExecutionResult, error) {
	results := make([]ExecutionResult, 0, len(orders))
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	timeout := time.After(time.Duration(config.TimeoutMinutes) * time.Minute)
	pending := make(map[string]contracts.Order)

	// 초기화: 모든 주문을 pending 맵에 추가
	for _, order := range orders {
		pending[order.ID] = order
	}

	for {
		select {
		case <-ctx.Done():
			return results, ctx.Err()

		case <-timeout:
			m.logger.Warn("Order monitoring timeout")
			return results, fmt.Errorf("monitoring timeout after %d minutes", config.TimeoutMinutes)

		case <-ticker.C:
			// 모든 pending 주문 상태 확인
			for orderID, order := range pending {
				status, err := m.broker.GetOrderStatus(ctx, orderID)
				if err != nil {
					m.logger.WithFields(map[string]interface{}{
						"order_id": orderID,
						"error":    err,
					}).Warn("Failed to get order status")
					continue
				}

				// 상태 업데이트
				if status.Status == contracts.StatusFilled || status.Status == contracts.StatusRejected || status.Status == contracts.StatusCanceled {
					result := ExecutionResult{
						Order:       order,
						Status:      status.Status,
						FilledQty:   status.FilledQty,
						FilledPrice: status.FilledPrice,
						UpdatedAt:   time.Now(),
					}
					results = append(results, result)

					// DB 저장
					if err := m.repository.SaveExecutionResult(ctx, &result); err != nil {
						m.logger.WithFields(map[string]interface{}{
							"order_id": orderID,
							"error":    err,
						}).Warn("Failed to save execution result")
					}

					// pending에서 제거
					delete(pending, orderID)

					m.logger.WithFields(map[string]interface{}{
						"order_id":     orderID,
						"status":       status.Status,
						"filled_qty":   status.FilledQty,
						"filled_price": status.FilledPrice,
					}).Info("Order completed")
				}
			}

			// 모든 주문 완료 시 종료
			if len(pending) == 0 {
				m.logger.WithFields(map[string]interface{}{
					"total_orders": len(results),
					"filled":       m.countByStatus(results, contracts.StatusFilled),
					"rejected":     m.countByStatus(results, contracts.StatusRejected),
				}).Info("All orders completed")
				return results, nil
			}
		}
	}
}

// ExecutionResult represents execution result
type ExecutionResult struct {
	Order       contracts.Order
	Status      contracts.Status
	FilledQty   int
	FilledPrice float64
	UpdatedAt   time.Time
}

// countByStatus counts results by status
func (m *Monitor) countByStatus(results []ExecutionResult, status contracts.Status) int {
	count := 0
	for _, r := range results {
		if r.Status == status {
			count++
		}
	}
	return count
}

// DefaultMonitorConfig returns default monitoring configuration
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		PollInterval:   5 * time.Second,  // 5초마다 조회
		MaxRetries:     3,                // 최대 3회 재시도
		RetryDelay:     5 * time.Second,  // 5초 대기
		TimeoutMinutes: 30,               // 30분 타임아웃
	}
}
