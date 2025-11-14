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

