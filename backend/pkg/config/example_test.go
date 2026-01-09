package config_test

import (
	"fmt"

	"github.com/wonny/aegis/v13/backend/pkg/config"
)

// Example demonstrates how to use the config package
func Example() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// Access configuration values
	fmt.Printf("Server running on port: %s\n", cfg.Port)
	fmt.Printf("Environment: %s\n", cfg.Env)
	fmt.Printf("Database: %s\n", cfg.Database.Name)
	fmt.Printf("DB Max Connections: %d\n", cfg.Database.MaxConns)
}
