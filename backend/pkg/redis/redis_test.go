package redis

import (
	"testing"

	"github.com/wonny/aegis/v13/backend/pkg/config"
)

func TestNewClient_Disabled(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Enabled: false,
		},
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.Enabled() {
		t.Error("Expected client to be disabled")
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Enabled: false,
		},
	}

	client, _ := New(cfg)
	limiter := NewRateLimiter(client, "test")

	// When Redis is disabled, all requests should be allowed
	allowed, remaining, err := limiter.Allow(nil, KISRateLimit)
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if !allowed {
		t.Error("Expected request to be allowed when Redis disabled")
	}
	if remaining != KISRateLimit.Limit {
		t.Errorf("Expected remaining = %d, got %d", KISRateLimit.Limit, remaining)
	}
}

func TestCache_Disabled(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Enabled: false,
		},
	}

	client, _ := New(cfg)
	cache := NewCache(client, "test")

	// When Redis is disabled, cache operations should be no-ops
	var result string
	found, err := cache.Get(nil, "key", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if found {
		t.Error("Expected cache miss when Redis disabled")
	}
}

func TestCacheKeys(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{
			name:     "StockInfoKey",
			fn:       func() string { return StockInfoKey("005930") },
			expected: "stock:info:005930",
		},
		{
			name:     "PriceKey",
			fn:       func() string { return PriceKey("005930", "2024-01-15") },
			expected: "price:005930:2024-01-15",
		},
		{
			name:     "FinancialKey",
			fn:       func() string { return FinancialKey("005930", 2024, 1) },
			expected: "financial:005930:2024:Q1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
