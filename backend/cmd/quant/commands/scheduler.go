package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/wonny/aegis/v13/backend/internal/external/dart"
	"github.com/wonny/aegis/v13/backend/internal/external/krx"
	"github.com/wonny/aegis/v13/backend/internal/external/naver"
	"github.com/wonny/aegis/v13/backend/internal/realtime/cache"
	"github.com/wonny/aegis/v13/backend/internal/s0_data"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/collector"
	"github.com/wonny/aegis/v13/backend/internal/s0_data/quality"
	"github.com/wonny/aegis/v13/backend/internal/s1_universe"
	"github.com/wonny/aegis/v13/backend/internal/scheduler"
	"github.com/wonny/aegis/v13/backend/internal/scheduler/jobs"
	"github.com/wonny/aegis/v13/backend/pkg/config"
	"github.com/wonny/aegis/v13/backend/pkg/database"
	"github.com/wonny/aegis/v13/backend/pkg/httputil"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// schedulerCmd represents the scheduler command
var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "스케줄러 관리",
	Long: `스케줄러를 시작하거나 작업을 관리합니다.

이 명령어는:
- 스케줄러 데몬 시작
- 등록된 작업 조회
- 작업 실행 이력 조회

Subcommands:
  start   - 스케줄러 시작
  list    - 등록된 작업 목록
  run     - 특정 작업 즉시 실행
  status  - 작업 실행 상태 조회

Example:
  go run ./cmd/quant scheduler start
  go run ./cmd/quant scheduler list
  go run ./cmd/quant scheduler run data_collection`,
}

var (
	schedulerStartCmd = &cobra.Command{
		Use:   "start",
		Short: "스케줄러 시작",
		Long: `스케줄러를 시작하고 등록된 모든 작업을 스케줄합니다.

등록되는 작업:
- data_collection: 매일 오후 4시 (전체 데이터 수집)
- price_collection: 평일 9-15시 매시간 (가격 데이터)
- investor_flow: 매일 오후 5시 (투자자 수급)
- disclosure_collection: 6시간마다 (공시 데이터)
- universe_generation: 매일 오후 6시 (Universe 생성)
- cache_cleanup: 5분마다 (캐시 정리)

스케줄러는 Ctrl+C로 종료할 수 있습니다.`,
		RunE: runScheduler,
	}

	schedulerListCmd = &cobra.Command{
		Use:   "list",
		Short: "등록된 작업 목록",
		RunE:  listJobs,
	}

	schedulerRunCmd = &cobra.Command{
		Use:   "run [job_name]",
		Short: "특정 작업 즉시 실행",
		Args:  cobra.ExactArgs(1),
		RunE:  runJob,
	}

	schedulerStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "작업 실행 상태 조회",
		RunE:  showStatus,
	}
)

func init() {
	rootCmd.AddCommand(schedulerCmd)
	schedulerCmd.AddCommand(schedulerStartCmd)
	schedulerCmd.AddCommand(schedulerListCmd)
	schedulerCmd.AddCommand(schedulerRunCmd)
	schedulerCmd.AddCommand(schedulerStatusCmd)
}

func runScheduler(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Aegis v13 Scheduler ===\n")

	// Initialize dependencies
	sched, err := initScheduler()
	if err != nil {
		return fmt.Errorf("init scheduler: %w", err)
	}

	// Start scheduler
	sched.Start()

	fmt.Println("\n✅ Scheduler started successfully")
	fmt.Println("\nRegistered jobs:")
	for _, jobName := range sched.GetAllJobs() {
		fmt.Printf("  - %s\n", jobName)
	}
	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down scheduler...")
	sched.Stop()
	fmt.Println("Scheduler stopped")

	return nil
}

func listJobs(cmd *cobra.Command, args []string) error {
	sched, err := initScheduler()
	if err != nil {
		return fmt.Errorf("init scheduler: %w", err)
	}

	jobs := sched.GetAllJobs()

	fmt.Println("Registered jobs:")
	for _, jobName := range jobs {
		fmt.Printf("  - %s\n", jobName)
	}

	return nil
}

