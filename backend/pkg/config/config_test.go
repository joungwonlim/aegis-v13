package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Set required environment variables
	os.Setenv("DATABASE_URL", "postgresql://test:test@localhost:5432/testdb")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check defaults
	if cfg.Port != "8080" {
		t.Errorf("Expected Port to be 8080, got %s", cfg.Port)
	}

	if cfg.Env != "development" {
		t.Errorf("Expected Env to be development, got %s", cfg.Env)
	}

	if cfg.Database.MaxConns != 25 {
		t.Errorf("Expected DB MaxConns to be 25, got %d", cfg.Database.MaxConns)
	}
}

func TestLoadWithCustomValues(t *testing.T) {
	// Set custom environment variables
	os.Setenv("PORT", "9000")
	os.Setenv("ENV", "production")
	os.Setenv("DATABASE_URL", "postgresql://test:test@localhost:5432/testdb")
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("LOG_LEVEL", "info")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENV")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("DB_MAX_CONNS")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Port != "9000" {
		t.Errorf("Expected Port to be 9000, got %s", cfg.Port)
	}

	if cfg.Env != "production" {
		t.Errorf("Expected Env to be production, got %s", cfg.Env)
	}

	if cfg.Database.MaxConns != 50 {
		t.Errorf("Expected DB MaxConns to be 50, got %d", cfg.Database.MaxConns)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be info, got %s", cfg.LogLevel)
	}
}

func TestValidateMissingDatabaseURL(t *testing.T) {
	// Unset DATABASE_URL
	os.Unsetenv("DATABASE_URL")

	_, err := Load()
	if err == nil {
		t.Error("Expected error when DATABASE_URL is missing, got nil")
	}
}

func TestValidateInvalidEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgresql://test:test@localhost:5432/testdb")
	os.Setenv("ENV", "invalid")

	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("ENV")
	}()

	_, err := Load()
	if err == nil {
		t.Error("Expected error when ENV is invalid, got nil")
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	os.Setenv("TEST_DURATION", "2h")
	defer os.Unsetenv("TEST_DURATION")

	duration := getEnvAsDuration("TEST_DURATION", "1h")
	expected := 2 * time.Hour

	if duration != expected {
		t.Errorf("Expected duration to be %v, got %v", expected, duration)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	os.Setenv("TEST_INT", "100")
	defer os.Unsetenv("TEST_INT")

	value := getEnvAsInt("TEST_INT", 50)
	if value != 100 {
		t.Errorf("Expected value to be 100, got %d", value)
	}
}

func TestGetEnvAsBool(t *testing.T) {
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	value := getEnvAsBool("TEST_BOOL", false)
	if value != true {
		t.Errorf("Expected value to be true, got %v", value)
	}
}
