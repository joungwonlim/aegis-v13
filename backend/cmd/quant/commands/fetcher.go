package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// fetcherCmd represents the fetcher command
var fetcherCmd = &cobra.Command{
	Use:   "fetcher",
	Short: "데이터 수집 도구",
	Long: `외부 API (KIS, DART, Naver)에서 데이터를 수집합니다.

이 명령어는:
- KIS API에서 시세/체결 데이터 수집
- DART에서 공시 데이터 수집
- Naver Finance에서 보조 데이터 수집

Example:
  go run ./cmd/quant fetcher collect --source kis
  go run ./cmd/quant fetcher collect --source dart
  go run ./cmd/quant fetcher collect all`,
}

// fetcherCollectCmd represents the collect subcommand
var fetcherCollectCmd = &cobra.Command{
	Use:   "collect [source]",
	Short: "데이터 수집 실행",
	Long: `지정된 소스에서 데이터를 수집합니다.

소스:
  kis    - 한국투자증권 API (시세, 체결)
  dart   - DART 공시 데이터
  naver  - Naver Finance
  all    - 모든 소스

Example:
  go run ./cmd/quant fetcher collect kis
  go run ./cmd/quant fetcher collect dart
  go run ./cmd/quant fetcher collect all`,
	Args: cobra.ExactArgs(1),
	RunE: runFetcherCollect,
}

var (
	// Fetcher flags
	fetcherSource string
	fetcherAsync  bool
)

func init() {
	rootCmd.AddCommand(fetcherCmd)
	fetcherCmd.AddCommand(fetcherCollectCmd)

	// Flags
	fetcherCollectCmd.Flags().BoolVar(&fetcherAsync, "async", false, "비동기 수집 (큐에 작업 추가)")
}

func runFetcherCollect(cmd *cobra.Command, args []string) error {
	source := args[0]

	fmt.Printf("=== Aegis v13 Data Fetcher ===\n\n")
	fmt.Printf("Source: %s\n", source)
	fmt.Printf("Mode: %s\n\n", getMode(fetcherAsync))

	switch source {
	case "kis":
		return collectKIS()
	case "dart":
		return collectDART()
	case "naver":
		return collectNaver()
	case "all":
		return collectAll()
	default:
		return fmt.Errorf("unknown source: %s (valid: kis, dart, naver, all)", source)
	}
}

func collectKIS() error {
	fmt.Println()
	PrintSeparator()
	fmt.Println("📊 KIS 데이터 수집 시작...")
	PrintSeparator()

	items := []string{
		"실시간 시세 데이터",
		"체결 데이터",
		"호가 데이터",
	}
	PrintList(items)
	fmt.Println()
	PrintWarning("구현 필요: internal/external/kis/")
	return nil
}

func collectDART() error {
	fmt.Println()
	PrintSeparator()
	fmt.Println("📄 DART 공시 데이터 수집 시작...")
	PrintSeparator()

	items := []string{
		"정기보고서",
		"주요사항보고",
		"재무제표",
	}
	PrintList(items)
	fmt.Println()
	PrintWarning("구현 필요: internal/external/dart/")
	return nil
}

func collectNaver() error {
	fmt.Println()
	PrintSeparator()
	fmt.Println("🔍 Naver Finance 데이터 수집 시작...")
	PrintSeparator()

	items := []string{
		"종목 정보",
		"투자자별 매매 동향",
		"신용/대차 잔고",
	}
	PrintList(items)
	fmt.Println()
	PrintWarning("구현 필요: internal/external/naver/")
	return nil
}

func collectAll() error {
	fmt.Println("🚀 전체 소스 데이터 수집 시작...\n")

	if err := collectKIS(); err != nil {
		return err
	}

	if err := collectDART(); err != nil {
		return err
	}

	if err := collectNaver(); err != nil {
		return err
	}

	fmt.Println("✅ 전체 데이터 수집 완료!")
	return nil
}

func getMode(async bool) string {
	if async {
		return "Async (Queue)"
	}
	return "Sync (Direct)"
}
