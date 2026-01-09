package cache

import (
	"sync"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/realtime"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// PriceCache is an in-memory cache for real-time prices
// ⭐ SSOT: 실시간 가격 캐싱은 이 구조체에서만
type PriceCache struct {
	mu      sync.RWMutex
	prices  map[string]*realtime.PriceTick
	ttl     time.Duration
	logger  *logger.Logger
}

// NewPriceCache creates a new price cache
func NewPriceCache(ttl time.Duration, log *logger.Logger) *PriceCache {
	return &PriceCache{
		prices: make(map[string]*realtime.PriceTick),
		ttl:    ttl,
		logger: log,
	}
}

// Update updates price in cache
// Only accepts newer data from higher priority sources
func (c *PriceCache) Update(tick *realtime.PriceTick) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	existing, exists := c.prices[tick.Code]

	// Check if update should be accepted
	if exists {
		// Don't accept older data
		if tick.Timestamp.Before(existing.Timestamp) {
			c.logger.WithFields(map[string]interface{}{
				"code":      tick.Code,
				"new_time":  tick.Timestamp,
				"old_time":  existing.Timestamp,
				"new_source": tick.Source,
				"old_source": existing.Source,
			}).Debug("Rejected older price data")
			return false
		}

		// Accept same timestamp only from higher priority source
		if tick.Timestamp.Equal(existing.Timestamp) {
			newSource := realtime.PriceSource(tick.Source)
			oldSource := realtime.PriceSource(existing.Source)
			if newSource.Priority() <= oldSource.Priority() {
				return false
			}
		}
	}

	// Mark as stale if older than TTL
	tick.IsStale = time.Since(tick.Timestamp) > c.ttl

	// Update cache
	c.prices[tick.Code] = tick

	c.logger.WithFields(map[string]interface{}{
		"code":   tick.Code,
		"price":  tick.Price,
		"source": tick.Source,
		"stale":  tick.IsStale,
	}).Debug("Updated price cache")

	return true
}

// Get retrieves price from cache
func (c *PriceCache) Get(code string) (*realtime.PriceTick, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tick, exists := c.prices[code]
	if !exists {
		return nil, false
	}

	// Check staleness
	if time.Since(tick.Timestamp) > c.ttl {
		tick.IsStale = true
	}

	return tick, true
}

// GetAll retrieves all prices from cache
func (c *PriceCache) GetAll() map[string]*realtime.PriceTick {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*realtime.PriceTick, len(c.prices))
	for code, tick := range c.prices {
		// Check staleness
		if time.Since(tick.Timestamp) > c.ttl {
			tick.IsStale = true
		}
		result[code] = tick
	}

	return result
}

// GetMany retrieves multiple prices from cache
func (c *PriceCache) GetMany(codes []string) map[string]*realtime.PriceTick {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*realtime.PriceTick, len(codes))
	for _, code := range codes {
		if tick, exists := c.prices[code]; exists {
			// Check staleness
			if time.Since(tick.Timestamp) > c.ttl {
				tick.IsStale = true
			}
			result[code] = tick
		}
	}

	return result
}

// Delete removes price from cache
func (c *PriceCache) Delete(code string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.prices, code)
}

// Clear clears all prices from cache
func (c *PriceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.prices = make(map[string]*realtime.PriceTick)
	c.logger.Info("Cleared price cache")
}

// Len returns the number of prices in cache
func (c *PriceCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.prices)
}

// CleanStale removes stale prices from cache
func (c *PriceCache) CleanStale() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0

	for code, tick := range c.prices {
		if now.Sub(tick.Timestamp) > c.ttl {
			delete(c.prices, code)
			count++
		}
	}

	if count > 0 {
		c.logger.WithField("count", count).Info("Cleaned stale prices from cache")
	}

	return count
}

// Stats returns cache statistics
func (c *PriceCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		TotalCount: len(c.prices),
	}

	now := time.Now()
	for _, tick := range c.prices {
		age := now.Sub(tick.Timestamp)
		if age > c.ttl {
			stats.StaleCount++
		}

		// Track by source
		switch realtime.PriceSource(tick.Source) {
		case realtime.SourceKISWebSocket:
			stats.KISWebSocketCount++
		case realtime.SourceKISREST:
			stats.KISRESTCount++
		case realtime.SourceNaver:
			stats.NaverCount++
		}
	}

	stats.FreshCount = stats.TotalCount - stats.StaleCount

	return stats
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalCount        int `json:"total_count"`
	FreshCount        int `json:"fresh_count"`
	StaleCount        int `json:"stale_count"`
	KISWebSocketCount int `json:"kis_websocket_count"`
	KISRESTCount      int `json:"kis_rest_count"`
	NaverCount        int `json:"naver_count"`
}
