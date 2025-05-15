package logger

import (
	"context"
	"log/slog"
	"os"
)

var (
	// Default logger instance
	Log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
)

// WithContext returns a logger with context values
func WithContext(ctx context.Context) *slog.Logger {
	return Log.With(
		"trace_id", ctx.Value("trace_id"),
		"request_id", ctx.Value("request_id"),
	)
}

// Error logs an error with additional context
func Error(msg string, err error, args ...any) {
	args = append(args, "error", err.Error())
	Log.Error(msg, args...)
}

// Info logs an info message with additional context
func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

// Debug logs a debug message with additional context
func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

// Warn logs a warning message with additional context
func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}
