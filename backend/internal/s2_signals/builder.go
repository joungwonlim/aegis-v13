package s2_signals

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Builder orchestrates all signal calculators to generate SignalSet
// ⭐ SSOT: 시그널 생성 오케스트레이션은 여기서만
type Builder struct {
	// Signal calculators
	momentum  *MomentumCalculator
	technical *TechnicalCalculator
	value     *ValueCalculator
	quality   *QualityCalculator
	flow      *FlowCalculator
	event     *EventCalculator

	// Data repositories
	priceRepo      contracts.PriceRepository
	flowRepo       contracts.InvestorFlowRepository
	financialRepo  contracts.FinancialRepository
	disclosureRepo contracts.DisclosureRepository

	logger *logger.Logger
}

// NewBuilder creates a new signal builder
func NewBuilder(
	momentum *MomentumCalculator,
	technical *TechnicalCalculator,
	value *ValueCalculator,
	quality *QualityCalculator,
	flow *FlowCalculator,
	event *EventCalculator,
	priceRepo contracts.PriceRepository,
	flowRepo contracts.InvestorFlowRepository,
	financialRepo contracts.FinancialRepository,
	disclosureRepo contracts.DisclosureRepository,
	logger *logger.Logger,
) *Builder {
	return &Builder{
		momentum:       momentum,
		technical:      technical,
		value:          value,
		quality:        quality,
		flow:           flow,
		event:          event,
		priceRepo:      priceRepo,
		flowRepo:       flowRepo,
		financialRepo:  financialRepo,
		disclosureRepo: disclosureRepo,
		logger:         logger,
	}
}

// Build generates SignalSet for all stocks in the universe
func (b *Builder) Build(ctx context.Context, universe *contracts.Universe, date time.Time) (*contracts.SignalSet, error) {
	b.logger.WithFields(map[string]interface{}{
		"date":        date.Format("2006-01-02"),
		"stock_count": len(universe.Stocks),
	}).Info("Starting signal generation")

	signalSet := &contracts.SignalSet{
		Date:    date,
		Signals: make(map[string]*contracts.StockSignals),
	}

	// Calculate signals for each stock
	successCount := 0
	for _, code := range universe.Stocks {
		signals, err := b.calculateStockSignals(ctx, code, date)
		if err != nil {
			b.logger.WithFields(map[string]interface{}{
				"code":  code,
				"error": err.Error(),
			}).Warn("Failed to calculate signals for stock")
			continue
		}

		signalSet.Signals[code] = signals
		successCount++
	}

	b.logger.WithFields(map[string]interface{}{
		"total":   len(universe.Stocks),
		"success": successCount,
		"failed":  len(universe.Stocks) - successCount,
	}).Info("Signal generation completed")

	return signalSet, nil
}

// calculateStockSignals calculates all signals for a single stock
func (b *Builder) calculateStockSignals(ctx context.Context, code string, date time.Time) (*contracts.StockSignals, error) {
	signals := &contracts.StockSignals{
		Code: code,
	}

	// 1. Fetch price data (for momentum and technical)
	prices, err := b.fetchPriceData(ctx, code, date)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price data: %w", err)
	}

	// 2. Calculate momentum signal
	if len(prices) >= 60 {
		score, details, err := b.momentum.Calculate(ctx, code, prices)
		if err == nil {
			signals.Momentum = score
			signals.Details.Return1M = details.Return1M
			signals.Details.Return3M = details.Return3M
			signals.Details.VolumeRate = details.VolumeRate
		}
	}

	// 3. Calculate technical signal
	if len(prices) >= 120 {
		score, details, err := b.technical.Calculate(ctx, code, prices)
		if err == nil {
			signals.Technical = score
			signals.Details.RSI = details.RSI
			signals.Details.MACD = details.MACD
			signals.Details.MA20Cross = details.MA20Cross
		}
	}

	// 4. Fetch and calculate value signal
	valueMetrics, err := b.fetchValueMetrics(ctx, code, date)
	if err == nil {
		score, details, err := b.value.Calculate(ctx, code, valueMetrics)
		if err == nil {
			signals.Value = score
			signals.Details.PER = details.PER
			signals.Details.PBR = details.PBR
			signals.Details.PSR = details.PSR
		}
	}

	// 5. Fetch and calculate quality signal
	qualityMetrics, err := b.fetchQualityMetrics(ctx, code, date)
	if err == nil {
		score, details, err := b.quality.Calculate(ctx, code, qualityMetrics)
		if err == nil {
			signals.Quality = score
			signals.Details.ROE = details.ROE
			signals.Details.DebtRatio = details.DebtRatio
		}
	}

	// 6. Fetch and calculate flow signal
	flowData, err := b.fetchFlowData(ctx, code, date)
	if err == nil && len(flowData) >= 20 {
		score, details, err := b.flow.Calculate(ctx, code, flowData)
		if err == nil {
			signals.Flow = score
			signals.Details.ForeignNet5D = details.ForeignNet5D
			signals.Details.ForeignNet20D = details.ForeignNet20D
			signals.Details.InstNet5D = details.InstNet5D
			signals.Details.InstNet20D = details.InstNet20D
		}
	}

	// 7. Fetch and calculate event signal
	events, err := b.fetchEvents(ctx, code, date)
	if err == nil {
		score, _, err := b.event.Calculate(ctx, code, events, date)
		if err == nil {
			signals.Event = score
		}
	}

	return signals, nil
}

