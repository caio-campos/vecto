package vecto

import (
	"context"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]interface{})
	Info(ctx context.Context, msg string, fields map[string]interface{})
	Warn(ctx context.Context, msg string, fields map[string]interface{})
	Error(ctx context.Context, msg string, fields map[string]interface{})
	IsNoop() bool
}

type noopLogger struct{}

func (n *noopLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {}
func (n *noopLogger) Info(ctx context.Context, msg string, fields map[string]interface{})  {}
func (n *noopLogger) Warn(ctx context.Context, msg string, fields map[string]interface{})  {}
func (n *noopLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {}
func (n *noopLogger) IsNoop() bool                                                         { return true }

func newNoopLogger() Logger {
	return &noopLogger{}
}
