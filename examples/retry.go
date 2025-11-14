//go:build example_retry
// +build example_retry

// This is a standalone example program. Each example file has its own main function
// and should be run individually: go run -tags example_retry retry_example.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caio-campos/vecto"
)

func main() {
	fmt.Println("=== Vecto Retry Examples ===")
	fmt.Println()

	exponentialBackoffExample()
	linearBackoffExample()
	customRetryConditionExample()
	perRequestRetryExample()
}

func exponentialBackoffExample() {
	fmt.Println("1. Exponential Backoff Example")
	fmt.Println("-------------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
		Retry: &vecto.RetryConfig{
			MaxAttempts:             5,
			WaitTime:                time.Second,
			MaxWaitTime:             30 * time.Second,
			Backoff:                 vecto.ExponentialBackoff,
			RespectRetryAfterHeader: true,
			OnRetry: func(attempt int, err error) {
				fmt.Printf("  → Retrying... Attempt %d\n", attempt)
				if err != nil {
					fmt.Printf("    Error: %v\n", err)
				}
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Attempting request to /status/503 (Service Unavailable)...")
	res, err := client.Get(ctx, "/status/503", nil)
	if err != nil {
		log.Printf("Request failed after retries: %v\n", err)
	} else if res.IsError() {
		fmt.Printf("Request completed with error status: %d\n", res.StatusCode)
	} else {
		fmt.Printf("Request succeeded with status: %d\n", res.StatusCode)
	}
	fmt.Println()
}

func linearBackoffExample() {
	fmt.Println("2. Linear Backoff Example")
	fmt.Println("--------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
		Retry: &vecto.RetryConfig{
			MaxAttempts: 3,
			WaitTime:    500 * time.Millisecond,
			MaxWaitTime: 5 * time.Second,
			Backoff:     vecto.LinearBackoff,
			OnRetry: func(attempt int, err error) {
				fmt.Printf("  → Retry attempt %d (linear backoff)\n", attempt)
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Attempting request to /status/500 (Internal Server Error)...")
	res, err := client.Get(ctx, "/status/500", nil)
	if err != nil {
		log.Printf("Request failed after retries: %v\n", err)
	} else if res.IsError() {
		fmt.Printf("Request completed with error status: %d\n", res.StatusCode)
	}
	fmt.Println()
}

func customRetryConditionExample() {
	fmt.Println("3. Custom Retry Condition Example")
	fmt.Println("----------------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
		Retry: &vecto.RetryConfig{
			MaxAttempts: 3,
			WaitTime:    time.Second,
			Backoff:     vecto.FixedBackoff,
			RetryCondition: func(res *vecto.Response, err error) bool {
				if err != nil {
					return true
				}
				if res == nil {
					return false
				}

				return res.StatusCode == 404 || res.StatusCode >= 500
			},
			OnRetry: func(attempt int, err error) {
				fmt.Printf("  → Custom retry logic triggered (attempt %d)\n", attempt)
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Attempting request to /status/404 (Not Found)...")
	fmt.Println("Note: Our custom condition will retry on 404...")
	res, err := client.Get(ctx, "/status/404", nil)
	if err != nil {
		log.Printf("Request failed after retries: %v\n", err)
	} else if res.IsClientError() {
		fmt.Printf("Request completed with client error: %d\n", res.StatusCode)
	}
	fmt.Println()

	fmt.Println("Attempting request to /status/200 (Success)...")
	res2, err := client.Get(ctx, "/status/200", nil)
	if err != nil {
		log.Printf("Request failed: %v\n", err)
	} else if res2.IsSuccess() {
		fmt.Printf("Request succeeded on first try: %d\n", res2.StatusCode)
	}

	fmt.Println("\n=== Examples completed ===")
}

func perRequestRetryExample() {
	fmt.Println("4. Per-Request Retry Override Example")
	fmt.Println("--------------------------------------")

	client, err := vecto.New(vecto.Config{
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
		Retry: &vecto.RetryConfig{
			MaxAttempts: 5,
			WaitTime:    time.Second,
			Backoff:     vecto.ExponentialBackoff,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	maxRetries := 2

	fmt.Printf("Attempting request with only %d retries (overriding default)...\n", maxRetries)
	res, err := client.Get(ctx, "/status/503", &vecto.RequestOptions{
		MaxRetries: &maxRetries,
	})

	if err != nil {
		log.Printf("Request failed after retries: %v\n", err)
	} else if res.IsError() {
		fmt.Printf("Request completed with error status: %d\n", res.StatusCode)
	}

	fmt.Println()
}
