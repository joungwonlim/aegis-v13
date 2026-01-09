package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/wonny/aegis/v13/backend/internal/realtime"
	"github.com/wonny/aegis/v13/backend/internal/realtime/cache"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

const (
	// KIS API Rate Limit: 10 req/sec
	totalRateLimit = 10

	// Tier allocations (60%, 30%, 10%)
	tier1RateLimit = 6
	tier2RateLimit = 3
	tier3RateLimit = 1
)

// TieredRESTPoller manages tiered REST polling for KIS API
// ⭐ SSOT: KIS REST 폴링 및 Rate Limit 관리는 이 폴러에서만
type TieredRESTPoller struct {
	config     *config.Config
	logger     *logger.Logger
	httpClient *httputil.Client
	cache      *cache.PriceCache

	// Tier symbols
	tier1Symbols map[string]bool
	tier2Symbols map[string]bool
	tier3Symbols map[string]bool
	tierMu       sync.RWMutex

	// Rate limiters per tier
	tier1Limiter *rate.Limiter
	tier2Limiter *rate.Limiter
	tier3Limiter *rate.Limiter

	// Intervals
	tier1Interval time.Duration // 2-5 seconds
	tier2Interval time.Duration // 10-15 seconds
	tier3Interval time.Duration // 30-60 seconds

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewTieredRESTPoller creates a new tiered REST poller
func NewTieredRESTPoller(cfg *config.Config, log *logger.Logger, httpClient *httputil.Client, priceCache *cache.PriceCache) *TieredRESTPoller {
	return &TieredRESTPoller{
		config:     cfg,
		logger:     log,
		httpClient: httpClient,
		cache:      priceCache,

		tier1Symbols: make(map[string]bool),
		tier2Symbols: make(map[string]bool),
		tier3Symbols: make(map[string]bool),

		// Rate limiters per tier
		tier1Limiter: rate.NewLimiter(rate.Limit(tier1RateLimit), tier1RateLimit),
		tier2Limiter: rate.NewLimiter(rate.Limit(tier2RateLimit), tier2RateLimit),
		tier3Limiter: rate.NewLimiter(rate.Limit(tier3RateLimit), tier3RateLimit),

		// Intervals
		tier1Interval: 3 * time.Second,  // Middle of 2-5 sec range
		tier2Interval: 12 * time.Second, // Middle of 10-15 sec range
		tier3Interval: 45 * time.Second, // Middle of 30-60 sec range

		stopCh: make(chan struct{}),
	}
}

// Start starts all tier polling loops
func (p *TieredRESTPoller) Start(ctx context.Context) error {
	p.logger.Info("Starting tiered REST poller")

	p.wg.Add(3)
	go p.pollTier1(ctx)
	go p.pollTier2(ctx)
	go p.pollTier3(ctx)

	return nil
}

// Stop stops all tier polling loops
func (p *TieredRESTPoller) Stop() {
	p.logger.Info("Stopping tiered REST poller")
	close(p.stopCh)
	p.wg.Wait()
}

// UpdateTierSymbols updates symbols for a specific tier
func (p *TieredRESTPoller) UpdateTierSymbols(tier realtime.Tier, symbols []string) {
	p.tierMu.Lock()
	defer p.tierMu.Unlock()

	symbolMap := make(map[string]bool)
	for _, code := range symbols {
		symbolMap[code] = true
	}

	switch tier {
	case realtime.Tier1:
		p.tier1Symbols = symbolMap
		p.logger.WithField("count", len(symbols)).Info("Updated Tier1 symbols")
	case realtime.Tier2:
		p.tier2Symbols = symbolMap
		p.logger.WithField("count", len(symbols)).Info("Updated Tier2 symbols")
	case realtime.Tier3:
		p.tier3Symbols = symbolMap
		p.logger.WithField("count", len(symbols)).Info("Updated Tier3 symbols")
	}
}

// pollTier1 polls Tier1 symbols (2-5 sec, high priority)
func (p *TieredRESTPoller) pollTier1(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.tier1Interval)
	defer ticker.Stop()

	p.logger.WithField("interval", p.tier1Interval).Info("Started Tier1 polling loop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.pollTier(ctx, realtime.Tier1)
		}
	}
}

// pollTier2 polls Tier2 symbols (10-15 sec, medium priority)
func (p *TieredRESTPoller) pollTier2(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.tier2Interval)
	defer ticker.Stop()

	p.logger.WithField("interval", p.tier2Interval).Info("Started Tier2 polling loop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.pollTier(ctx, realtime.Tier2)
		}
	}
}

// pollTier3 polls Tier3 symbols (30-60 sec, low priority)
func (p *TieredRESTPoller) pollTier3(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.tier3Interval)
	defer ticker.Stop()

	p.logger.WithField("interval", p.tier3Interval).Info("Started Tier3 polling loop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.pollTier(ctx, realtime.Tier3)
		}
	}
}

