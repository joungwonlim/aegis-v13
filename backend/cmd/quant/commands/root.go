package commands

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	configFile string
	env        string
	verbose    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "quant",
	Short: "Aegis v13 - 기관급 퀀트 트레이딩 시스템",
	Long: `Aegis v13 Unified CLI

Go BFF 기반 퀀트 트레이딩 시스템.
7단계 파이프라인으로 데이터 수집부터 주문 실행까지.

Usage:
  go run ./cmd/quant [command]

Examples:
  go run ./cmd/quant api
  go run ./cmd/quant fetcher collect all
  go run ./cmd/quant test-db
  go run ./cmd/quant test-logger`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .env)")
	rootCmd.PersistentFlags().StringVar(&env, "env", "development", "environment (development|staging|production)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
