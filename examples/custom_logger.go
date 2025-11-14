package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/caio-campos/vecto"
)

type SimpleLogger struct {
	prefix string
}

func NewSimpleLogger(prefix string) *SimpleLogger {
	return &SimpleLogger{prefix: prefix}
}

func (l *SimpleLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	log.Printf("[DEBUG] %s: %s %v", l.prefix, msg, fields)
}

func (l *SimpleLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	log.Printf("[INFO] %s: %s %v", l.prefix, msg, fields)
}

func (l *SimpleLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	log.Printf("[WARN] %s: %s %v", l.prefix, msg, fields)
}

func (l *SimpleLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	log.Printf("[ERROR] %s: %s %v", l.prefix, msg, fields)
}

func (l *SimpleLogger) IsNoop() bool {
	return false
}

func ExampleWithLogger() {
	logger := NewSimpleLogger("MyAPI")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://api.example.com",
		Logger:  logger,
	})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	res, err := client.Get(ctx, "/users/1", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Status: %d\n", res.StatusCode)
}

