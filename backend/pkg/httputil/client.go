package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
	"github.com/wonny/aegis/v13/backend/pkg/redis"
)

// Client is an HTTP client wrapper with retry logic and logging
// ⭐ SSOT: 모든 HTTP 요청은 이 클라이언트를 통해서만 수행
type Client struct {
	httpClient   *http.Client
	logger       *logger.Logger
	retryConfig  RetryConfig
	rateLimiter  *redis.RateLimiter
	rateLimitCfg *redis.RateLimitConfig
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Enabled      bool
}

// New creates a new HTTP client from config
// ⭐ SSOT: http.Client 인스턴스는 여기서만 생성
func New(cfg *config.Config, log *logger.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Default timeout
		},
		logger: log,
		retryConfig: RetryConfig{
			MaxRetries:   3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     10 * time.Second,
			Enabled:      true,
		},
	}
}

// NewWithTimeout creates a client with custom timeout
func NewWithTimeout(cfg *config.Config, log *logger.Logger, timeout time.Duration) *Client {
	client := New(cfg, log)
	client.httpClient.Timeout = timeout
	return client
}

// WithRetry configures retry behavior
func (c *Client) WithRetry(maxRetries int, initialDelay time.Duration) *Client {
	c.retryConfig.MaxRetries = maxRetries
	c.retryConfig.InitialDelay = initialDelay
	c.retryConfig.Enabled = true
	return c
}

// DisableRetry disables automatic retry
func (c *Client) DisableRetry() *Client {
	c.retryConfig.Enabled = false
	return c
}

// WithRateLimiter sets the rate limiter for this client
func (c *Client) WithRateLimiter(limiter *redis.RateLimiter, cfg redis.RateLimitConfig) *Client {
	c.rateLimiter = limiter
	c.rateLimitCfg = &cfg
	return c
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	return c.do(req)
}

// Post performs a POST request with body
func (c *Client) Post(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	return c.do(req)
}

// PostJSON performs a POST request with JSON body
func (c *Client) PostJSON(ctx context.Context, url string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.Post(ctx, url, "application/json", bytes.NewReader(jsonData))
}

// PostForm performs a POST request with form data
func (c *Client) PostForm(ctx context.Context, targetURL string, formData url.Values) (*http.Response, error) {
	return c.Post(ctx, targetURL, "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
}

// do executes the request with retry logic and logging
func (c *Client) do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	startTime := time.Now()
	url := req.URL.String()
	method := req.Method

	// Check rate limit
	if c.rateLimiter != nil && c.rateLimitCfg != nil {
		if err := c.rateLimiter.Wait(req.Context(), *c.rateLimitCfg); err != nil {
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	// Log request
	c.logger.WithFields(map[string]interface{}{
		"method": method,
		"url":    url,
	}).Debug("HTTP request started")

	// Execute with retry
	if c.retryConfig.Enabled {
		resp, err = c.doWithRetry(req)
	} else {
		resp, err = c.httpClient.Do(req)
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Log response
	if err != nil {
		c.logger.WithFields(map[string]interface{}{
			"method":   method,
			"url":      url,
			"duration": duration,
			"error":    err.Error(),
		}).Error("HTTP request failed")
		return nil, err
	}

	c.logger.WithFields(map[string]interface{}{
		"method":      method,
		"url":         url,
		"status_code": resp.StatusCode,
		"duration":    duration,
	}).Debug("HTTP request completed")

	return resp, nil
}

// doWithRetry executes the request with exponential backoff retry
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	delay := c.retryConfig.InitialDelay

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Execute request
		resp, err = c.httpClient.Do(req)

		// Success
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// Last attempt - return error
		if attempt == c.retryConfig.MaxRetries {
			break
		}

		// Log retry
		c.logger.WithFields(map[string]interface{}{
			"attempt": attempt + 1,
			"delay":   delay,
			"url":     req.URL.String(),
		}).Warn("Retrying HTTP request")

		// Wait before retry
		time.Sleep(delay)

		// Exponential backoff
		delay *= 2
		if delay > c.retryConfig.MaxDelay {
			delay = c.retryConfig.MaxDelay
		}
	}

	return resp, err
}

// IsRetryableError checks if an error should be retried
func IsRetryableError(statusCode int) bool {
	// Retry on 5xx server errors and 429 Too Many Requests
	return statusCode >= 500 || statusCode == 429
}
