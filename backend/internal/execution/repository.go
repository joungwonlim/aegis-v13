package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Repository handles execution data persistence
// ⭐ SSOT: Execution 데이터 저장/조회는 여기서만
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new execution repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveOrder saves an order to database
func (r *Repository) SaveOrder(ctx context.Context, order *contracts.Order) error {
	query := `
		INSERT INTO execution.orders (
			order_id, stock_code, stock_name, order_date, order_action, order_type,
			order_price, order_qty, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (order_id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.pool.Exec(ctx, query,
		order.ID, order.Code, order.Name, order.CreatedAt.Truncate(24*time.Hour),
		order.Side, order.OrderType, order.Price, order.Qty,
		order.Status, order.CreatedAt, order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}

	return nil
}

// UpdateOrderStatus updates order status
func (r *Repository) UpdateOrderStatus(ctx context.Context, orderID string, status contracts.Status) error {
	query := `
		UPDATE execution.orders
		SET status = $1, updated_at = $2
		WHERE order_id = $3
	`

	_, err := r.pool.Exec(ctx, query, status, time.Now(), orderID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// GetOrder retrieves an order by ID
func (r *Repository) GetOrder(ctx context.Context, orderID string) (*contracts.Order, error) {
	query := `
		SELECT order_id, stock_code, stock_name, order_action, order_qty, order_price,
		       order_type, status, created_at, updated_at
		FROM execution.orders
		WHERE order_id = $1
	`

	var order contracts.Order
	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&order.ID, &order.Code, &order.Name, &order.Side, &order.Qty, &order.Price,
		&order.OrderType, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

// GetOrdersByDate retrieves orders for a specific date
func (r *Repository) GetOrdersByDate(ctx context.Context, date time.Time) ([]contracts.Order, error) {
	query := `
		SELECT order_id, stock_code, stock_name, order_action, order_qty, order_price,
		       order_type, status, created_at, updated_at
		FROM execution.orders
		WHERE order_date = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	orders := make([]contracts.Order, 0)

	for rows.Next() {
		var order contracts.Order
		err := rows.Scan(
			&order.ID, &order.Code, &order.Name, &order.Side, &order.Qty, &order.Price,
			&order.OrderType, &order.Status, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return orders, nil
}

// SaveExecution saves an execution record
func (r *Repository) SaveExecution(ctx context.Context, exec *Execution) error {
	query := `
		INSERT INTO execution.executions (
			order_id, exec_qty, exec_price, exec_time, fee
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		exec.OrderID, exec.ExecQty, exec.ExecPrice, exec.ExecTime, exec.Fee,
	)

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// GetExecutionsByOrderID retrieves executions for an order
func (r *Repository) GetExecutionsByOrderID(ctx context.Context, orderID string) ([]Execution, error) {
	query := `
		SELECT id, order_id, exec_qty, exec_price, exec_time, fee, created_at
		FROM execution.executions
		WHERE order_id = $1
		ORDER BY exec_time ASC
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer rows.Close()

	executions := make([]Execution, 0)

	for rows.Next() {
		var exec Execution
		err := rows.Scan(
			&exec.ID, &exec.OrderID, &exec.ExecQty, &exec.ExecPrice,
			&exec.ExecTime, &exec.Fee, &exec.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return executions, nil
}

// SaveExecutionResult saves execution result
func (r *Repository) SaveExecutionResult(ctx context.Context, result *ExecutionResult) error {
	// Update order status
	if err := r.UpdateOrderStatus(ctx, result.Order.ID, result.Status); err != nil {
		return err
	}

	// Save execution record if filled
	if result.Status == contracts.StatusFilled && result.FilledQty > 0 {
		exec := &Execution{
			OrderID:   result.Order.ID,
			ExecQty:   result.FilledQty,
			ExecPrice: result.FilledPrice,
			ExecTime:  result.UpdatedAt,
			Fee:       0, // TODO: Calculate fee
		}
		if err := r.SaveExecution(ctx, exec); err != nil {
			return err
		}
	}

	return nil
}

// Execution represents execution record
type Execution struct {
	ID        int64
	OrderID   string
	ExecQty   int
	ExecPrice float64
	ExecTime  time.Time
	Fee       float64
	CreatedAt time.Time
}

// GetExecutionSummary retrieves execution summary for a date
func (r *Repository) GetExecutionSummary(ctx context.Context, date time.Time) (*ExecutionSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_orders,
			COUNT(CASE WHEN status = 'FILLED' THEN 1 END) as filled_orders,
			COUNT(CASE WHEN status = 'REJECTED' THEN 1 END) as rejected_orders,
			COUNT(CASE WHEN status = 'CANCELED' THEN 1 END) as canceled_orders
		FROM execution.orders
		WHERE DATE(created_at) = $1
	`

	var summary ExecutionSummary
	err := r.pool.QueryRow(ctx, query, date).Scan(
		&summary.TotalOrders, &summary.FilledOrders, &summary.RejectedOrders, &summary.CanceledOrders,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get execution summary: %w", err)
	}

	summary.Date = date
	return &summary, nil
}

// ExecutionSummary represents execution summary statistics
type ExecutionSummary struct {
	Date           time.Time
	TotalOrders    int
	FilledOrders   int
	RejectedOrders int
	CanceledOrders int
}

// =============================================================================
// Risk Gate Events
// =============================================================================

// SaveGateEvent 리스크 게이트 이벤트 저장
func (r *Repository) SaveGateEvent(ctx context.Context, event *GateEvent) error {
	query := `
		INSERT INTO execution.risk_gate_events (
			run_id, mode, passed, would_block,
			violation_count, var_95, var_99, message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		event.RunID, event.Mode, event.Passed, event.WouldBlock,
		event.ViolationCount, event.VaR95, event.VaR99, event.Message, event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save gate event: %w", err)
	}

	return nil
}

// GetGateEventsByDate 특정 날짜의 게이트 이벤트 조회
func (r *Repository) GetGateEventsByDate(ctx context.Context, date time.Time) ([]GateEvent, error) {
	query := `
		SELECT id, run_id, mode, passed, would_block,
		       violation_count, var_95, var_99, message, created_at
		FROM execution.risk_gate_events
		WHERE DATE(created_at) = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query gate events: %w", err)
	}
	defer rows.Close()

	events := make([]GateEvent, 0)
	for rows.Next() {
		var event GateEvent
		var mode string
		err := rows.Scan(
			&event.ID, &event.RunID, &mode, &event.Passed, &event.WouldBlock,
			&event.ViolationCount, &event.VaR95, &event.VaR99, &event.Message, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan gate event: %w", err)
		}
		event.Mode = GateMode(mode)
		events = append(events, event)
	}

	return events, nil
}

// GetShadowBlockStats Shadow 모드 차단 통계 조회
func (r *Repository) GetShadowBlockStats(ctx context.Context, from, to time.Time) (*ShadowBlockStats, error) {
	query := `
		SELECT
			COUNT(*) as total_checks,
			COUNT(CASE WHEN would_block = true THEN 1 END) as would_block_count,
			AVG(var_95) as avg_var_95,
			MAX(var_95) as max_var_95
		FROM execution.risk_gate_events
		WHERE mode = 'shadow'
		  AND created_at >= $1
		  AND created_at < $2
	`

	var stats ShadowBlockStats
	err := r.pool.QueryRow(ctx, query, from, to).Scan(
		&stats.TotalChecks, &stats.WouldBlockCount, &stats.AvgVaR95, &stats.MaxVaR95,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get shadow block stats: %w", err)
	}

	stats.From = from
	stats.To = to
	if stats.TotalChecks > 0 {
		stats.BlockRate = float64(stats.WouldBlockCount) / float64(stats.TotalChecks)
	}

	return &stats, nil
}

// ShadowBlockStats Shadow 모드 차단 통계
type ShadowBlockStats struct {
	From            time.Time
	To              time.Time
	TotalChecks     int
	WouldBlockCount int
	BlockRate       float64
	AvgVaR95        float64
	MaxVaR95        float64
}
