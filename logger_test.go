package vecto

import (
	"context"
	"testing"
)

type mockLogger struct {
	debugCalls []logCall
	infoCalls  []logCall
	warnCalls  []logCall
	errorCalls []logCall
}

type logCall struct {
	msg    string
	fields map[string]interface{}
}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	m.debugCalls = append(m.debugCalls, logCall{msg: msg, fields: fields})
}

func (m *mockLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	m.infoCalls = append(m.infoCalls, logCall{msg: msg, fields: fields})
}

func (m *mockLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	m.warnCalls = append(m.warnCalls, logCall{msg: msg, fields: fields})
}

func (m *mockLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	m.errorCalls = append(m.errorCalls, logCall{msg: msg, fields: fields})
}

func (m *mockLogger) IsNoop() bool {
	return false
}

func TestNoopLogger(t *testing.T) {
	t.Run("noop logger does not panic", func(t *testing.T) {
		logger := newNoopLogger()
		ctx := context.Background()

		logger.Debug(ctx, "debug message", map[string]interface{}{"key": "value"})
		logger.Info(ctx, "info message", map[string]interface{}{"key": "value"})
		logger.Warn(ctx, "warn message", map[string]interface{}{"key": "value"})
		logger.Error(ctx, "error message", map[string]interface{}{"key": "value"})
	})

	t.Run("noop logger returns true for IsNoop", func(t *testing.T) {
		logger := newNoopLogger()
		if !logger.IsNoop() {
			t.Error("expected IsNoop to return true")
		}
	})

	t.Run("noop logger with nil fields", func(t *testing.T) {
		logger := newNoopLogger()
		ctx := context.Background()

		logger.Debug(ctx, "debug", nil)
		logger.Info(ctx, "info", nil)
		logger.Warn(ctx, "warn", nil)
		logger.Error(ctx, "error", nil)
	})

	t.Run("noop logger with empty fields", func(t *testing.T) {
		logger := newNoopLogger()
		ctx := context.Background()

		logger.Debug(ctx, "debug", map[string]interface{}{})
		logger.Info(ctx, "info", map[string]interface{}{})
		logger.Warn(ctx, "warn", map[string]interface{}{})
		logger.Error(ctx, "error", map[string]interface{}{})
	})

	t.Run("noop logger with nil context", func(t *testing.T) {
		logger := newNoopLogger()

		logger.Debug(context.TODO(), "debug", nil)
		logger.Info(context.TODO(), "info", nil)
		logger.Warn(context.TODO(), "warn", nil)
		logger.Error(context.TODO(), "error", nil)
	})
}

func TestMockLogger(t *testing.T) {
	t.Run("mock logger records debug calls", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := context.Background()
		fields := map[string]interface{}{"user_id": 123}

		logger.Debug(ctx, "test debug", fields)

		if len(logger.debugCalls) != 1 {
			t.Fatalf("expected 1 debug call, got %d", len(logger.debugCalls))
		}
		if logger.debugCalls[0].msg != "test debug" {
			t.Errorf("expected message 'test debug', got %s", logger.debugCalls[0].msg)
		}
	})

	t.Run("mock logger records info calls", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := context.Background()
		fields := map[string]interface{}{"request_id": "abc123"}

		logger.Info(ctx, "test info", fields)

		if len(logger.infoCalls) != 1 {
			t.Fatalf("expected 1 info call, got %d", len(logger.infoCalls))
		}
		if logger.infoCalls[0].msg != "test info" {
			t.Errorf("expected message 'test info', got %s", logger.infoCalls[0].msg)
		}
	})

	t.Run("mock logger records warn calls", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := context.Background()
		fields := map[string]interface{}{"warning": "rate limit"}

		logger.Warn(ctx, "test warn", fields)

		if len(logger.warnCalls) != 1 {
			t.Fatalf("expected 1 warn call, got %d", len(logger.warnCalls))
		}
		if logger.warnCalls[0].msg != "test warn" {
			t.Errorf("expected message 'test warn', got %s", logger.warnCalls[0].msg)
		}
	})

	t.Run("mock logger records error calls", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := context.Background()
		fields := map[string]interface{}{"error_code": 500}

		logger.Error(ctx, "test error", fields)

		if len(logger.errorCalls) != 1 {
			t.Fatalf("expected 1 error call, got %d", len(logger.errorCalls))
		}
		if logger.errorCalls[0].msg != "test error" {
			t.Errorf("expected message 'test error', got %s", logger.errorCalls[0].msg)
		}
	})

	t.Run("mock logger is not noop", func(t *testing.T) {
		logger := &mockLogger{}
		if logger.IsNoop() {
			t.Error("expected IsNoop to return false")
		}
	})

	t.Run("mock logger records multiple calls", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := context.Background()

		logger.Debug(ctx, "debug 1", nil)
		logger.Debug(ctx, "debug 2", nil)
		logger.Info(ctx, "info 1", nil)
		logger.Info(ctx, "info 2", nil)
		logger.Warn(ctx, "warn 1", nil)
		logger.Error(ctx, "error 1", nil)

		if len(logger.debugCalls) != 2 {
			t.Errorf("expected 2 debug calls, got %d", len(logger.debugCalls))
		}
		if len(logger.infoCalls) != 2 {
			t.Errorf("expected 2 info calls, got %d", len(logger.infoCalls))
		}
		if len(logger.warnCalls) != 1 {
			t.Errorf("expected 1 warn call, got %d", len(logger.warnCalls))
		}
		if len(logger.errorCalls) != 1 {
			t.Errorf("expected 1 error call, got %d", len(logger.errorCalls))
		}
	})
}

func TestLoggerIntegration(t *testing.T) {
	t.Run("vecto uses logger when provided", func(t *testing.T) {
		logger := &mockLogger{}

		_, err := New(Config{
			BaseURL: "https://api.example.com",
			Logger:  logger,
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}
	})

	t.Run("vecto uses noop logger by default", func(t *testing.T) {
		v, err := New(Config{
			BaseURL: "https://api.example.com",
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		if !v.logger.IsNoop() {
			t.Error("expected default logger to be noop")
		}
	})
}
