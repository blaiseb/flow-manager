package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

var (
	// DebugMode defines if we should log debug messages.
	DebugMode bool = false
	// Current handler
	handler slog.Handler
)

// InitLogger initializes the global logger.
func InitLogger(logFile io.Writer) {
	level := slog.LevelInfo
	if DebugMode {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// We use a multi-writer to log to both file and stdout.
	w := io.MultiWriter(logFile, os.Stdout)
	handler = slog.NewTextHandler(w, opts)
	
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Fatal logs an error message and exits.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// With returns a logger with the given attributes.
func With(args ...any) *slog.Logger {
	return slog.Default().With(args...)
}

// LogRequest is a helper for logging requests (for future use).
func LogRequest(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}
