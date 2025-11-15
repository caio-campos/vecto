package vecto

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkConcurrentRequests(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := vecto.Get(context.Background(), "/test/status/200", nil)
			if err != nil {
				b.Logf("request failed: %v", err)
				continue
			}
		}
	})
}

func BenchmarkGet(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vecto.Get(context.Background(), "/test/status/200", nil)
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
	}
}

func BenchmarkPost(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vecto.Post(context.Background(), "/test/methods", nil)
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
	}
}

func BenchmarkRequestWithHeaders(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{
		BaseURL: srv.URL,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "value",
		},
	})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{
			Headers: map[string]string{
				"X-Request-Id": "test-123",
			},
		})
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
	}
}

func BenchmarkRequestWithParams(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vecto.Get(context.Background(), "/test/query", &RequestOptions{
			Params: map[string]any{
				"added_param": "1",
			},
		})
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
	}
}

func BenchmarkRequestCreation(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vecto.newRequest("/test/pets/1", "GET", &RequestOptions{
			Headers: map[string]string{
				"X-Request-Id": "test-123",
			},
			Params: map[string]any{
				"page": 1,
				"limit": 10,
			},
		})
		if err != nil {
			b.Fatalf("request creation failed: %v", err)
		}
	}
}

func BenchmarkCircuitBreaker_ExecuteClosed(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker("test", config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			res, err := cb.Execute(context.Background(), func() (*Response, error) {
				return &Response{StatusCode: 200}, nil
			})
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			cb.RecordResult(res, nil)
		}
	})
}

func BenchmarkCircuitBreaker_ExecuteOpen(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	cb := NewCircuitBreaker("test", config)

	res, err := cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 500}, nil
	})
	if err != nil {
		b.Fatalf("failed to open circuit: %v", err)
	}
	cb.RecordResult(res, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := cb.Execute(context.Background(), func() (*Response, error) {
				return &Response{StatusCode: 200}, nil
			})
			if err == nil {
				b.Fatalf("expected circuit breaker error")
			}
		}
	})
}

func BenchmarkCircuitBreaker_ExecuteHalfOpen(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.Timeout = 1 * time.Hour
	config.HalfOpenMaxRequests = 10
	cb := NewCircuitBreaker("test", config)

	res, err := cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 500}, nil
	})
	if err != nil {
		b.Fatalf("failed to open circuit: %v", err)
	}
	cb.RecordResult(res, nil)

	now := time.Now()
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.stateChangeTime = now
	cb.halfOpenRequests = 0
	cb.consecutiveSuccesses = 0
	cb.mu.Unlock()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			res, err := cb.Execute(context.Background(), func() (*Response, error) {
				return &Response{StatusCode: 200}, nil
			})
			if err != nil {
				continue
			}
			cb.RecordResult(res, nil)
		}
	})
}

func BenchmarkCircuitBreaker_RecordResult(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker("test", config)
	res := &Response{StatusCode: 200}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.RecordResult(res, nil)
		}
	})
}

func BenchmarkCircuitBreaker_GetStats(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker("test", config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.GetStats()
		}
	})
}

func BenchmarkCircuitBreakerManager_GetOrCreate(b *testing.B) {
	config := DefaultCircuitBreakerConfig()
	mgr := NewCircuitBreakerManager(config, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("test-key-%d", i%100)
			_ = mgr.GetOrCreate(key, nil)
			i++
		}
	})
}

func BenchmarkCircuitBreaker_WithVecto(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	cbConfig := DefaultCircuitBreakerConfig()
	vecto, err := New(Config{
		BaseURL:        srv.URL,
		CircuitBreaker: &cbConfig,
	})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := vecto.Get(context.Background(), "/test/status/200", nil)
			if err != nil {
				b.Logf("request failed: %v", err)
				continue
			}
		}
	})
}

func BenchmarkCircuitBreaker_WithVectoOpen(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	cbConfig := DefaultCircuitBreakerConfig()
	cbConfig.FailureThreshold = 1
	vecto, err := New(Config{
		BaseURL:        srv.URL,
		CircuitBreaker: &cbConfig,
	})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	for i := 0; i < 2; i++ {
		_, _ = vecto.Get(context.Background(), "/test/status/500", nil)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := vecto.Get(context.Background(), "/test/status/200", nil)
			if err != nil {
				continue
			}
		}
	})
}

func BenchmarkCircuitBreaker_Overhead(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	withoutCB, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	cbConfig := DefaultCircuitBreakerConfig()
	withCB, err := New(Config{
		BaseURL:        srv.URL,
		CircuitBreaker: &cbConfig,
	})
	if err != nil {
		b.Fatalf("failed to create vecto instance: %v", err)
	}

	b.Run("WithoutCircuitBreaker", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := withoutCB.Get(context.Background(), "/test/status/200", nil)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
		}
	})

	b.Run("WithCircuitBreaker", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := withCB.Get(context.Background(), "/test/status/200", nil)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
		}
	})
}

