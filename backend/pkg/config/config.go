package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
// ⭐ SSOT: 모든 환경변수는 여기서만 읽음
type Config struct {
	// Server
	Port string
	Env  string // development, staging, production

	// Database
	Database DatabaseConfig

	// Redis
	Redis RedisConfig

	// External APIs
	KIS   KISConfig
	DART  DARTConfig
	Naver NaverConfig

	// Logging
	LogLevel  string
	LogFormat string

	// Monitoring
	MetricsEnabled bool
	MetricsPort    string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Enabled  bool
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	URL      string

	// Connection Pool
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// KISConfig holds KIS (한국투자증권) API configuration
type KISConfig struct {
	AppKey    string
	AppSecret string
	AccountNo string
	BaseURL   string
	IsVirtual bool   // 모의투자 여부
	HtsID     string // HTS ID (체결통보 구독용)
}

// DARTConfig holds DART (전자공시) API configuration
type DARTConfig struct {
	APIKey  string
	BaseURL string
}

// NaverConfig holds Naver Finance configuration
type NaverConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
}

// Load reads configuration from environment variables
// ⭐ SSOT: 이 함수만 os.Getenv()를 호출함
func Load() (*Config, error) {
	// Try multiple paths for .env file
	loadEnvFile()

	cfg := &Config{
		// Server
		Port: getEnv("PORT", "8089"),
		Env:  getEnv("ENV", "development"),

		// Database
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "aegis_v13"),
			User:            getEnv("DB_USER", "aegis_v13"),
			Password:        getEnv("DB_PASSWORD", ""),
			URL:             getEnv("DATABASE_URL", ""),
			MaxConns:        getEnvAsInt("DB_MAX_CONNS", 25),
			MinConns:        getEnvAsInt("DB_MIN_CONNS", 5),
			MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", "1h"),
			MaxConnIdleTime: getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", "30m"),
		},

		// Redis
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			Enabled:  getEnvAsBool("REDIS_ENABLED", true),
		},

		// External APIs
		KIS: KISConfig{
			AppKey:    getEnv("KIS_APP_KEY", ""),
			AppSecret: getEnv("KIS_APP_SECRET", ""),
			AccountNo: getEnv("KIS_ACCOUNT_NO", ""),
			BaseURL:   getEnv("KIS_BASE_URL", "https://openapi.koreainvestment.com:9443"),
			IsVirtual: getEnvAsBool("KIS_IS_VIRTUAL", false),
			HtsID:     getEnv("KIS_HTS_ID", ""),
		},

		DART: DARTConfig{
			APIKey:  getEnv("DART_API_KEY", ""),
			BaseURL: getEnv("DART_BASE_URL", "https://opendart.fss.or.kr/api"),
		},

		Naver: NaverConfig{
			BaseURL:      getEnv("NAVER_BASE_URL", "https://finance.naver.com"),
			ClientID:     getEnv("NAVER_CLIENT_ID", ""),
			ClientSecret: getEnv("NAVER_CLIENT_SECRET", ""),
		},

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "debug"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		// Monitoring
		MetricsEnabled: getEnvAsBool("METRICS_ENABLED", true),
		MetricsPort:    getEnv("METRICS_PORT", "9090"),
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// validate checks if required configuration values are set
func (c *Config) validate() error {
	// Database URL is required
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	// Validate environment
	if c.Env != "development" && c.Env != "staging" && c.Env != "production" {
		return fmt.Errorf("ENV must be one of: development, staging, production")
	}

	return nil
}

// Helper functions (private, only used within this file)

// loadEnvFile tries to load .env from multiple locations
func loadEnvFile() {
	// Try paths in order of priority
	paths := []string{
		".env",           // Current directory
		"backend/.env",   // From project root
	}

	// Also try relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		paths = append(paths,
			filepath.Join(exeDir, ".env"),
			filepath.Join(exeDir, "..", ".env"),
			filepath.Join(exeDir, "..", "..", ".env"),
		)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			return
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		valueStr = defaultValue
	}

	duration, err := time.ParseDuration(valueStr)
	if err != nil {
		// Fallback to default
		duration, _ = time.ParseDuration(defaultValue)
	}

	return duration
}