// pollTier polls all symbols in a specific tier
func (p *TieredRESTPoller) pollTier(ctx context.Context, tier realtime.Tier) {
	p.tierMu.RLock()
	var symbols map[string]bool
	var limiter *rate.Limiter

	switch tier {
	case realtime.Tier1:
		symbols = p.tier1Symbols
		limiter = p.tier1Limiter
	case realtime.Tier2:
		symbols = p.tier2Symbols
		limiter = p.tier2Limiter
	case realtime.Tier3:
		symbols = p.tier3Symbols
		limiter = p.tier3Limiter
	default:
		p.tierMu.RUnlock()
		return
	}

	// Make a copy to release lock quickly
	codes := make([]string, 0, len(symbols))
	for code := range symbols {
		codes = append(codes, code)
	}
	p.tierMu.RUnlock()

	if len(codes) == 0 {
		return
	}

	// Poll each symbol with rate limiting
	successCount := 0
	errorCount := 0

	for _, code := range codes {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		default:
		}

		// Wait for rate limiter
		if err := limiter.Wait(ctx); err != nil {
			p.logger.WithError(err).Error("Rate limiter wait failed")
			return
		}

		// Fetch price
		tick, err := p.fetchPrice(ctx, code)
		if err != nil {
			p.logger.WithError(err).WithField("code", code).Debug("Failed to fetch price")
			errorCount++
			continue
		}

		// Update cache
		if p.cache.Update(tick) {
			successCount++
		}
	}

	if successCount > 0 || errorCount > 0 {
		p.logger.WithFields(map[string]interface{}{
			"tier":    tier,
			"success": successCount,
			"error":   errorCount,
			"total":   len(codes),
		}).Debug("Completed tier polling")
	}
}

// fetchPrice fetches price for a single symbol from KIS REST API
func (p *TieredRESTPoller) fetchPrice(ctx context.Context, code string) (*realtime.PriceTick, error) {
	// Build KIS REST API URL with query parameters
	url := fmt.Sprintf("%s/uapi/domestic-stock/v1/quotations/inquire-price?FID_COND_MRKT_DIV_CODE=J&FID_INPUT_ISCD=%s",
		p.config.KIS.BaseURL, code)

	// Use httpClient.Get() method
	httpResp, err := p.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Parse response
	var resp KISPriceResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.RetCode != "0" {
		return nil, fmt.Errorf("KIS API error: %s - %s", resp.RetCode, resp.RetMsg)
	}

	// Convert to PriceTick
	tick := p.convertToPriceTick(code, &resp)
	return tick, nil
}

// convertToPriceTick converts KIS REST response to PriceTick
func (p *TieredRESTPoller) convertToPriceTick(code string, resp *KISPriceResponse) *realtime.PriceTick {
	return &realtime.PriceTick{
		Code:       code,
		Price:      resp.Output.CurrentPrice,
		Change:     resp.Output.PriceChange,
		ChangeRate: resp.Output.ChangeRate,
		Volume:     resp.Output.AccVolume,
		Value:      resp.Output.AccTradeValue,
		High:       resp.Output.HighPrice,
		Low:        resp.Output.LowPrice,
		Open:       resp.Output.OpenPrice,
		PrevClose:  resp.Output.PrevClose,
		Timestamp:  time.Now(),
		Source:     string(realtime.SourceKISREST),
		IsStale:    false,
	}
}

// GetTierStats returns statistics for each tier
func (p *TieredRESTPoller) GetTierStats() TierStats {
	p.tierMu.RLock()
	defer p.tierMu.RUnlock()

	return TierStats{
		Tier1Count: len(p.tier1Symbols),
		Tier2Count: len(p.tier2Symbols),
		Tier3Count: len(p.tier3Symbols),
		Tier1Rate:  tier1RateLimit,
		Tier2Rate:  tier2RateLimit,
		Tier3Rate:  tier3RateLimit,
	}
}

// TierStats represents statistics for tiered polling
type TierStats struct {
	Tier1Count int `json:"tier1_count"`
	Tier2Count int `json:"tier2_count"`
	Tier3Count int `json:"tier3_count"`
	Tier1Rate  int `json:"tier1_rate"`
	Tier2Rate  int `json:"tier2_rate"`
	Tier3Rate  int `json:"tier3_rate"`
}

// KISPriceResponse represents KIS REST API price response
type KISPriceResponse struct {
	RetCode string `json:"rt_cd"`
	RetMsg  string `json:"msg1"`
	Output  struct {
		CurrentPrice   int64   `json:"stck_prpr,string"`      // 현재가
		PriceChange    int64   `json:"prdy_vrss,string"`      // 전일대비
		ChangeRate     float64 `json:"prdy_ctrt,string"`      // 등락율
		AccVolume      int64   `json:"acml_vol,string"`       // 누적거래량
		AccTradeValue  int64   `json:"acml_tr_pbmn,string"`   // 누적거래대금
		HighPrice      int64   `json:"stck_hgpr,string"`      // 고가
		LowPrice       int64   `json:"stck_lwpr,string"`      // 저가
		OpenPrice      int64   `json:"stck_oprc,string"`      // 시가
		PrevClose      int64   `json:"stck_sdpr,string"`      // 전일종가
	} `json:"output"`
}