func BenchmarkStringBuilding(b *testing.B) {
	scheme := "https"
	host := "api.example.com"
	path := "/v1/users/123"

	b.Run("WithStringBuilder", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sb := getStringBuilder()
			sb.WriteString(scheme)
			sb.WriteString("://")
			sb.WriteString(host)
			sb.WriteString(path)
			_ = sb.String()
			putStringBuilder(sb)
		}
	})

	b.Run("WithSprintf", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("%s://%s%s", scheme, host, path)
		}
	})

	b.Run("WithConcatenation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = scheme + "://" + host + path
		}
	})
}

func BenchmarkMapPreallocation(b *testing.B) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token",
		"X-Request-Id":  "123",
		"User-Agent":    "vecto/1.0",
	}

	b.Run("WithPreallocation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := make(map[string]string, len(headers))
			for k, v := range headers {
				result[k] = v
			}
		}
	})

	b.Run("WithoutPreallocation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := make(map[string]string)
			for k, v := range headers {
				result[k] = v
			}
		}
	})
}

func BenchmarkRequestBuilderOptimized(b *testing.B) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token",
		"X-Request-Id":  "123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := newRequestBuilder("http://api.example.com/users", "GET")
		builder.SetHeaders(headers)
		_, _ = builder.Build()
	}
}

func BenchmarkRetryMechanism(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	b.Run("NoRetry", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})

	b.Run("WithRetryDisabled", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
			Retry: &RetryConfig{
				MaxAttempts: 1,
			},
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})

	b.Run("WithRetryEnabled", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
			Retry: &RetryConfig{
				MaxAttempts: 3,
				WaitTime:    1 * time.Millisecond,
				MaxWaitTime: 10 * time.Millisecond,
				Backoff:     ExponentialBackoff,
			},
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})
}

func BenchmarkMetricsCollection(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	b.Run("WithoutMetrics", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})

	b.Run("WithMetrics", func(b *testing.B) {
		collector := &mockMetricsCollector{}
		vecto, err := New(Config{
			BaseURL:          srv.URL,
			MetricsCollector: collector,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})

	b.Run("MetricsCollectionConcurrent", func(b *testing.B) {
		collector := &mockMetricsCollector{}
		vecto, err := New(Config{
			BaseURL:          srv.URL,
			MetricsCollector: collector,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
			}
		})
	})
}

func BenchmarkMiddlewareChain(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	b.Run("NoMiddleware", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})

	b.Run("SingleRequestMiddleware", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		vecto.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
			req.SetHeader("X-Custom-Header", "test")
			return req, nil
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})

	b.Run("MultipleMiddlewares", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		for j := 0; j < 5; j++ {
			headerName := fmt.Sprintf("X-Header-%d", j)
			vecto.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
				req.SetHeader(headerName, "value")
				return req, nil
			})
			vecto.UseResponse(func(ctx context.Context, res *Response) (*Response, error) {
				return res, nil
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})
}

func BenchmarkTraceEnabled(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	b.Run("WithoutTrace", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL:     srv.URL,
			EnableTrace: false,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})

	b.Run("WithTrace", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL:     srv.URL,
			EnableTrace: true,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
		}
	})
}

func BenchmarkRetryBackoffStrategies(b *testing.B) {
	config := &RetryConfig{
		WaitTime:    1 * time.Millisecond,
		MaxWaitTime: 100 * time.Millisecond,
	}

	b.Run("ExponentialBackoff", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for attempt := 0; attempt < 5; attempt++ {
				_ = ExponentialBackoff(attempt, config)
			}
		}
	})

	b.Run("LinearBackoff", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for attempt := 0; attempt < 5; attempt++ {
				_ = LinearBackoff(attempt, config)
			}
		}
	})

	b.Run("FixedBackoff", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for attempt := 0; attempt < 5; attempt++ {
				_ = FixedBackoff(attempt, config)
			}
		}
	})
}

func BenchmarkCompleteRequestFlow(b *testing.B) {
	srv := newHTTPTestServer()
	defer srv.Close()

	b.Run("MinimalConfig", func(b *testing.B) {
		vecto, err := New(Config{
			BaseURL: srv.URL,
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
			}
		})
	})

	b.Run("FullFeatured", func(b *testing.B) {
		collector := &mockMetricsCollector{}
		cbConfig := DefaultCircuitBreakerConfig()

		vecto, err := New(Config{
			BaseURL:          srv.URL,
			MetricsCollector: collector,
			CircuitBreaker:   &cbConfig,
			EnableTrace:      true,
			Retry: &RetryConfig{
				MaxAttempts: 2,
				WaitTime:    1 * time.Millisecond,
			},
		})
		if err != nil {
			b.Fatalf("failed to create vecto instance: %v", err)
		}

		vecto.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
			return req, nil
		})
		vecto.UseResponse(func(ctx context.Context, res *Response) (*Response, error) {
			return res, nil
		})

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = vecto.Get(context.Background(), "/test/status/200", nil)
			}
		})
	})
}

