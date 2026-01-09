package feed

import (
	"context"
	"sync"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/realtime"
	"github.com/wonny/aegis/v13/backend/internal/realtime/cache"
	"github.com/wonny/aegis/v13/backend/internal/realtime/queue"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// FeedManager orchestrates all real-time price feeds
// ⭐ SSOT: 실시간 가격 피드 조율은 이 매니저에서만
type FeedManager struct {
	config     *config.Config
	logger     *logger.Logger

	// Feed sources
	wsClient   *KISWebSocketClient
	restPoller *TieredRESTPoller
	naverFeed  *NaverFeed

	// Infrastructure
	cache      *cache.PriceCache
	syncQueue  *queue.SyncQueue

	// Symbol priority management
	priorities map[string]*realtime.SymbolPriority
	priorityMu sync.RWMutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewFeedManager creates a new feed manager
func NewFeedManager(cfg *config.Config, log *logger.Logger, httpClient *httputil.Client, priceCache *cache.PriceCache, syncQueue *queue.SyncQueue) *FeedManager {
	return &FeedManager{
		config:     cfg,
		logger:     log,
		wsClient:   NewKISWebSocketClient(cfg, log, priceCache),
		restPoller: NewTieredRESTPoller(cfg, log, httpClient, priceCache),
		naverFeed:  NewNaverFeed(httpClient, log, priceCache),
		cache:      priceCache,
		syncQueue:  syncQueue,
		priorities: make(map[string]*realtime.SymbolPriority),
		stopCh:     make(chan struct{}),
	}
}

// Start starts all feed sources
func (m *FeedManager) Start(ctx context.Context) error {
	m.logger.Info("Starting feed manager")

	// Start WebSocket client
	if err := m.wsClient.Start(ctx); err != nil {
		return err
	}

	// Start REST poller
	if err := m.restPoller.Start(ctx); err != nil {
		return err
	}

	// Start Naver feed (backup source)
	if err := m.naverFeed.Start(ctx); err != nil {
		return err
	}

	// Start sync queue worker
	go m.syncQueue.Start(ctx)

	// Start priority rebalancing loop
	m.wg.Add(1)
	go m.rebalanceLoop(ctx)

	// Start cache to DB sync loop
	m.wg.Add(1)
	go m.syncLoop(ctx)

	m.logger.Info("Feed manager started successfully")
	return nil
}

// Stop stops all feed sources
func (m *FeedManager) Stop() {
	m.logger.Info("Stopping feed manager")

	close(m.stopCh)

	// Stop feed sources
	m.wsClient.Stop()
	m.restPoller.Stop()
	m.naverFeed.Stop()

	// Stop sync queue
	m.syncQueue.Stop()

	// Wait for goroutines
	m.wg.Wait()

	m.logger.Info("Feed manager stopped")
}

// UpdateSymbolPriority updates the priority of a symbol
func (m *FeedManager) UpdateSymbolPriority(code string, priority *realtime.SymbolPriority) {
	m.priorityMu.Lock()
	m.priorities[code] = priority
	m.priorityMu.Unlock()

	// Update WebSocket client
	m.wsClient.UpdatePriority(priority)

	m.logger.WithFields(map[string]interface{}{
		"code":  code,
		"score": priority.Score,
	}).Debug("Updated symbol priority")
}

// RemoveSymbol removes a symbol from tracking
func (m *FeedManager) RemoveSymbol(code string) {
	m.priorityMu.Lock()
	delete(m.priorities, code)
	m.priorityMu.Unlock()

	// Remove from WebSocket client
	m.wsClient.RemoveSymbol(code)

	m.logger.WithField("code", code).Debug("Removed symbol from tracking")
}

// AddPortfolioSymbol adds a symbol from portfolio (highest priority)
func (m *FeedManager) AddPortfolioSymbol(code string) {
	priority := &realtime.SymbolPriority{
		Code:        code,
		InPortfolio: true,
	}
	priority.CalculateScore()

	m.UpdateSymbolPriority(code, priority)
}

// AddActiveOrderSymbol adds a symbol with active order (very high priority)
func (m *FeedManager) AddActiveOrderSymbol(code string) {
	priority := &realtime.SymbolPriority{
		Code:           code,
		HasActiveOrder: true,
	}
	priority.CalculateScore()

	m.UpdateSymbolPriority(code, priority)
}

// AddWatchingSymbol adds a symbol being watched by user (high priority)
func (m *FeedManager) AddWatchingSymbol(code string) {
	priority := &realtime.SymbolPriority{
		Code:         code,
		UserWatching: true,
	}
	priority.CalculateScore()

	m.UpdateSymbolPriority(code, priority)
}

// rebalanceLoop periodically rebalances symbol tiers
func (m *FeedManager) rebalanceLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.rebalanceTiers()
		}
	}
}

