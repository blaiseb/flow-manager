package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

var (
	DebugMode bool = false
	handler slog.Handler
)

func InitLogger(logFile io.Writer, levelStr string) {
	var level slog.Level
	
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	w := io.MultiWriter(logFile, os.Stdout)
	handler = slog.NewTextHandler(w, opts)
	
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func Info(msg string, args ...any) { slog.Info(msg, args...) }
func Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func Error(msg string, args ...any) { slog.Error(msg, args...) }
func Warn(msg string, args ...any) { slog.Warn(msg, args...) }
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
func With(args ...any) *slog.Logger { return slog.Default().With(args...) }
func LogRequest(ctx context.Context, msg string, args ...any) { slog.InfoContext(ctx, msg, args...) }
