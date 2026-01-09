package feed

import (
	"context"
	"sync"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/external/naver"
	"github.com/wonny/aegis/v13/backend/internal/realtime"
	"github.com/wonny/aegis/v13/backend/internal/realtime/cache"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

const (
	// Naver polling interval (backup source, less frequent)
	naverPollInterval = 60 * time.Second
)

// NaverFeed provides backup real-time prices from Naver
// ⭐ SSOT: Naver 백업 소스는 이 피드에서만
type NaverFeed struct {
	logger      *logger.Logger
	naverClient *naver.Client
	cache       *cache.PriceCache

	symbols   map[string]bool
	symbolsMu sync.RWMutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewNaverFeed creates a new Naver feed
func NewNaverFeed(httpClient *httputil.Client, log *logger.Logger, priceCache *cache.PriceCache) *NaverFeed {
	return &NaverFeed{
		logger:      log,
		naverClient: naver.NewClient(httpClient, log),
		cache:       priceCache,
		symbols:     make(map[string]bool),
		stopCh:      make(chan struct{}),
	}
}

// Start starts the Naver feed
func (f *NaverFeed) Start(ctx context.Context) error {
	f.logger.Info("Starting Naver feed (backup source)")

	f.wg.Add(1)
	go f.pollLoop(ctx)

	return nil
}

// Stop stops the Naver feed
func (f *NaverFeed) Stop() {
	f.logger.Info("Stopping Naver feed")
	close(f.stopCh)
	f.wg.Wait()
}

// UpdateSymbols updates the symbols to track
func (f *NaverFeed) UpdateSymbols(symbols []string) {
	f.symbolsMu.Lock()
	defer f.symbolsMu.Unlock()

	f.symbols = make(map[string]bool)
	for _, code := range symbols {
		f.symbols[code] = true
	}

	f.logger.WithField("count", len(symbols)).Info("Updated Naver feed symbols")
}

// pollLoop polls Naver for prices
func (f *NaverFeed) pollLoop(ctx context.Context) {
	defer f.wg.Done()

	ticker := time.NewTicker(naverPollInterval)
	defer ticker.Stop()

	f.logger.WithField("interval", naverPollInterval).Info("Started Naver polling loop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-f.stopCh:
			return
		case <-ticker.C:
			f.pollPrices(ctx)
		}
	}
}

// pollPrices polls prices for all tracked symbols
func (f *NaverFeed) pollPrices(ctx context.Context) {
	f.symbolsMu.RLock()
	codes := make([]string, 0, len(f.symbols))
	for code := range f.symbols {
		codes = append(codes, code)
	}
	f.symbolsMu.RUnlock()

	if len(codes) == 0 {
		return
	}

	successCount := 0
	errorCount := 0

	for _, code := range codes {
		select {
		case <-ctx.Done():
			return
		case <-f.stopCh:
			return
		default:
		}

		tick, err := f.fetchPrice(ctx, code)
		if err != nil {
			f.logger.WithError(err).WithField("code", code).Debug("Failed to fetch price from Naver")
			errorCount++
			continue
		}

		// Only update cache if price is not already fresher from other sources
		if f.cache.Update(tick) {
			successCount++
		}
	}

	if successCount > 0 || errorCount > 0 {
		f.logger.WithFields(map[string]interface{}{
			"success": successCount,
			"error":   errorCount,
			"total":   len(codes),
		}).Debug("Completed Naver polling")
	}
}

// fetchPrice fetches price for a single symbol from Naver
func (f *NaverFeed) fetchPrice(ctx context.Context, code string) (*realtime.PriceTick, error) {
	// NOTE: Naver client doesn't have real-time price API yet
	// This is a placeholder for future implementation
	// For now, Naver feed is disabled and KIS sources are used exclusively

	// TODO: Implement real-time price fetching from Naver Finance
	// Options:
	// 1. Add GetCurrentPrice() method to naver.Client
	// 2. Parse Naver Finance real-time page
	// 3. Use Naver Finance API if available

	return nil, nil
}