func runJob(cmd *cobra.Command, args []string) error {
	jobName := args[0]

	fmt.Printf("Running job: %s\n", jobName)

	sched, err := initScheduler()
	if err != nil {
		return fmt.Errorf("init scheduler: %w", err)
	}

	if err := sched.RunJob(jobName); err != nil {
		return fmt.Errorf("run job: %w", err)
	}

	fmt.Println("Job started (running in background)")
	return nil
}

func showStatus(cmd *cobra.Command, args []string) error {
	sched, err := initScheduler()
	if err != nil {
		return fmt.Errorf("init scheduler: %w", err)
	}

	stats := sched.GetJobStats()

	fmt.Println("Job Statistics:")
	fmt.Println()

	for jobName, stat := range stats {
		fmt.Printf("📊 %s\n", jobName)
		fmt.Printf("   Schedule: %s\n", stat.Schedule)
		fmt.Printf("   Total Runs: %d\n", stat.TotalRuns)
		fmt.Printf("   Success: %d (%.1f%%)\n", stat.SuccessCount, stat.SuccessRate*100)
		fmt.Printf("   Failures: %d\n", stat.FailureCount)

		if stat.LastRun != nil {
			fmt.Printf("   Last Run: %s\n", stat.LastRun.Format("2006-01-02 15:04:05"))
		}

		if stat.LastSuccess != nil {
			fmt.Printf("   Last Success: %s\n", stat.LastSuccess.Format("2006-01-02 15:04:05"))
		}

		if stat.LastFailure != nil {
			fmt.Printf("   Last Failure: %s\n", stat.LastFailure.Format("2006-01-02 15:04:05"))
		}

		fmt.Println()
	}

	return nil
}

func initScheduler() (*scheduler.Scheduler, error) {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize logger
	log := logger.New(cfg)

	// 3. Connect to database
	db, err := database.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// 4. Create HTTP client
	httpClient := httputil.New(cfg, log)

	// 5. Create external API clients
	naverClient := naver.NewClient(httpClient, log)
	dartClient := dart.NewClient(cfg.DART.APIKey, log)
	krxClient := krx.NewClient(httpClient, log)

	// 6. Create repositories
	dataRepo := s0_data.NewRepository(db.Pool)

	// 7. Create collector
	col := collector.NewCollector(naverClient, dartClient, krxClient, dataRepo, log)

	// 8. Create quality gate
	qualityConfig := quality.Config{
		MinPriceCoverage:      1.0,
		MinVolumeCoverage:     1.0,
		MinMarketCapCoverage:  0.95,
		MinFinancialCoverage:  0.80,
		MinInvestorCoverage:   0.80,
		MinDisclosureCoverage: 0.70,
	}
	qualityGate := quality.NewQualityGate(db.Pool, qualityConfig)

	// 9. Create universe builder
	universeConfig := s1_universe.Config{
		MinMarketCap:   10_000_000_000, // 100억 원
		MinVolume:      100_000_000,    // 1억 원
		MinListingDays: 20,             // 20일
		ExcludeAdmin:   true,
		ExcludeHalt:    true,
		ExcludeSPAC:    true,
	}
	universeBuilder := s1_universe.NewBuilder(db.Pool, universeConfig)

	// 10. Create price cache
	priceCache := cache.NewPriceCache(60*time.Second, log)

	// 11. Create scheduler
	sched := scheduler.New(log)

	// 12. Register jobs
	sched.AddJob(jobs.NewDataCollectionJob(col, cfg, log))
	sched.AddJob(jobs.NewPriceCollectionJob(col, cfg, log))
	sched.AddJob(jobs.NewInvestorFlowJob(col, cfg, log))
	sched.AddJob(jobs.NewDisclosureJob(col, log))
	sched.AddJob(jobs.NewUniverseJob(universeBuilder, qualityGate, log))
	sched.AddJob(jobs.NewCacheCleanupJob(priceCache, log))

	return sched, nil
}
