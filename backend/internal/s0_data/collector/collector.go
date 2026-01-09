package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wonny/aegis/v13/backend/internal/external/dart"
	"github.com/wonny/aegis/v13/backend/internal/external/krx"
	"github.com/wonny/aegis/v13/backend/internal/external/naver"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Collector orchestrates data collection from external sources
// ⭐ SSOT: 데이터 수집 오케스트레이션은 이 패키지에서만
type Collector struct {
	naverClient *naver.Client
	dartClient  *dart.Client
	krxClient   *krx.Client
	repo        *s0_data.Repository
	logger      *logger.Logger
}

// Config holds collector configuration
type Config struct {
	Workers int // Number of concurrent workers
}

// NewCollector creates a new Collector instance
func NewCollector(
	naverClient *naver.Client,
	dartClient *dart.Client,
	krxClient *krx.Client,
	repo *s0_data.Repository,
	log *logger.Logger,
) *Collector {
	return &Collector{
		naverClient: naverClient,
		dartClient:  dartClient,
		krxClient:   krxClient,
		repo:        repo,
		logger:      log.WithField("module", "collector"),
	}
}

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	StockCode    string
	PriceCount   int
	InvestorCount int
	Error        error
}

// FetchAllPrices fetches price data for all active stocks
func (c *Collector) FetchAllPrices(ctx context.Context, from, to time.Time, cfg Config) ([]FetchResult, error) {
	// 1. Get active stocks
	stocks, err := c.repo.GetActiveStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active stocks: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"stock_count": len(stocks),
		"from":        from.Format("2006-01-02"),
		"to":          to.Format("2006-01-02"),
		"workers":     cfg.Workers,
	}).Info("Starting price collection")

	// 2. Create worker pool
	results := make([]FetchResult, 0, len(stocks))
	resultCh := make(chan FetchResult, len(stocks))

	var wg sync.WaitGroup
	stockCh := make(chan s0_data.Stock, len(stocks))

	// Start workers
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.priceWorker(ctx, workerID, stockCh, resultCh, from, to)
		}(i)
	}

	// Send stocks to workers
	for _, stock := range stocks {
		stockCh <- stock
	}
	close(stockCh)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	successCount := 0
	failCount := 0
	for result := range resultCh {
		results = append(results, result)
		if result.Error != nil {
			failCount++
		} else {
			successCount++
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"success": successCount,
		"failed":  failCount,
		"total":   len(results),
	}).Info("Price collection completed")

	return results, nil
}

// priceWorker processes price fetching for stocks
func (c *Collector) priceWorker(ctx context.Context, workerID int, stockCh <-chan s0_data.Stock, resultCh chan<- FetchResult, from, to time.Time) {
	for stock := range stockCh {
		select {
		case <-ctx.Done():
			resultCh <- FetchResult{
				StockCode: stock.Code,
				Error:     ctx.Err(),
			}
			return
		default:
		}

		// Fetch prices
		prices, err := c.naverClient.FetchPrices(ctx, stock.Code, from, to)
		if err != nil {
			c.logger.WithError(err).WithFields(map[string]interface{}{
				"worker":     workerID,
				"stock_code": stock.Code,
			}).Error("Failed to fetch prices")
			resultCh <- FetchResult{
				StockCode: stock.Code,
				Error:     err,
			}
			continue
		}

		// Set stock code for each price
		for i := range prices {
			prices[i].StockCode = stock.Code
		}

		// Save to database
		if err := c.repo.SavePrices(ctx, prices); err != nil {
			c.logger.WithError(err).WithFields(map[string]interface{}{
				"worker":     workerID,
				"stock_code": stock.Code,
			}).Error("Failed to save prices")
			resultCh <- FetchResult{
				StockCode:  stock.Code,
				PriceCount: len(prices),
				Error:      err,
			}
			continue
		}

		c.logger.WithFields(map[string]interface{}{
			"worker":     workerID,
			"stock_code": stock.Code,
			"count":      len(prices),
		}).Debug("Fetched prices")

		resultCh <- FetchResult{
			StockCode:  stock.Code,
			PriceCount: len(prices),
		}
	}
}

