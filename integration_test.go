package vecto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := newHTTPTestServer()
	defer srv.Close()

	t.Run("complete workflow with all features", func(t *testing.T) {
		logger := &mockLogger{}
		collector := &mockMetricsCollector{}

		cbConfig := DefaultCircuitBreakerConfig()
		cbConfig.FailureThreshold = 3
		cbConfig.Timeout = 100 * time.Millisecond

		v, err := New(Config{
			BaseURL:          srv.URL,
			Logger:           logger,
			MetricsCollector: collector,
			CircuitBreaker:   &cbConfig,
			EnableTrace:      true,
			Headers: map[string]string{
				"X-API-Key": "test-key",
			},
			Retry: &RetryConfig{
				MaxAttempts: 3,
				WaitTime:    10 * time.Millisecond,
				MaxWaitTime: 100 * time.Millisecond,
				Backoff:     ExponentialBackoff,
			},
		})
		assert.Nil(t, err)

		requestCompleted := false
		v.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
			req.SetHeader("X-Request-Time", time.Now().Format(time.RFC3339))
			return req, nil
		})

		v.UseResponse(func(ctx context.Context, res *Response) (*Response, error) {
			requestCompleted = true
			return res, nil
		})

		res, err := v.Get(context.Background(), "/test/pets/1", &RequestOptions{
			Params: map[string]any{
				"expand": "owner",
			},
		})

		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.True(t, requestCompleted)
		assert.True(t, len(collector.requests) > 0)
		assert.NotNil(t, res.TraceInfo)
	})

	t.Run("concurrent requests with circuit breaker", func(t *testing.T) {
		cbConfig := DefaultCircuitBreakerConfig()
		cbConfig.FailureThreshold = 5

		v, err := New(Config{
			BaseURL:        srv.URL,
			CircuitBreaker: &cbConfig,
		})
		assert.Nil(t, err)

		var wg sync.WaitGroup
		results := make([]error, 20)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_, err := v.Get(context.Background(), "/test/status/200", nil)
				results[idx] = err
			}(i)
		}

		wg.Wait()

		successCount := 0
		for _, err := range results {
			if err == nil {
				successCount++
			}
		}

		assert.True(t, successCount > 0, "expected at least one successful request")
	})

	t.Run("retry mechanism with temporary failures", func(t *testing.T) {
		attempts := 0
		var mu sync.Mutex

		v, err := New(Config{
			BaseURL: srv.URL,
			Retry: &RetryConfig{
				MaxAttempts: 3,
				WaitTime:    10 * time.Millisecond,
				MaxWaitTime: 50 * time.Millisecond,
				Backoff:     LinearBackoff,
			},
		})
		assert.Nil(t, err)

		v.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
			mu.Lock()
			attempts++
			mu.Unlock()
			return req, nil
		})

		_, err = v.Get(context.Background(), "/test/status/500", nil)

		mu.Lock()
		finalAttempts := attempts
		mu.Unlock()

		assert.NotNil(t, err)
		assert.True(t, finalAttempts >= 1, fmt.Sprintf("expected multiple attempts, got %d", finalAttempts))
	})

	t.Run("metrics collection across multiple requests", func(t *testing.T) {
		collector := &mockMetricsCollector{}

		v, err := New(Config{
			BaseURL:          srv.URL,
			MetricsCollector: collector,
		})
		assert.Nil(t, err)

		endpoints := []string{
			"/test/status/200",
			"/test/status/201",
			"/test/status/404",
			"/test/pets/1",
		}

		for _, endpoint := range endpoints {
			_, _ = v.Get(context.Background(), endpoint, nil)
		}

		assert.Equal(t, len(endpoints), len(collector.requests))

		statusCodes := make(map[int]int)
		for _, metric := range collector.requests {
			statusCodes[metric.StatusCode]++
		}

		assert.True(t, len(statusCodes) > 0)
	})

	t.Run("authentication flow", func(t *testing.T) {
		v, err := New(Config{
			BaseURL: srv.URL,
		})
		assert.Nil(t, err)

		res, err := v.Get(context.Background(), "/test/methods", nil)

		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("request timeout handling", func(t *testing.T) {
		v, err := New(Config{
			BaseURL: srv.URL,
			Timeout: 50 * time.Millisecond,
		})
		assert.Nil(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err = v.Get(ctx, "/test/status/200", nil)
		assert.Nil(t, err)
	})

	t.Run("complex data transformation", func(t *testing.T) {
		v, err := New(Config{
			BaseURL: srv.URL,
		})
		assert.Nil(t, err)

		type TestData struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		testData := TestData{
			Name:  "John Doe",
			Email: "john@example.com",
		}

		res, err := v.Post(context.Background(), "/test/methods", &RequestOptions{
			Data: testData,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		})

		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("response parsing and validation", func(t *testing.T) {
		v, err := New(Config{
			BaseURL: srv.URL,
		})
		assert.Nil(t, err)

		res, err := v.Get(context.Background(), "/test/pets/1", nil)
		assert.Nil(t, err)

		var pet PetMockData
		err = json.Unmarshal(res.Data, &pet)
		assert.Nil(t, err)
		assert.NotEmpty(t, pet.ID)
	})

	t.Run("middleware chain execution order", func(t *testing.T) {
		v, err := New(Config{
			BaseURL: srv.URL,
		})
		assert.Nil(t, err)

		executionOrder := []string{}
		var mu sync.Mutex

		v.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
			mu.Lock()
			executionOrder = append(executionOrder, "request-1")
			mu.Unlock()
			return req, nil
		})

		v.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
			mu.Lock()
			executionOrder = append(executionOrder, "request-2")
			mu.Unlock()
			return req, nil
		})

		v.UseResponse(func(ctx context.Context, res *Response) (*Response, error) {
			mu.Lock()
			executionOrder = append(executionOrder, "response-1")
			mu.Unlock()
			return res, nil
		})

		v.UseResponse(func(ctx context.Context, res *Response) (*Response, error) {
			mu.Lock()
			executionOrder = append(executionOrder, "response-2")
			mu.Unlock()
			return res, nil
		})

		_, err = v.Get(context.Background(), "/test/status/200", nil)
		assert.Nil(t, err)

		mu.Lock()
		defer mu.Unlock()

		assert.Equal(t, "request-1", executionOrder[0])
		assert.Equal(t, "request-2", executionOrder[1])
		assert.Contains(t, executionOrder, "response-1")
		assert.Contains(t, executionOrder, "response-2")
	})
}

func TestIntegrationCircuitBreakerRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := newHTTPTestServer()
	defer srv.Close()

	t.Run("circuit breaker opens and recovers", func(t *testing.T) {
		cbConfig := DefaultCircuitBreakerConfig()
		cbConfig.FailureThreshold = 2
		cbConfig.Timeout = 100 * time.Millisecond
		cbConfig.HalfOpenMaxRequests = 2

		v, err := New(Config{
			BaseURL:        srv.URL,
			CircuitBreaker: &cbConfig,
		})
		assert.Nil(t, err)

		_, _ = v.Get(context.Background(), "/test/status/500", nil)
		_, _ = v.Get(context.Background(), "/test/status/500", nil)

		_, err = v.Get(context.Background(), "/test/status/200", nil)
		if err != nil {
			assert.Contains(t, err.Error(), "circuit breaker")
		}

		time.Sleep(150 * time.Millisecond)

		res, err := v.Get(context.Background(), "/test/status/200", nil)
		if err == nil {
			assert.Equal(t, http.StatusOK, res.StatusCode)
		}
	})
}

func TestIntegrationConcurrentSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := newHTTPTestServer()
	defer srv.Close()

	t.Run("concurrent requests are thread-safe", func(t *testing.T) {
		v, err := New(Config{
			BaseURL: srv.URL,
		})
		assert.Nil(t, err)

		var wg sync.WaitGroup
		numGoroutines := 50
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				endpoint := fmt.Sprintf("/test/status/200?id=%d", id)
				res, err := v.Get(context.Background(), endpoint, nil)

				if err == nil && res.StatusCode == http.StatusOK {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, numGoroutines, successCount)
	})
}
