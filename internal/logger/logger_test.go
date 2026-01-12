package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

type logRecord struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer

	// Use JSON handler for easier parsing in tests
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	testLogger := slog.New(handler)

	originalLogger := Logger
	Logger = testLogger
	defer func() { Logger = originalLogger }()

	tests := []struct {
		name  string
		fn    func(msg string, args ...any)
		level string
		msg   string
	}{
		{
			name:  "Info",
			fn:    Info,
			level: "INFO",
			msg:   "info message",
		},
		{
			name:  "Error",
			fn:    Error,
			level: "ERROR",
			msg:   "error message",
		},
		{
			name:  "Warn",
			fn:    Warn,
			level: "WARN",
			msg:   "warn message",
		},
		{
			name:  "Debug",
			fn:    Debug,
			level: "DEBUG",
			msg:   "debug message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.fn(tt.msg)

			var rec logRecord
			if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
				t.Fatalf("failed to unmarshal log output: %v", err)
			}

			if rec.Msg != tt.msg {
				t.Errorf("expected msg %q, got %q", tt.msg, rec.Msg)
			}
			if rec.Level != tt.level {
				t.Errorf("expected level %q, got %q", tt.level, rec.Level)
			}
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	if Logger == nil {
		t.Error("Logger should be initialized")
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	Info("test context")
}
