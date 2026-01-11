package forecast

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
)

// Repository forecast 데이터 저장소
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 새 저장소 생성
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveEvent 이벤트 저장
func (r *Repository) SaveEvent(ctx context.Context, event contracts.ForecastEvent) (int64, error) {
	query := `
		INSERT INTO analytics.forecast_events
			(code, event_date, event_type, day_return, close_to_high, gap_ratio, volume_z_score, sector, market_cap_bucket)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (code, event_date, event_type)
		DO UPDATE SET
			day_return = EXCLUDED.day_return,
			close_to_high = EXCLUDED.close_to_high,
			gap_ratio = EXCLUDED.gap_ratio,
			volume_z_score = EXCLUDED.volume_z_score,
			sector = EXCLUDED.sector,
			market_cap_bucket = EXCLUDED.market_cap_bucket
		RETURNING id`

	var id int64
	err := r.pool.QueryRow(ctx, query,
		event.Code, event.Date, event.EventType,
		event.DayReturn, event.CloseToHigh, event.GapRatio,
		event.VolumeZScore, event.Sector, event.MarketCapBucket,
	).Scan(&id)

	return id, err
}

// SaveEvents 이벤트 일괄 저장
func (r *Repository) SaveEvents(ctx context.Context, events []contracts.ForecastEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO analytics.forecast_events
			(code, event_date, event_type, day_return, close_to_high, gap_ratio, volume_z_score, sector, market_cap_bucket)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (code, event_date, event_type) DO UPDATE SET
			day_return = EXCLUDED.day_return,
			close_to_high = EXCLUDED.close_to_high,
			gap_ratio = EXCLUDED.gap_ratio,
			volume_z_score = EXCLUDED.volume_z_score,
			sector = EXCLUDED.sector,
			market_cap_bucket = EXCLUDED.market_cap_bucket`

	for _, e := range events {
		batch.Queue(query, e.Code, e.Date, e.EventType,
			e.DayReturn, e.CloseToHigh, e.GapRatio,
			e.VolumeZScore, e.Sector, e.MarketCapBucket)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range events {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

// GetEvent 이벤트 조회
func (r *Repository) GetEvent(ctx context.Context, id int64) (*contracts.ForecastEvent, error) {
	query := `
		SELECT id, code, event_date, event_type, day_return, close_to_high, gap_ratio,
			   volume_z_score, sector, market_cap_bucket, created_at
		FROM analytics.forecast_events
		WHERE id = $1`

	var e contracts.ForecastEvent
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.Code, &e.Date, &e.EventType, &e.DayReturn, &e.CloseToHigh,
		&e.GapRatio, &e.VolumeZScore, &e.Sector, &e.MarketCapBucket, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// GetEventsByDateRange 날짜 범위로 이벤트 조회
func (r *Repository) GetEventsByDateRange(ctx context.Context, from, to time.Time) ([]contracts.ForecastEvent, error) {
	query := `
		SELECT id, code, event_date, event_type, day_return, close_to_high, gap_ratio,
			   volume_z_score, sector, market_cap_bucket, created_at
		FROM analytics.forecast_events
		WHERE event_date BETWEEN $1 AND $2
		ORDER BY event_date DESC, code`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []contracts.ForecastEvent
	for rows.Next() {
		var e contracts.ForecastEvent
		if err := rows.Scan(
			&e.ID, &e.Code, &e.Date, &e.EventType, &e.DayReturn, &e.CloseToHigh,
			&e.GapRatio, &e.VolumeZScore, &e.Sector, &e.MarketCapBucket, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// GetEventsByCode 종목 코드로 이벤트 조회
func (r *Repository) GetEventsByCode(ctx context.Context, code string) ([]contracts.ForecastEvent, error) {
	query := `
		SELECT id, code, event_date, event_type, day_return, close_to_high, gap_ratio,
			   volume_z_score, sector, market_cap_bucket, created_at
		FROM analytics.forecast_events
		WHERE code = $1
		ORDER BY event_date DESC`

	rows, err := r.pool.Query(ctx, query, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []contracts.ForecastEvent
	for rows.Next() {
		var e contracts.ForecastEvent
		if err := rows.Scan(
			&e.ID, &e.Code, &e.Date, &e.EventType, &e.DayReturn, &e.CloseToHigh,
			&e.GapRatio, &e.VolumeZScore, &e.Sector, &e.MarketCapBucket, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// GetEventsByCodeWithPerformance 종목 코드로 이벤트와 전방 성과 함께 조회
func (r *Repository) GetEventsByCodeWithPerformance(ctx context.Context, code string) ([]contracts.EventWithPerformance, error) {
	query := `
		SELECT
			e.id, e.code, e.event_date, e.event_type, e.day_return, e.close_to_high,
			e.gap_ratio, e.volume_z_score, e.created_at,
			f.fwd_ret_1d, f.fwd_ret_2d, f.fwd_ret_3d, f.fwd_ret_5d,
			f.max_runup_5d, f.max_drawdown_5d, f.gap_hold_3d
		FROM analytics.forecast_events e
		LEFT JOIN analytics.forward_performance f ON e.id = f.event_id
		WHERE e.code = $1
		ORDER BY e.event_date DESC`

	rows, err := r.pool.Query(ctx, query, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []contracts.EventWithPerformance
	for rows.Next() {
		var e contracts.EventWithPerformance
		var id int64
		var code string
		var eventDate time.Time
		var eventType contracts.ForecastEventType
		var dayReturn, closeToHigh, gapRatio, volZ float64
		var createdAt time.Time
		var fwdRet1D, fwdRet2D, fwdRet3D, fwdRet5D, maxRunup5D, maxDrawdown5D *float64
		var gapHold3D *bool

		if err := rows.Scan(
			&id, &code, &eventDate, &eventType, &dayReturn, &closeToHigh,
			&gapRatio, &volZ, &createdAt,
			&fwdRet1D, &fwdRet2D, &fwdRet3D, &fwdRet5D,
			&maxRunup5D, &maxDrawdown5D, &gapHold3D,
		); err != nil {
			return nil, err
		}

		e.ID = id
		e.Symbol = code
		e.TradeDate = eventDate.Format("2006-01-02")
		e.EventType = string(eventType)
		e.Ret = dayReturn
		e.Gap = gapRatio
		e.CloseToHigh = closeToHigh
		e.VolZ = volZ
		e.FwdRet1D = fwdRet1D
		e.FwdRet2D = fwdRet2D
		e.FwdRet3D = fwdRet3D
		e.FwdRet5D = fwdRet5D
		e.MaxRunup5D = maxRunup5D
		e.MaxDrawdown5D = maxDrawdown5D
		e.GapHold3D = gapHold3D
		e.CreatedAt = createdAt.Format(time.RFC3339)
		e.UpdatedAt = createdAt.Format(time.RFC3339)

		events = append(events, e)
	}

	return events, nil
}

// GetEventsWithoutForward 전방 성과가 없는 이벤트 조회
func (r *Repository) GetEventsWithoutForward(ctx context.Context) ([]contracts.ForecastEvent, error) {
	query := `
		SELECT e.id, e.code, e.event_date, e.event_type, e.day_return, e.close_to_high,
			   e.gap_ratio, e.volume_z_score, e.sector, e.market_cap_bucket, e.created_at
		FROM analytics.forecast_events e
		LEFT JOIN analytics.forward_performance f ON e.id = f.event_id
		WHERE f.id IS NULL
		  AND e.event_date < CURRENT_DATE - INTERVAL '5 days'
		ORDER BY e.event_date`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []contracts.ForecastEvent
	for rows.Next() {
		var e contracts.ForecastEvent
		if err := rows.Scan(
			&e.ID, &e.Code, &e.Date, &e.EventType, &e.DayReturn, &e.CloseToHigh,
			&e.GapRatio, &e.VolumeZScore, &e.Sector, &e.MarketCapBucket, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// SaveForwardPerformance 전방 성과 저장
func (r *Repository) SaveForwardPerformance(ctx context.Context, perf contracts.ForwardPerformance) error {
	query := `
		INSERT INTO analytics.forward_performance
			(event_id, fwd_ret_1d, fwd_ret_2d, fwd_ret_3d, fwd_ret_5d,
			 max_runup_5d, max_drawdown_5d, gap_hold_3d)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (event_id) DO UPDATE SET
			fwd_ret_1d = EXCLUDED.fwd_ret_1d,
			fwd_ret_2d = EXCLUDED.fwd_ret_2d,
			fwd_ret_3d = EXCLUDED.fwd_ret_3d,
			fwd_ret_5d = EXCLUDED.fwd_ret_5d,
			max_runup_5d = EXCLUDED.max_runup_5d,
			max_drawdown_5d = EXCLUDED.max_drawdown_5d,
			gap_hold_3d = EXCLUDED.gap_hold_3d`

	_, err := r.pool.Exec(ctx, query,
		perf.EventID, perf.FwdRet1D, perf.FwdRet2D, perf.FwdRet3D, perf.FwdRet5D,
		perf.MaxRunup5D, perf.MaxDrawdown5D, perf.GapHold3D,
	)
	return err
}

// SaveForwardPerformances 전방 성과 일괄 저장
func (r *Repository) SaveForwardPerformances(ctx context.Context, perfs []contracts.ForwardPerformance) error {
	if len(perfs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO analytics.forward_performance
			(event_id, fwd_ret_1d, fwd_ret_2d, fwd_ret_3d, fwd_ret_5d,
			 max_runup_5d, max_drawdown_5d, gap_hold_3d)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (event_id) DO UPDATE SET
			fwd_ret_1d = EXCLUDED.fwd_ret_1d,
			fwd_ret_2d = EXCLUDED.fwd_ret_2d,
			fwd_ret_3d = EXCLUDED.fwd_ret_3d,
			fwd_ret_5d = EXCLUDED.fwd_ret_5d,
			max_runup_5d = EXCLUDED.max_runup_5d,
			max_drawdown_5d = EXCLUDED.max_drawdown_5d,
			gap_hold_3d = EXCLUDED.gap_hold_3d`

	for _, p := range perfs {
		batch.Queue(query, p.EventID, p.FwdRet1D, p.FwdRet2D, p.FwdRet3D, p.FwdRet5D,
			p.MaxRunup5D, p.MaxDrawdown5D, p.GapHold3D)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range perfs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

// GetEventsWithPerformance 이벤트와 전방 성과 조인 조회
func (r *Repository) GetEventsWithPerformance(ctx context.Context) ([]EventWithPerformance, error) {
	query := `
		SELECT e.id, e.code, e.event_date, e.event_type, e.day_return, e.close_to_high,
			   e.gap_ratio, e.volume_z_score, e.sector, e.market_cap_bucket, e.created_at,
			   f.id, f.event_id, f.fwd_ret_1d, f.fwd_ret_2d, f.fwd_ret_3d, f.fwd_ret_5d,
			   f.max_runup_5d, f.max_drawdown_5d, f.gap_hold_3d, f.filled_at
		FROM analytics.forecast_events e
		INNER JOIN analytics.forward_performance f ON e.id = f.event_id
		ORDER BY e.event_date DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []EventWithPerformance
	for rows.Next() {
		var ewp EventWithPerformance
		if err := rows.Scan(
			&ewp.Event.ID, &ewp.Event.Code, &ewp.Event.Date, &ewp.Event.EventType,
			&ewp.Event.DayReturn, &ewp.Event.CloseToHigh, &ewp.Event.GapRatio,
			&ewp.Event.VolumeZScore, &ewp.Event.Sector, &ewp.Event.MarketCapBucket, &ewp.Event.CreatedAt,
			&ewp.Performance.ID, &ewp.Performance.EventID, &ewp.Performance.FwdRet1D,
			&ewp.Performance.FwdRet2D, &ewp.Performance.FwdRet3D, &ewp.Performance.FwdRet5D,
			&ewp.Performance.MaxRunup5D, &ewp.Performance.MaxDrawdown5D, &ewp.Performance.GapHold3D,
			&ewp.Performance.FilledAt,
		); err != nil {
			return nil, err
		}
		results = append(results, ewp)
	}

	return results, nil
}

// SaveStats 통계 저장
func (r *Repository) SaveStats(ctx context.Context, stats contracts.ForecastStats) error {
	query := `
		INSERT INTO analytics.forecast_stats
			(level, key, event_type, sample_count, avg_ret_1d, avg_ret_2d, avg_ret_3d, avg_ret_5d,
			 win_rate_1d, win_rate_5d, p10_mdd)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (level, key, event_type) DO UPDATE SET
			sample_count = EXCLUDED.sample_count,
			avg_ret_1d = EXCLUDED.avg_ret_1d,
			avg_ret_2d = EXCLUDED.avg_ret_2d,
			avg_ret_3d = EXCLUDED.avg_ret_3d,
			avg_ret_5d = EXCLUDED.avg_ret_5d,
			win_rate_1d = EXCLUDED.win_rate_1d,
			win_rate_5d = EXCLUDED.win_rate_5d,
			p10_mdd = EXCLUDED.p10_mdd,
			updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query,
		stats.Level, stats.Key, stats.EventType, stats.SampleCount,
		stats.AvgRet1D, stats.AvgRet2D, stats.AvgRet3D, stats.AvgRet5D,
		stats.WinRate1D, stats.WinRate5D, stats.P10MDD,
	)
	return err
}

// SaveAllStats 통계 일괄 저장
func (r *Repository) SaveAllStats(ctx context.Context, statsList []contracts.ForecastStats) error {
	if len(statsList) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO analytics.forecast_stats
			(level, key, event_type, sample_count, avg_ret_1d, avg_ret_2d, avg_ret_3d, avg_ret_5d,
			 win_rate_1d, win_rate_5d, p10_mdd)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (level, key, event_type) DO UPDATE SET
			sample_count = EXCLUDED.sample_count,
			avg_ret_1d = EXCLUDED.avg_ret_1d,
			avg_ret_2d = EXCLUDED.avg_ret_2d,
			avg_ret_3d = EXCLUDED.avg_ret_3d,
			avg_ret_5d = EXCLUDED.avg_ret_5d,
			win_rate_1d = EXCLUDED.win_rate_1d,
			win_rate_5d = EXCLUDED.win_rate_5d,
			p10_mdd = EXCLUDED.p10_mdd,
			updated_at = NOW()`

	for _, s := range statsList {
		batch.Queue(query, s.Level, s.Key, s.EventType, s.SampleCount,
			s.AvgRet1D, s.AvgRet2D, s.AvgRet3D, s.AvgRet5D,
			s.WinRate1D, s.WinRate5D, s.P10MDD)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range statsList {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

// GetStats 통계 조회 (StatsProvider 인터페이스 구현)
func (r *Repository) GetStats(ctx context.Context, level contracts.ForecastStatsLevel, key string, eventType contracts.ForecastEventType) (*contracts.ForecastStats, error) {
	query := `
		SELECT id, level, key, event_type, sample_count, avg_ret_1d, avg_ret_2d, avg_ret_3d, avg_ret_5d,
			   win_rate_1d, win_rate_5d, p10_mdd, updated_at
		FROM analytics.forecast_stats
		WHERE level = $1 AND key = $2 AND event_type = $3`

	var s contracts.ForecastStats
	err := r.pool.QueryRow(ctx, query, level, key, eventType).Scan(
		&s.ID, &s.Level, &s.Key, &s.EventType, &s.SampleCount,
		&s.AvgRet1D, &s.AvgRet2D, &s.AvgRet3D, &s.AvgRet5D,
		&s.WinRate1D, &s.WinRate5D, &s.P10MDD, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// GetRecentEvents 최근 N일 이벤트 조회
func (r *Repository) GetRecentEvents(ctx context.Context, days int) ([]contracts.ForecastEvent, error) {
	query := `
		SELECT id, code, event_date, event_type, day_return, close_to_high, gap_ratio,
			   volume_z_score, sector, market_cap_bucket, created_at
		FROM analytics.forecast_events
		WHERE event_date >= CURRENT_DATE - $1::interval
		ORDER BY event_date DESC, code`

	rows, err := r.pool.Query(ctx, query, time.Duration(days)*24*time.Hour)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []contracts.ForecastEvent
	for rows.Next() {
		var e contracts.ForecastEvent
		if err := rows.Scan(
			&e.ID, &e.Code, &e.Date, &e.EventType, &e.DayReturn, &e.CloseToHigh,
			&e.GapRatio, &e.VolumeZScore, &e.Sector, &e.MarketCapBucket, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}
