package s2_signals

import (
	"context"
	"fmt"
	"strings"
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
	// Fetch last 120+ days of price data
	// Need 200 calendar days to get ~120 trading days (weekends/holidays)
	from := date.AddDate(0, 0, -200)
	to := date

	prices, err := b.priceRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		return nil, err
	}

	// Convert to PricePoint in reverse order (DESC: newest first)
	// momentum.go and technical.go expect prices[0] to be the most recent
	n := len(prices)
	pricePoints := make([]PricePoint, n)
	for i, p := range prices {
		pricePoints[n-1-i] = PricePoint{
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
	// Fetch last 20+ days of flow data
	// Need 40 calendar days to get ~20 trading days
	from := date.AddDate(0, 0, -40)
	to := date

	flows, err := b.flowRepo.GetByCodeAndDateRange(ctx, code, from, to)
	if err != nil {
		return nil, err
	}

	// Convert to FlowData in reverse order (DESC: newest first)
	// flow.go expects flowData[0] to be the most recent
	n := len(flows)
	flowData := make([]FlowData, n)
	for i, f := range flows {
		flowData[n-1-i] = FlowData{
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
		// Map disclosure title to event type and impact
		// DB category field contains market type (KOSPI, KOSDAQ), so we parse title instead
		eventType := mapDisclosureToEventType(d.Title)
		impact := GetEventImpact(eventType)

		// Debug: 공시 제목과 매핑된 이벤트 타입 로깅
		b.logger.WithFields(map[string]interface{}{
			"code":       code,
			"title":      d.Title,
			"event_type": string(eventType),
			"impact":     impact,
		}).Debug("Disclosure mapped to event")

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

// mapDisclosureToEventType maps DART disclosure title to event types
// Parses Korean keywords from title to determine event type
func mapDisclosureToEventType(title string) EventType {
	// strings.Contains를 사용하기 위해 import 필요
	// 공시 제목에서 키워드를 찾아 이벤트 유형 결정

	// Positive events
	// 자기주식/자사주 매입
	if containsAny(title, "자기주식취득", "자사주매입", "자기주식매입", "자기주식신탁") {
		return EventShareBuyback
	}

	// 전환사채 조기 상환/취득 (희석 위험 감소 → 긍정적)
	if containsAny(title, "사채취득", "조기상환", "사채상환") {
		return EventShareBuyback // 희석 위험 감소
	}

	// 대량보유 보고 (기관 관심 → 약간 긍정적)
	if containsAny(title, "대량보유상황보고", "주식등의대량보유") {
		return EventPartnership // 0.6 점
	}

	// 배당
	if containsAny(title, "배당", "배당금") {
		return EventDividendIncrease
	}

	// 신규사업/신제품
	if containsAny(title, "신규사업", "신제품", "신규계약") {
		return EventNewProduct
	}

	// 설비투자
	if containsAny(title, "설비투자", "공장증설", "투자결정") {
		return EventCapexIncrease
	}

	// 인수합병 (긍정적으로 가정)
	if containsAny(title, "인수", "합병", "경영권") {
		return EventMergerPositive
	}

	// 파트너십/MOU
	if containsAny(title, "MOU", "양해각서", "업무협약", "파트너십", "제휴") {
		return EventPartnership
	}

	// 특허
	if containsAny(title, "특허", "기술이전") {
		return EventPatent
	}

	// Negative events
	// 소송
	if containsAny(title, "소송", "소제기", "피소", "손해배상") {
		return EventLawsuit
	}

	// 감사의견
	if containsAny(title, "감사의견", "감사보고서", "한정의견", "부적정의견") {
		return EventAuditOpinion
	}

	// 규제/행정처분
	if containsAny(title, "행정처분", "과징금", "제재", "시정명령") {
		return EventRegulatory
	}

	// 경영진 변경 (부정적: 사임, 해임)
	if containsAny(title, "사임", "해임", "퇴임") {
		return EventManagementChange
	}

	// 경영진 선임 (중립)
	if containsAny(title, "대표이사", "임원", "선임", "이사회") {
		return EventGeneralNews
	}

	// 리콜
	if containsAny(title, "리콜", "자진회수") {
		return EventRecall
	}

	// 실적 관련 (구체적인 내용을 알 수 없으므로 중립으로 처리)
	if containsAny(title, "실적", "매출", "영업이익", "순이익", "사업보고서", "반기보고서", "분기보고서") {
		return EventGeneralNews
	}

	// 증자 (희석 가능성 있으므로 중립)
	if containsAny(title, "유상증자", "무상증자", "증자결정") {
		return EventGeneralNews
	}

	// 전환사채 (희석 가능성 있으므로 중립)
	if containsAny(title, "전환사채", "CB발행", "CB)", "신주인수권", "사채권발행") {
		return EventGeneralNews
	}

	// 주주총회
	if containsAny(title, "주주총회", "임시주총", "정기주총") {
		return EventGeneralNews
	}

	// 기타 일반 공시
	return EventAnnouncement
}

// containsAny checks if s contains any of the keywords
func containsAny(s string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}
