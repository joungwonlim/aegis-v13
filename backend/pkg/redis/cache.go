package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Cache provides typed caching utilities
// ⭐ SSOT: 캐시 헬퍼는 여기서만
type Cache struct {
	client *Client
	prefix string
}

// NewCache creates a new cache helper
func NewCache(client *Client, prefix string) *Cache {
	return &Cache{
		client: client,
		prefix: prefix,
	}
}

// Get retrieves a cached value
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if !c.client.Enabled() {
		return false, nil
	}

	fullKey := fmt.Sprintf("%s:cache:%s", c.prefix, key)
	data, err := c.client.Redis().Get(ctx, fullKey).Bytes()
	if err != nil {
		// Key not found is not an error
		return false, nil
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("cache unmarshal failed: %w", err)
	}

	return true, nil
}

// Set stores a value in cache with TTL
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.client.Enabled() {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal failed: %w", err)
	}

	fullKey := fmt.Sprintf("%s:cache:%s", c.prefix, key)
	return c.client.Redis().Set(ctx, fullKey, data, ttl).Err()
}

// Delete removes a cached value
func (c *Cache) Delete(ctx context.Context, key string) error {
	if !c.client.Enabled() {
		return nil
	}

	fullKey := fmt.Sprintf("%s:cache:%s", c.prefix, key)
	return c.client.Redis().Del(ctx, fullKey).Err()
}

// GetOrSet retrieves from cache or calls fn to populate it
func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fn func() (interface{}, error)) error {
	// Try cache first
	found, err := c.Get(ctx, key, dest)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	// Cache miss - call function
	value, err := fn()
	if err != nil {
		return err
	}

	// Store in cache
	if err := c.Set(ctx, key, value, ttl); err != nil {
		// Log but don't fail
		return nil
	}

	// Unmarshal into dest
	data, _ := json.Marshal(value)
	return json.Unmarshal(data, dest)
}

// Predefined TTLs
const (
	TTLShort  = 1 * time.Minute   // 실시간 시세
	TTLMedium = 10 * time.Minute  // 종목 정보
	TTLLong   = 1 * time.Hour     // 마스터 데이터
	TTLDaily  = 24 * time.Hour    // 일별 데이터
)

// Common cache key generators
func StockInfoKey(code string) string {
	return fmt.Sprintf("stock:info:%s", code)
}

func PriceKey(code string, date string) string {
	return fmt.Sprintf("price:%s:%s", code, date)
}

func FinancialKey(code string, year int, quarter int) string {
	return fmt.Sprintf("financial:%s:%d:Q%d", code, year, quarter)
}
