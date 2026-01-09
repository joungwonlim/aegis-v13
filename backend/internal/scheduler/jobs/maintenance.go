package jobs

import (
	"context"

	"github.com/wonny/aegis/v13/backend/internal/realtime/cache"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// CacheCleanupJob cleans stale prices from cache
type CacheCleanupJob struct {
	cache  *cache.PriceCache
	logger *logger.Logger
}

// NewCacheCleanupJob creates a new cache cleanup job
func NewCacheCleanupJob(priceCache *cache.PriceCache, log *logger.Logger) *CacheCleanupJob {
	return &CacheCleanupJob{
		cache:  priceCache,
		logger: log,
	}
}

// Name returns the job name
func (j *CacheCleanupJob) Name() string {
	return "cache_cleanup"
}

// Schedule returns the cron schedule (every 5 minutes)
func (j *CacheCleanupJob) Schedule() string {
	return "0 */5 * * * *" // Every 5 minutes
}

// Run executes the cache cleanup
func (j *CacheCleanupJob) Run(ctx context.Context) error {
	j.logger.Debug("Starting scheduled cache cleanup")

	count := j.cache.CleanStale()

	if count > 0 {
		j.logger.WithField("removed", count).Info("Cache cleanup completed")
	}

	return nil
}