// fetchPriceData fetches historical price data for momentum and technical signals
func (b *Builder) fetchPriceData(ctx context.Context, code string, date time.Time) ([]PricePoint, error) {
	// Fetch last 120 days of price data
	from := date.AddDate(0, 0, -150) // Extra buffer for weekends/holidays
	to := date

	prices, err := b.priceRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		return nil, err
	}

	// Convert to PricePoint
	pricePoints := make([]PricePoint, len(prices))
	for i, p := range prices {
		pricePoints[i] = PricePoint{
			Date:   p.Date,
			Price:  p.Close,
			Volume: p.Volume,
		}
	}

	return pricePoints, nil
}

// fetchValueMetrics fetches valuation metrics (PER, PBR, PSR)
func (b *Builder) fetchValueMetrics(ctx context.Context, code string, date time.Time) (ValueMetrics, error) {
	// Get latest financials before the date
	financial, err := b.financialRepo.GetLatestByCode(ctx, code, date)
	if err != nil {
		return ValueMetrics{}, err
	}

	// Calculate metrics
	// Note: This is simplified - actual calculation would need shares outstanding
	metrics := ValueMetrics{
		PER: financial.PER,
		PBR: financial.PBR,
		PSR: financial.PSR,
	}

	return metrics, nil
}

// fetchQualityMetrics fetches quality metrics (ROE, Debt Ratio)
func (b *Builder) fetchQualityMetrics(ctx context.Context, code string, date time.Time) (QualityMetrics, error) {
	// Get latest financials before the date
	financial, err := b.financialRepo.GetLatestByCode(ctx, code, date)
	if err != nil {
		return QualityMetrics{}, err
	}

	metrics := QualityMetrics{
		ROE:       financial.ROE,
		DebtRatio: financial.DebtRatio,
	}

	return metrics, nil
}

// fetchFlowData fetches investor flow data
func (b *Builder) fetchFlowData(ctx context.Context, code string, date time.Time) ([]FlowData, error) {
	// Fetch last 20 days of flow data
	from := date.AddDate(0, 0, -30) // Extra buffer
	to := date

	flows, err := b.flowRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		return nil, err
	}

	// Convert to FlowData
	flowData := make([]FlowData, len(flows))
	for i, f := range flows {
		flowData[i] = FlowData{
			Date:          f.Date.Format("2006-01-02"),
			ForeignNet:    f.ForeignNet,
			InstNet:       f.InstitutionNet,
			IndividualNet: f.IndividualNet,
		}
	}

	return flowData, nil
}

// fetchEvents fetches recent events (disclosures)
func (b *Builder) fetchEvents(ctx context.Context, code string, date time.Time) ([]contracts.EventSignal, error) {
	// Fetch last 90 days of disclosures
	from := date.AddDate(0, 0, -90)
	to := date

	disclosures, err := b.disclosureRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		return nil, err
	}

	// Convert disclosures to event signals
	events := make([]contracts.EventSignal, 0, len(disclosures))
	for _, d := range disclosures {
		// Map disclosure type to event type and impact
		eventType := mapDisclosureToEventType(d.Type)
		impact := GetEventImpact(eventType)

		event := contracts.EventSignal{
			Type:      string(eventType),
			Score:     impact,
			Source:    "DART",
			Timestamp: d.Date,
		}

		events = append(events, event)
	}

	return events, nil
}

// mapDisclosureToEventType maps DART disclosure types to event types
func mapDisclosureToEventType(disclosureType string) EventType {
	// Simplified mapping - actual implementation would be more sophisticated
	switch disclosureType {
	case "earnings":
		return EventEarningsPositive // Would need to check if positive/negative
	case "dividend":
		return EventDividendIncrease
	case "lawsuit":
		return EventLawsuit
	case "audit":
		return EventAuditOpinion
	case "merger":
		return EventMergerPositive
	default:
		return EventAnnouncement
	}
}
