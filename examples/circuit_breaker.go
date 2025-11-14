//go:build example_circuit_breaker
// +build example_circuit_breaker

// This is a standalone example program. Each example file has its own main function
// and should be run individually: go run -tags example_circuit_breaker circuit_breaker.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caio-campos/vecto"
)

func main() {
	ExampleCircuitBreaker()
}

func ExampleCircuitBreaker() {
	cbConfig := vecto.DefaultCircuitBreakerConfig()
	cbConfig.FailureThreshold = 3
	cbConfig.SuccessThreshold = 2
	cbConfig.Timeout = 30 * time.Second
	cbConfig.HalfOpenMaxRequests = 1
	cbConfig.WindowSize = 60 * time.Second

	cbConfig.OnStateChange = func(from, to vecto.CircuitBreakerState, key string) {
		log.Printf("Circuit breaker state changed for %s: %s -> %s", key, from.String(), to.String())
	}

	cbConfig.ShouldTrip = func(res *vecto.Response, err error) bool {
		if err != nil {
			return true
		}
		if res == nil {
			return true
		}
		return res.StatusCode >= 500
	}

	config := vecto.Config{
		BaseURL:        "https://httpbin.org",
		CircuitBreaker: &cbConfig,
	}

	client, err := vecto.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		res, err := client.Get(ctx, "/status/500", nil)
		if err != nil {
			if cbErr, ok := err.(*vecto.CircuitBreakerError); ok {
				fmt.Printf("Request %d: Blocked by circuit breaker (state: %s)\n", i+1, cbErr.State.String())
			} else {
				fmt.Printf("Request %d: Error: %v\n", i+1, err)
			}
		} else {
			fmt.Printf("Request %d: Status %d\n", i+1, res.StatusCode)
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\nWaiting for circuit breaker to transition to half-open...")
	time.Sleep(35 * time.Second)

	fmt.Println("\nAttempting requests after timeout:")
	for i := 0; i < 3; i++ {
		res, err := client.Get(ctx, "/status/200", nil)
		if err != nil {
			if cbErr, ok := err.(*vecto.CircuitBreakerError); ok {
				fmt.Printf("Request %d: Blocked by circuit breaker (state: %s)\n", i+1, cbErr.State.String())
			} else {
				fmt.Printf("Request %d: Error: %v\n", i+1, err)
			}
		} else {
			fmt.Printf("Request %d: Status %d\n", i+1, res.StatusCode)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
