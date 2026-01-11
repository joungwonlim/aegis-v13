package selection

import (
	"context"
	"sort"

	"github.com/wonny/aegis/v13/backend/internal/contracts"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Ranker implements S4: Ranking with weighted scores
// ⭐ SSOT: S4 랭킹 로직은 여기서만
type Ranker struct {
	weights WeightConfig
	logger  *logger.Logger
}

// WeightConfig defines signal weights for total score calculation
// SSOT: config/strategy/korea_equity_v13.yaml ranking.weights_pct
type WeightConfig struct {
	Momentum  float64 // 모멘텀 (기본: 25%)
	Flow      float64 // 수급 (기본: 20%)
	Technical float64 // 기술적 (기본: 15%)
	Event     float64 // 이벤트 (기본: 15%)
	Value     float64 // 가치 (기본: 15%)
	Quality   float64 // 퀄리티 (기본: 10%)
}

// NewRanker creates a new ranker
func NewRanker(weights WeightConfig, logger *logger.Logger) *Ranker {
	return &Ranker{
		weights: weights,
		logger:  logger,
	}
}

// Rank calculates total scores and ranks stocks
func (r *Ranker) Rank(ctx context.Context, codes []string, signals *contracts.SignalSet) ([]contracts.RankedStock, error) {
	ranked := make([]contracts.RankedStock, 0, len(codes))

	for _, code := range codes {
		signal, exists := signals.Signals[code]
		if !exists {
			r.logger.WithFields(map[string]interface{}{
				"code": code,
			}).Warn("Signal not found for code")
			continue
		}

		// Calculate weighted total score
		totalScore := r.calculateTotalScore(signal)

		ranked = append(ranked, contracts.RankedStock{
			Code:       code,
			TotalScore: totalScore,
			Scores: contracts.ScoreDetail{
				Momentum:  signal.Momentum,
				Technical: signal.Technical,
				Value:     signal.Value,
				Quality:   signal.Quality,
				Flow:      signal.Flow,
				Event:     signal.Event,
			},
		})
	}

	// Sort by total score (descending)
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].TotalScore > ranked[j].TotalScore
	})

	// Assign ranks
	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	// Log result (handle empty case)
	if len(ranked) > 0 {
		r.logger.WithFields(map[string]interface{}{
			"total_stocks": len(ranked),
			"top_score":    ranked[0].TotalScore,
			"top_code":     ranked[0].Code,
		}).Info("Ranking completed")
	} else {
		r.logger.WithFields(map[string]interface{}{
			"total_stocks": 0,
		}).Warn("Ranking completed with no stocks")
	}

	return ranked, nil
}

// calculateTotalScore calculates weighted total score
func (r *Ranker) calculateTotalScore(signal *contracts.StockSignals) float64 {
	return signal.Momentum*r.weights.Momentum +
		signal.Technical*r.weights.Technical +
		signal.Value*r.weights.Value +
		signal.Quality*r.weights.Quality +
		signal.Flow*r.weights.Flow +
		signal.Event*r.weights.Event
}

// ValidateWeights checks if weights sum to 1.0
func (w *WeightConfig) ValidateWeights() bool {
	sum := w.Momentum + w.Technical + w.Value + w.Quality + w.Flow + w.Event
	// Allow small floating point error
	return sum >= 0.99 && sum <= 1.01
}

// DefaultWeightConfig returns default weight configuration
// SSOT: config/strategy/korea_equity_v13.yaml ranking.weights_pct
func DefaultWeightConfig() WeightConfig {
	return WeightConfig{
		Momentum:  0.25, // 25% - 모멘텀 ⭐
		Flow:      0.20, // 20% - 수급
		Technical: 0.15, // 15% - 기술적
		Event:     0.15, // 15% - 이벤트
		Value:     0.15, // 15% - 가치
		Quality:   0.10, // 10% - 퀄리티
	}
	// Total: 100%
}