// FetchAllInvestorFlow fetches investor flow data for all active stocks
func (c *Collector) FetchAllInvestorFlow(ctx context.Context, from, to time.Time, cfg Config) ([]FetchResult, error) {
	// 1. Get active stocks
	stocks, err := c.repo.GetActiveStocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active stocks: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"stock_count": len(stocks),
		"from":        from.Format("2006-01-02"),
		"to":          to.Format("2006-01-02"),
		"workers":     cfg.Workers,
	}).Info("Starting investor flow collection")

	// 2. Create worker pool
	results := make([]FetchResult, 0, len(stocks))
	resultCh := make(chan FetchResult, len(stocks))

	var wg sync.WaitGroup
	stockCh := make(chan s0_data.Stock, len(stocks))

	// Start workers
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.investorWorker(ctx, workerID, stockCh, resultCh, from, to)
		}(i)
	}

	// Send stocks to workers
	for _, stock := range stocks {
		stockCh <- stock
	}
	close(stockCh)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	successCount := 0
	failCount := 0
	for result := range resultCh {
		results = append(results, result)
		if result.Error != nil {
			failCount++
		} else {
			successCount++
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"success": successCount,
		"failed":  failCount,
		"total":   len(results),
	}).Info("Investor flow collection completed")

	return results, nil
}

// investorWorker processes investor flow fetching for stocks
func (c *Collector) investorWorker(ctx context.Context, workerID int, stockCh <-chan s0_data.Stock, resultCh chan<- FetchResult, from, to time.Time) {
	for stock := range stockCh {
		select {
		case <-ctx.Done():
			resultCh <- FetchResult{
				StockCode: stock.Code,
				Error:     ctx.Err(),
			}
			return
		default:
		}

		// Fetch investor flow
		flows, err := c.naverClient.FetchInvestorFlow(ctx, stock.Code, from, to)
		if err != nil {
			c.logger.WithError(err).WithFields(map[string]interface{}{
				"worker":     workerID,
				"stock_code": stock.Code,
			}).Error("Failed to fetch investor flow")
			resultCh <- FetchResult{
				StockCode: stock.Code,
				Error:     err,
			}
			continue
		}

		// Set stock code for each flow
		for i := range flows {
			flows[i].StockCode = stock.Code
		}

		// Save to database
		if err := c.repo.SaveInvestorFlow(ctx, flows); err != nil {
			c.logger.WithError(err).WithFields(map[string]interface{}{
				"worker":     workerID,
				"stock_code": stock.Code,
			}).Error("Failed to save investor flow")
			resultCh <- FetchResult{
				StockCode:     stock.Code,
				InvestorCount: len(flows),
				Error:         err,
			}
			continue
		}

		c.logger.WithFields(map[string]interface{}{
			"worker":     workerID,
			"stock_code": stock.Code,
			"count":      len(flows),
		}).Debug("Fetched investor flow")

		resultCh <- FetchResult{
			StockCode:     stock.Code,
			InvestorCount: len(flows),
		}
	}
}

// FetchAll fetches both prices and investor flow concurrently
func (c *Collector) FetchAll(ctx context.Context, from, to time.Time, cfg Config) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Fetch prices
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c.FetchAllPrices(ctx, from, to, cfg)
		if err != nil {
			errCh <- fmt.Errorf("fetch prices: %w", err)
		}
	}()

	// Fetch investor flow
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c.FetchAllInvestorFlow(ctx, from, to, cfg)
		if err != nil {
			errCh <- fmt.Errorf("fetch investor flow: %w", err)
		}
	}()

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("collection errors: %v", errs)
	}

	return nil
}

