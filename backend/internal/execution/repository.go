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
			order_id, code, name, side, quantity, price, order_type, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (order_id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.pool.Exec(ctx, query,
		order.ID, order.Code, order.Name, order.Side, order.Qty, order.Price,
		order.OrderType, order.Status, order.CreatedAt, order.UpdatedAt,
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
		SELECT order_id, code, name, side, quantity, price, order_type, status, created_at, updated_at
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
		SELECT order_id, code, name, side, quantity, price, order_type, status, created_at, updated_at
		FROM execution.orders
		WHERE DATE(created_at) = $1
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
