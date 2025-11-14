//go:build example_trace_debug
// +build example_trace_debug

// This is a standalone example program. Each example file has its own main function
// and should be run individually: go run -tags example_trace_debug trace_debug_example.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/caio-campos/vecto"
)

func main() {
	fmt.Println("=== Vecto Trace & Debug Examples ===")
	fmt.Println()

	traceExample()
	debugModeExample()
	combinedExample()
}

func traceExample() {
	fmt.Println("1. Request Tracing Example")
	fmt.Println("---------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL:     "https://httpbin.org",
		Timeout:     30 * time.Second,
		EnableTrace: true,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	res, err := client.Get(ctx, "/get", nil)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if res.TraceInfo != nil {
		fmt.Println("\nRequest Trace Information:")
		fmt.Printf("  DNS Lookup:        %v\n", res.TraceInfo.DNSLookup)
		fmt.Printf("  TCP Connection:    %v\n", res.TraceInfo.TCPConnection)
		fmt.Printf("  TLS Handshake:     %v\n", res.TraceInfo.TLSHandshake)
		fmt.Printf("  Server Processing: %v\n", res.TraceInfo.ServerProcessing)
		fmt.Printf("  Content Transfer:  %v\n", res.TraceInfo.ContentTransfer)
		fmt.Printf("  Total Time:        %v\n", res.TraceInfo.Total)
		fmt.Printf("  Conn Reused:       %v\n", res.TraceInfo.ConnReused)
		
		if res.TraceInfo.ConnReused {
			fmt.Printf("  Conn Was Idle:     %v\n", res.TraceInfo.ConnWasIdle)
			fmt.Printf("  Conn Idle Time:    %v\n", res.TraceInfo.ConnIdleTime)
		}
	}
	fmt.Println()
}

func debugModeExample() {
	fmt.Println("2. Debug Mode Example")
	fmt.Println("---------------------")

	logger := &simpleLogger{writer: os.Stdout}

	client, err := vecto.New(vecto.Config{
		BaseURL:   "https://httpbin.org",
		Timeout:   30 * time.Second,
		DebugMode: true,
		Logger:    logger,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	fmt.Println("Making request with debug mode enabled...")
	_, err = client.Post(ctx, "/post", &vecto.RequestOptions{
		Data: map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		},
	})
	
	if err != nil {
		log.Printf("Request failed: %v", err)
	}
	
	fmt.Println()
}

func combinedExample() {
	fmt.Println("3. Combined Trace + Debug Example")
	fmt.Println("----------------------------------")

	logger := &simpleLogger{writer: os.Stdout}

	client, err := vecto.New(vecto.Config{
		BaseURL:     "https://httpbin.org",
		Timeout:     30 * time.Second,
		EnableTrace: true,
		DebugMode:   true,
		Logger:      logger,
		Headers: map[string]string{
			"User-Agent": "Vecto-Example/1.0",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.SetBearerToken("example-token-123")

	ctx := context.Background()
	
	fmt.Println("Making authenticated request with full tracing...")
	res, err := client.Get(ctx, "/bearer", nil)
	
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	fmt.Printf("\nResponse Status: %d\n", res.StatusCode)
	if res.IsSuccess() {
		fmt.Println("✓ Request succeeded")
	}
	
	fmt.Println("\n=== Examples completed ===")
}

type simpleLogger struct {
	writer *os.File
}

func (l *simpleLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Fprintf(l.writer, "[DEBUG] %s\n", message)
}

func (l *simpleLogger) Info(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Fprintf(l.writer, "[INFO] %s\n", message)
}

func (l *simpleLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Fprintf(l.writer, "[WARN] %s\n", message)
}

func (l *simpleLogger) Error(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Fprintf(l.writer, "[ERROR] %s\n", message)
}

func (l *simpleLogger) IsNoop() bool {
	return false
}

