package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/wonny/aegis/v13/backend/pkg/config"
)

// Client wraps the Redis client with additional utilities
// ⭐ SSOT: Redis 연결은 여기서만 관리
type Client struct {
	rdb     *redis.Client
	enabled bool
}

// New creates a new Redis client
func New(cfg *config.Config) (*Client, error) {
	if !cfg.Redis.Enabled {
		return &Client{enabled: false}, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Client{
		rdb:     rdb,
		enabled: true,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.rdb != nil {
		return c.rdb.Close()
	}
	return nil
}

// Enabled returns whether Redis is enabled
func (c *Client) Enabled() bool {
	return c.enabled
}

// Redis returns the underlying redis client for advanced usage
func (c *Client) Redis() *redis.Client {
	return c.rdb
}
