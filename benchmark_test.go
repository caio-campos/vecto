package vecto

import (
	"context"
	"testing"
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