// rebalanceTiers rebalances symbols across tiers based on priority
func (m *FeedManager) rebalanceTiers() {
	m.priorityMu.RLock()
	priorities := make([]*realtime.SymbolPriority, 0, len(m.priorities))
	for _, priority := range m.priorities {
		priorities = append(priorities, priority)
	}
	m.priorityMu.RUnlock()

	if len(priorities) == 0 {
		return
	}

	// Separate symbols by tier
	tier1Symbols := make([]string, 0)
	tier2Symbols := make([]string, 0)
	tier3Symbols := make([]string, 0)

	wsSymbols := m.wsClient.GetActiveSymbols()
	wsSymbolsMap := make(map[string]bool)
	for _, code := range wsSymbols {
		wsSymbolsMap[code] = true
	}

	for _, priority := range priorities {
		// WebSocket symbols don't need REST polling
		if wsSymbolsMap[priority.Code] {
			continue
		}

		tier := realtime.GetTierFromScore(priority.Score)
		switch tier {
		case realtime.Tier1:
			tier1Symbols = append(tier1Symbols, priority.Code)
		case realtime.Tier2:
			tier2Symbols = append(tier2Symbols, priority.Code)
		case realtime.Tier3:
			tier3Symbols = append(tier3Symbols, priority.Code)
		}
	}

	// Update REST poller tiers
	m.restPoller.UpdateTierSymbols(realtime.Tier1, tier1Symbols)
	m.restPoller.UpdateTierSymbols(realtime.Tier2, tier2Symbols)
	m.restPoller.UpdateTierSymbols(realtime.Tier3, tier3Symbols)

	m.logger.WithFields(map[string]interface{}{
		"websocket": len(wsSymbols),
		"tier1":     len(tier1Symbols),
		"tier2":     len(tier2Symbols),
		"tier3":     len(tier3Symbols),
	}).Info("Rebalanced symbol tiers")
}

// syncLoop periodically syncs cache to database
func (m *FeedManager) syncLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.syncCacheToQueue(ctx)
		}
	}
}

// syncCacheToQueue syncs cache prices to sync queue
func (m *FeedManager) syncCacheToQueue(ctx context.Context) {
	prices := m.cache.GetAll()

	if len(prices) == 0 {
		return
	}

	ticks := make([]*realtime.PriceTick, 0, len(prices))
	for _, tick := range prices {
		// Only sync non-stale prices
		if !tick.IsStale {
			ticks = append(ticks, tick)
		}
	}

	if len(ticks) == 0 {
		return
	}

	if err := m.syncQueue.EnqueueBatch(ctx, ticks); err != nil {
		m.logger.WithError(err).Error("Failed to enqueue batch to sync queue")
		return
	}

	m.logger.WithField("count", len(ticks)).Debug("Synced cache to queue")
}

// GetStats returns statistics for all components
func (m *FeedManager) GetStats(ctx context.Context) (*FeedStats, error) {
	cacheStats := m.cache.Stats()
	tierStats := m.restPoller.GetTierStats()
	queueStats, err := m.syncQueue.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	wsSymbols := m.wsClient.GetActiveSymbols()

	return &FeedStats{
		WebSocketSymbols: len(wsSymbols),
		Tier1Symbols:     tierStats.Tier1Count,
		Tier2Symbols:     tierStats.Tier2Count,
		Tier3Symbols:     tierStats.Tier3Count,
		CacheTotal:       cacheStats.TotalCount,
		CacheFresh:       cacheStats.FreshCount,
		CacheStale:       cacheStats.StaleCount,
		QueuePending:     queueStats.Pending,
		QueueDone:        queueStats.Done,
		QueueFailed:      queueStats.Failed,
	}, nil
}

// FeedStats represents statistics for the feed manager
type FeedStats struct {
	WebSocketSymbols int `json:"websocket_symbols"`
	Tier1Symbols     int `json:"tier1_symbols"`
	Tier2Symbols     int `json:"tier2_symbols"`
	Tier3Symbols     int `json:"tier3_symbols"`
	CacheTotal       int `json:"cache_total"`
	CacheFresh       int `json:"cache_fresh"`
	CacheStale       int `json:"cache_stale"`
	QueuePending     int `json:"queue_pending"`
	QueueDone        int `json:"queue_done"`
	QueueFailed      int `json:"queue_failed"`
}
