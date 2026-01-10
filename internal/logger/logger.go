// Package logger provides a simple wrapper around slog for structured logging.
package logger

import (
	"log/slog"
	"os"
)

// Logger is the global logger instance.
var Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

// Error logs an error message.
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}
