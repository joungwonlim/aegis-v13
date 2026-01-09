package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/wonny/aegis/v13/backend/pkg/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.Config
		wantLevel zerolog.Level
	}{
		{
			name: "debug level",
			cfg: &config.Config{
				Env:       "development",
				LogLevel:  "debug",
				LogFormat: "json",
			},
			wantLevel: zerolog.DebugLevel,
		},
		{
			name: "info level",
			cfg: &config.Config{
				Env:       "production",
				LogLevel:  "info",
				LogFormat: "json",
			},
			wantLevel: zerolog.InfoLevel,
		},
		{
			name: "warn level",
			cfg: &config.Config{
				Env:       "staging",
				LogLevel:  "warn",
				LogFormat: "json",
			},
			wantLevel: zerolog.WarnLevel,
		},
		{
			name: "error level",
			cfg: &config.Config{
				Env:       "production",
				LogLevel:  "error",
				LogFormat: "json",
			},
			wantLevel: zerolog.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.cfg)
			if logger == nil {
				t.Fatal("Expected logger to be created")
			}

			// Verify global level is set
			if zerolog.GlobalLevel() != tt.wantLevel {
				t.Errorf("Expected global level %v, got %v", tt.wantLevel, zerolog.GlobalLevel())
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  zerolog.Level
	}{
		{"debug", zerolog.DebugLevel},
		{"DEBUG", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"INFO", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"warning", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"fatal", zerolog.FatalLevel},
		{"panic", zerolog.PanicLevel},
		{"invalid", zerolog.InfoLevel}, // Default
		{"", zerolog.InfoLevel},        // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	// Set global level to debug to capture all logs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Create logger with custom output
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	tests := []struct {
		name     string
		logFunc  func()
		wantMsg  string
		wantLevel string
	}{
		{
			name: "debug",
			logFunc: func() {
				logger.Debug("debug message")
			},
			wantMsg:  "debug message",
			wantLevel: "debug",
		},
		{
			name: "info",
			logFunc: func() {
				logger.Info("info message")
			},
			wantMsg:  "info message",
			wantLevel: "info",
		},
		{
			name: "warn",
			logFunc: func() {
				logger.Warn("warn message")
			},
			wantMsg:  "warn message",
			wantLevel: "warn",
		},
		{
			name: "error",
			logFunc: func() {
				logger.Error("error message")
			},
			wantMsg:  "error message",
			wantLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}

			if logEntry["level"] != tt.wantLevel {
				t.Errorf("Expected level %q, got %q", tt.wantLevel, logEntry["level"])
			}

			if logEntry["message"] != tt.wantMsg {
				t.Errorf("Expected message %q, got %q", tt.wantMsg, logEntry["message"])
			}
		})
	}
}

func TestLoggerFormattedMethods(t *testing.T) {
	var buf bytes.Buffer

	// Set global level to debug to capture all logs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	tests := []struct {
		name     string
		logFunc  func()
		wantMsg  string
		wantLevel string
	}{
		{
			name: "debugf",
			logFunc: func() {
				logger.Debugf("user: %s, age: %d", "alice", 30)
			},
			wantMsg:  "user: alice, age: 30",
			wantLevel: "debug",
		},
		{
			name: "infof",
			logFunc: func() {
				logger.Infof("count: %d", 42)
			},
			wantMsg:  "count: 42",
			wantLevel: "info",
		},
		{
			name: "warnf",
			logFunc: func() {
				logger.Warnf("retry attempt: %d", 3)
			},
			wantMsg:  "retry attempt: 3",
			wantLevel: "warn",
		},
		{
			name: "errorf",
			logFunc: func() {
				logger.Errorf("failed to connect: %s", "timeout")
			},
			wantMsg:  "failed to connect: timeout",
			wantLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log output: %v", err)
			}

			if logEntry["level"] != tt.wantLevel {
				t.Errorf("Expected level %q, got %q", tt.wantLevel, logEntry["level"])
			}

			if logEntry["message"] != tt.wantMsg {
				t.Errorf("Expected message %q, got %q", tt.wantMsg, logEntry["message"])
			}
		})
	}
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer

	// Set global level to debug to capture all logs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	enrichedLogger := logger.WithField("user_id", "12345")
	enrichedLogger.Info("user action")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["user_id"] != "12345" {
		t.Errorf("Expected user_id to be 12345, got %v", logEntry["user_id"])
	}

	if logEntry["message"] != "user action" {
		t.Errorf("Expected message 'user action', got %v", logEntry["message"])
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer

	// Set global level to debug to capture all logs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	fields := map[string]interface{}{
		"user_id":  "12345",
		"stock_id": "005930",
		"price":    72300,
	}

	enrichedLogger := logger.WithFields(fields)
	enrichedLogger.Info("trade executed")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["user_id"] != "12345" {
		t.Errorf("Expected user_id to be 12345, got %v", logEntry["user_id"])
	}

	if logEntry["stock_id"] != "005930" {
		t.Errorf("Expected stock_id to be 005930, got %v", logEntry["stock_id"])
	}

	if logEntry["price"] != float64(72300) {
		t.Errorf("Expected price to be 72300, got %v", logEntry["price"])
	}
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	testErr := errors.New("database connection failed")
	enrichedLogger := logger.WithError(testErr)
	enrichedLogger.Error("operation failed")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["error"] != "database connection failed" {
		t.Errorf("Expected error to be 'database connection failed', got %v", logEntry["error"])
	}

	if logEntry["message"] != "operation failed" {
		t.Errorf("Expected message 'operation failed', got %v", logEntry["message"])
	}
}

func TestLogFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"json format", "json"},
		{"console format", "console"},
		{"pretty format", "pretty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily redirect stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			cfg := &config.Config{
				Env:       "test",
				LogLevel:  "info",
				LogFormat: tt.format,
			}

			logger := New(cfg)
			logger.Info("test message")

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			// Verify output exists
			if output == "" {
				t.Error("Expected log output, got empty string")
			}

			// Verify message appears in output
			if !strings.Contains(output, "test message") {
				t.Errorf("Expected output to contain 'test message', got: %s", output)
			}
		})
	}
}
