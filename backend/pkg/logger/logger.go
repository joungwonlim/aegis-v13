package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/wonny/aegis/v13/backend/pkg/config"
)

// Logger is a structured logger wrapper around zerolog
// ⭐ SSOT: 모든 로깅은 이 패키지를 통해서만 수행
type Logger struct {
	zlog zerolog.Logger
}

// New creates a new Logger instance from config
// ⭐ SSOT: zerolog 인스턴스는 여기서만 생성
func New(cfg *config.Config) *Logger {
	// Configure output format
	var output io.Writer
	if cfg.LogFormat == "console" || cfg.LogFormat == "pretty" {
		// Human-readable console output
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	} else {
		// JSON output (default)
		output = os.Stdout
	}

	// Set log level
	level := parseLogLevel(cfg.LogLevel)
	zerolog.SetGlobalLevel(level)

	// Create logger
	zlog := zerolog.New(output).
		With().
		Timestamp().
		Str("env", cfg.Env).
		Logger()

	return &Logger{zlog: zlog}
}

// parseLogLevel converts string log level to zerolog.Level
func parseLogLevel(levelStr string) zerolog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.zlog.Debug().Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.zlog.Info().Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.zlog.Warn().Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.zlog.Error().Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.zlog.Fatal().Msg(msg)
}

// WithField returns a new logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := l.zlog.With().Interface(key, value).Logger()
	return &Logger{zlog: newLogger}
}

// WithFields returns a new logger with multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.zlog.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{zlog: ctx.Logger()}
}

// WithError returns a new logger with an error field
func (l *Logger) WithError(err error) *Logger {
	newLogger := l.zlog.With().Err(err).Logger()
	return &Logger{zlog: newLogger}
}

// WithContext returns a new logger with context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	newLogger := l.zlog.With().Ctx(ctx).Logger()
	return &Logger{zlog: newLogger}
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.zlog.Debug().Msgf(format, args...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.zlog.Info().Msgf(format, args...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.zlog.Warn().Msgf(format, args...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.zlog.Error().Msgf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.zlog.Fatal().Msgf(format, args...)
}

// Zerolog returns the underlying zerolog.Logger
// 외부 패키지에서 zerolog 기능이 필요할 때 사용
func (l *Logger) Zerolog() zerolog.Logger {
	return l.zlog
}
