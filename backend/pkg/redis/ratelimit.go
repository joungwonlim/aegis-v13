package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements sliding window rate limiting using Redis
// ⭐ SSOT: 레이트 리밋은 여기서만
type RateLimiter struct {
	client *Client
	prefix string
}

// RateLimitConfig defines rate limit parameters
type RateLimitConfig struct {
	Key      string        // Unique identifier (e.g., "kis", "dart", "naver")
	Limit    int           // Maximum requests allowed
	Window   time.Duration // Time window
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(client *Client, prefix string) *RateLimiter {
	return &RateLimiter{
		client: client,
		prefix: prefix,
	}
}

// Allow checks if a request is allowed under the rate limit
// Returns (allowed, remaining, error)
func (r *RateLimiter) Allow(ctx context.Context, cfg RateLimitConfig) (bool, int, error) {
	if !r.client.Enabled() {
		// If Redis is disabled, allow all requests
		return true, cfg.Limit, nil
	}

	key := fmt.Sprintf("%s:ratelimit:%s", r.prefix, cfg.Key)
	now := time.Now().UnixMilli()
	windowStart := now - cfg.Window.Milliseconds()

	rdb := r.client.Redis()

	// Use Lua script for atomic operation
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_ms = tonumber(ARGV[4])

		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current requests in window
		local count = redis.call('ZCARD', key)

		if count < limit then
			-- Add current request
			redis.call('ZADD', key, now, now)
			redis.call('PEXPIRE', key, window_ms)
			return {1, limit - count - 1}
		else
			return {0, 0}
		end
	`)

	result, err := script.Run(ctx, rdb, []string{key},
		now,
		windowStart,
		cfg.Limit,
		cfg.Window.Milliseconds(),
	).Slice()
	if err != nil {
		return false, 0, fmt.Errorf("rate limit script failed: %w", err)
	}

	allowed := result[0].(int64) == 1
	remaining := int(result[1].(int64))

	return allowed, remaining, nil
}

// Wait blocks until a request is allowed or context is cancelled
func (r *RateLimiter) Wait(ctx context.Context, cfg RateLimitConfig) error {
	for {
		allowed, _, err := r.Allow(ctx, cfg)
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Retry
		}
	}
}

// Predefined rate limit configs for external APIs
var (
	// KIS API: 초당 5회 제한
	KISRateLimit = RateLimitConfig{
		Key:    "kis",
		Limit:  5,
		Window: time.Second,
	}

	// DART API: 분당 100회 제한 (보수적)
	DARTRateLimit = RateLimitConfig{
		Key:    "dart",
		Limit:  100,
		Window: time.Minute,
	}

	// Naver Finance: 초당 10회 제한 (보수적)
	NaverRateLimit = RateLimitConfig{
		Key:    "naver",
		Limit:  10,
		Window: time.Second,
	}
)
