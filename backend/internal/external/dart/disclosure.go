package dart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// FetchDisclosures fetches disclosures for a specific corp_code within date range
// ⭐ SSOT: DART 공시 데이터 호출은 이 함수에서만
func (c *Client) FetchDisclosures(ctx context.Context, corpCode string, from, to time.Time) ([]Disclosure, error) {
	bgn := from.Format("20060102")
	end := to.Format("20060102")

	return c.fetchByCorpCodeWithRetry(ctx, corpCode, bgn, end)
}

// FetchDisclosuresForPage fetches a single page of disclosures for all companies
func (c *Client) FetchDisclosuresForPage(ctx context.Context, from, to time.Time, page int) ([]Disclosure, int, error) {
	bgn := from.Format("20060102")
	end := to.Format("20060102")

	return c.fetchPage(ctx, bgn, end, page)
}

// fetchByCorpCode fetches disclosures for a specific corp_code
func (c *Client) fetchByCorpCode(ctx context.Context, corpCode, bgn, end string) ([]Disclosure, error) {
	url := fmt.Sprintf(
		"%s/api/list.json?crtfc_key=%s&corp_code=%s&bgn_de=%s&end_de=%s&page_count=100",
		c.baseURL, c.apiKey, corpCode, bgn, end,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result DisclosureResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Status codes:
	// 000 = success
	// 013 = no data (ok)
	// others = error
	if result.Status != "000" {
		if result.Status == "013" {
			return nil, nil // No data is not an error
		}
		return nil, fmt.Errorf("API error: %s - %s", result.Status, result.Message)
	}

	return result.Disclosures, nil
}

// fetchByCorpCodeWithRetry fetches disclosures with exponential backoff retry
func (c *Client) fetchByCorpCodeWithRetry(ctx context.Context, corpCode, bgn, end string) ([]Disclosure, error) {
	const maxRetries = 3
	const initialBackoff = 500 * time.Millisecond
	const maxBackoff = 5 * time.Second

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		disclosures, err := c.fetchByCorpCode(ctx, corpCode, bgn, end)

		// Success
		if err == nil {
			return disclosures, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			// Non-retryable error (e.g., auth failure)
			return nil, err
		}

		// Last attempt failed - don't retry
		if attempt == maxRetries-1 {
			break
		}

		// Wait before retry with exponential backoff
		c.logger.WithError(err).WithFields(map[string]interface{}{
			"attempt":   attempt + 1,
			"max":       maxRetries - 1,
			"corp_code": corpCode,
			"backoff":   backoff,
		}).Debug("Retrying DART API call")

		select {
		case <-time.After(backoff):
			// Continue to next retry
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Exponential backoff for next retry
		backoff = backoff * 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return nil, fmt.Errorf("max retries exceeded for corp_code %s: %w", corpCode, lastErr)
}

// fetchPage fetches a single page of disclosures for all companies
func (c *Client) fetchPage(ctx context.Context, bgn, end string, page int) ([]Disclosure, int, error) {
	url := fmt.Sprintf(
		"%s/api/list.json?crtfc_key=%s&bgn_de=%s&end_de=%s&page_no=%d&page_count=100",
		c.baseURL, c.apiKey, bgn, end, page,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result DisclosureResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decode response: %w", err)
	}

	if result.Status != "000" {
		if result.Status == "013" {
			return nil, 0, nil // No data
		}
		return nil, 0, fmt.Errorf("API error: %s - %s", result.Status, result.Message)
	}

	return result.Disclosures, result.TotalPage, nil
}

// isRetryableError checks if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network-related errors that are retryable
	retryablePatterns := []string{
		"connection reset by peer",
		"eof",
		"connection refused",
		"network unreachable",
		"timeout",
		"i/o timeout",
		"connect: operation timed out",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}
