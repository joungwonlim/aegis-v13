package s2_signals

import (
	"context"
	"math"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// EventCalculator calculates event signals
// ⭐ SSOT: 이벤트 시그널 계산은 여기서만
type EventCalculator struct {
	logger *logger.Logger
}

// NewEventCalculator creates a new event calculator
func NewEventCalculator(log *logger.Logger) *EventCalculator {
	return &EventCalculator{
		logger: log,
	}
}

// Calculate calculates event signal for a stock
func (c *EventCalculator) Calculate(ctx context.Context, code string, events []contracts.EventSignal, currentDate time.Time) (float64, contracts.SignalDetails, error) {
	details := contracts.SignalDetails{}

	if len(events) == 0 {
		// No events → neutral
		return 0.0, details, nil
	}

	// Calculate weighted event score
	score := c.calculateScore(events, currentDate)

	c.logger.WithFields(map[string]interface{}{
		"code":        code,
		"event_count": len(events),
		"score":       score,
	}).Debug("Calculated event signal")

	return score, details, nil
}

// calculateScore calculates event score (-1.0 ~ 1.0)
func (c *EventCalculator) calculateScore(events []contracts.EventSignal, currentDate time.Time) float64 {
	if len(events) == 0 {
		return 0.0
	}

	var weightedSum float64
	var totalWeight float64

	for _, event := range events {
		// Get event score (already normalized to -1.0 ~ 1.0)
		score := event.Score

		// Apply time decay
		// Recent events matter more
		daysSince := currentDate.Sub(event.Timestamp).Hours() / 24
		timeWeight := c.calculateTimeWeight(daysSince)

		// Weight the event score by time
		eventScore := score * timeWeight

		weightedSum += eventScore
		totalWeight += timeWeight
	}

	if totalWeight == 0 {
		return 0.0
	}

	// Calculate weighted average
	finalScore := weightedSum / totalWeight

	// Clamp to -1.0 ~ 1.0
	if finalScore > 1.0 {
		finalScore = 1.0
	} else if finalScore < -1.0 {
		finalScore = -1.0
	}

	return finalScore
}

// calculateTimeWeight calculates time-based weight
// Recent events have more weight
func (c *EventCalculator) calculateTimeWeight(daysSince float64) float64 {
	// Exponential decay
	// Within 7 days: ~100%
	// Within 30 days: ~50%
	// Within 90 days: ~25%
	// Beyond 90 days: ~10%

	// Use exponential decay: weight = exp(-k * days)
	// k = 0.023 gives reasonable decay curve
	const decayRate = 0.023

	weight := math.Exp(-decayRate * daysSince)

	// Floor at 0.1 (don't completely ignore old events)
	if weight < 0.1 {
		weight = 0.1
	}

	return weight
}

// EventType represents different types of events
type EventType string

const (
	// Positive events
	EventEarningsPositive EventType = "earnings_positive"   // 실적 개선
	EventDividendIncrease EventType = "dividend_increase"   // 배당 증가
	EventNewProduct       EventType = "new_product"         // 신제품 출시
	EventCapexIncrease    EventType = "capex_increase"      // 설비 투자
	EventShareBuyback     EventType = "share_buyback"       // 자사주 매입
	EventMergerPositive   EventType = "merger_positive"     // 긍정적 인수합병
	EventPartnership      EventType = "partnership"         // 파트너십 체결
	EventPatent           EventType = "patent"              // 특허 취득

	// Negative events
	EventEarningsNegative EventType = "earnings_negative"   // 실적 악화
	EventDividendDecrease EventType = "dividend_decrease"   // 배당 감소
	EventLawsuit          EventType = "lawsuit"             // 소송
	EventAuditOpinion     EventType = "audit_opinion"       // 감사 의견
	EventMergerNegative   EventType = "merger_negative"     // 부정적 인수합병
	EventManagementChange EventType = "management_change"   // 경영진 교체
	EventRegulatory       EventType = "regulatory"          // 규제 이슈
	EventRecall           EventType = "recall"              // 제품 리콜

	// Neutral events
	EventGeneralNews EventType = "general_news" // 일반 뉴스
	EventAnnouncement EventType = "announcement" // 일반 공시
)

// GetEventImpact returns the base impact score for an event type
func GetEventImpact(eventType EventType) float64 {
	impactMap := map[EventType]float64{
		// Positive events (0.3 ~ 1.0)
		EventEarningsPositive: 1.0,
		EventDividendIncrease: 0.6,
		EventNewProduct:       0.7,
		EventCapexIncrease:    0.5,
		EventShareBuyback:     0.8,
		EventMergerPositive:   0.9,
		EventPartnership:      0.6,
		EventPatent:           0.5,

		// Negative events (-0.3 ~ -1.0)
		EventEarningsNegative: -1.0,
		EventDividendDecrease: -0.6,
		EventLawsuit:          -0.7,
		EventAuditOpinion:     -0.9,
		EventMergerNegative:   -0.8,
		EventManagementChange: -0.5,
		EventRegulatory:       -0.7,
		EventRecall:           -0.8,

		// Neutral events
		EventGeneralNews:  0.0,
		EventAnnouncement: 0.0,
	}

	if impact, ok := impactMap[eventType]; ok {
		return impact
	}

	return 0.0 // Default neutral
}
