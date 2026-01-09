package main

import (
	"os"

	"github.com/wonny/aegis/v13/backend/cmd/quant/commands"
)

// main is the entry point for the Aegis CLI
// ⭐ 통합 CLI 진입점: go run ./cmd/quant [command]
func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