// FetchDisclosures fetches disclosure data from DART for all active stocks
// ⭐ SSOT: DART 공시 수집은 이 함수에서만
func (c *Collector) FetchDisclosures(ctx context.Context, from, to time.Time) error {
	c.logger.WithFields(map[string]interface{}{
		"from": from.Format("2006-01-02"),
		"to":   to.Format("2006-01-02"),
	}).Info("Starting disclosure collection")

	// Fetch disclosures for each page (DART API is paginated)
	allDisclosures := []dart.Disclosure{}
	page := 1
	maxPages := 10 // Limit to prevent infinite loops

	for page <= maxPages {
		disclosures, totalPages, err := c.dartClient.FetchDisclosuresForPage(ctx, from, to, page)
		if err != nil {
			c.logger.WithError(err).WithField("page", page).Error("Failed to fetch disclosures page")
			return fmt.Errorf("fetch disclosures page %d: %w", page, err)
		}

		if len(disclosures) == 0 {
			break
		}

		allDisclosures = append(allDisclosures, disclosures...)

		c.logger.WithFields(map[string]interface{}{
			"page":        page,
			"total_pages": totalPages,
			"count":       len(disclosures),
		}).Debug("Fetched disclosures page")

		if page >= totalPages {
			break
		}

		page++
	}

	// Save to database
	if len(allDisclosures) > 0 {
		if err := c.repo.SaveDisclosures(ctx, allDisclosures); err != nil {
			return fmt.Errorf("save disclosures: %w", err)
		}

		c.logger.WithField("count", len(allDisclosures)).Info("Saved disclosures")
	}

	return nil
}

// FetchMarketTrends fetches market trend data from KRX (via Naver)
// ⭐ SSOT: KRX 시장 지표 수집은 이 함수에서만
func (c *Collector) FetchMarketTrends(ctx context.Context) error {
	c.logger.Info("Starting market trend collection")

	// Fetch KOSPI trend
	kospiTrend, err := c.krxClient.FetchMarketTrend(ctx, "KOSPI")
	if err != nil {
		return fmt.Errorf("fetch KOSPI trend: %w", err)
	}

	if kospiTrend != nil {
		if err := c.repo.SaveMarketTrend(ctx, "KOSPI", kospiTrend); err != nil {
			return fmt.Errorf("save KOSPI trend: %w", err)
		}

		c.logger.WithFields(map[string]interface{}{
			"index":       "KOSPI",
			"foreign_net": kospiTrend.ForeignNet,
			"inst_net":    kospiTrend.InstitutionNet,
		}).Info("Saved KOSPI trend")
	}

	// Fetch KOSDAQ trend
	kosdaqTrend, err := c.krxClient.FetchMarketTrend(ctx, "KOSDAQ")
	if err != nil {
		return fmt.Errorf("fetch KOSDAQ trend: %w", err)
	}

	if kosdaqTrend != nil {
		if err := c.repo.SaveMarketTrend(ctx, "KOSDAQ", kosdaqTrend); err != nil {
			return fmt.Errorf("save KOSDAQ trend: %w", err)
		}

		c.logger.WithFields(map[string]interface{}{
			"index":       "KOSDAQ",
			"foreign_net": kosdaqTrend.ForeignNet,
			"inst_net":    kosdaqTrend.InstitutionNet,
		}).Info("Saved KOSDAQ trend")
	}

	return nil
}

// FetchMarketCaps fetches market capitalization for all stocks
// ⭐ SSOT: 시가총액 수집은 이 함수에서만
func (c *Collector) FetchMarketCaps(ctx context.Context) error {
	c.logger.Info("Starting market cap collection")

	allCaps := []naver.MarketCapData{}

	// Fetch KOSPI market caps
	kospiCaps, err := c.naverClient.FetchAllMarketCaps(ctx, "KOSPI")
	if err != nil {
		return fmt.Errorf("fetch KOSPI market caps: %w", err)
	}
	allCaps = append(allCaps, kospiCaps...)

	c.logger.WithField("count", len(kospiCaps)).Info("Fetched KOSPI market caps")

	// Fetch KOSDAQ market caps
	kosdaqCaps, err := c.naverClient.FetchAllMarketCaps(ctx, "KOSDAQ")
	if err != nil {
		return fmt.Errorf("fetch KOSDAQ market caps: %w", err)
	}
	allCaps = append(allCaps, kosdaqCaps...)

	c.logger.WithField("count", len(kosdaqCaps)).Info("Fetched KOSDAQ market caps")

	// Save to database
	if len(allCaps) > 0 {
		if err := c.repo.SaveMarketCaps(ctx, allCaps); err != nil {
			return fmt.Errorf("save market caps: %w", err)
		}

		c.logger.WithField("total_count", len(allCaps)).Info("Saved market caps")
	}

	return nil
}
