package vecto

import (
	"context"
	"net/http"
	"testing"
	"time"
)

type mockMetricsCollector struct {
	requests []RequestMetrics
}

func (m *mockMetricsCollector) RecordRequest(ctx context.Context, metrics RequestMetrics) {
	m.requests = append(m.requests, metrics)
}

func TestRecordMetrics(t *testing.T) {
	t.Run("records metrics with complete request and response", func(t *testing.T) {
		collector := &mockMetricsCollector{}
		v, err := New(Config{
			BaseURL:          "https://api.example.com",
			MetricsCollector: collector,
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		req, err := v.newRequest("/users/123", "GET", &RequestOptions{
			Headers: map[string]string{"X-Request-Id": "test"},
		})
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		res := &Response{
			StatusCode: 200,
			Data:       []byte(`{"id": 123}`),
			success:    true,
		}

		duration := 150 * time.Millisecond
		v.recordMetrics(context.Background(), req, res, duration, nil)

		if len(collector.requests) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(collector.requests))
		}

		metric := collector.requests[0]
		if metric.Method != "GET" {
			t.Errorf("expected method GET, got %s", metric.Method)
		}
		if metric.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", metric.StatusCode)
		}
		if metric.Duration != duration {
			t.Errorf("expected duration %v, got %v", duration, metric.Duration)
		}
		if !metric.Success {
			t.Error("expected success to be true")
		}
		if metric.ResponseSize == 0 {
			t.Errorf("expected response size > 0, got %d", metric.ResponseSize)
		}
	})

	t.Run("records metrics with nil request", func(t *testing.T) {
		collector := &mockMetricsCollector{}
		v, err := New(Config{
			BaseURL:          "https://api.example.com",
			MetricsCollector: collector,
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		res := &Response{
			StatusCode: 200,
			Data:       []byte(`{"status": "ok"}`),
			success:    true,
		}

		v.recordMetrics(context.Background(), nil, res, 100*time.Millisecond, nil)

		if len(collector.requests) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(collector.requests))
		}

		metric := collector.requests[0]
		if metric.Method != "" {
			t.Errorf("expected empty method, got %s", metric.Method)
		}
		if metric.FullURL != "" {
			t.Errorf("expected empty URL, got %s", metric.FullURL)
		}
	})

	t.Run("records metrics with nil response", func(t *testing.T) {
		collector := &mockMetricsCollector{}
		v, err := New(Config{
			BaseURL:          "https://api.example.com",
			MetricsCollector: collector,
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		req, err := v.newRequest("/users", "POST", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		v.recordMetrics(context.Background(), req, nil, 50*time.Millisecond, nil)

		if len(collector.requests) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(collector.requests))
		}

		metric := collector.requests[0]
		if metric.StatusCode != 0 {
			t.Errorf("expected status 0, got %d", metric.StatusCode)
		}
		if metric.Success {
			t.Error("expected success to be false")
		}
		if metric.ResponseSize != 0 {
			t.Errorf("expected response size 0, got %d", metric.ResponseSize)
		}
	})

	t.Run("does not record when collector is nil", func(t *testing.T) {
		v, err := New(Config{BaseURL: "https://api.example.com"})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		req, _ := v.newRequest("/test", "GET", nil)
		res := &Response{StatusCode: 200, success: true}

		v.recordMetrics(context.Background(), req, res, 100*time.Millisecond, nil)
	})

	t.Run("records metrics with error", func(t *testing.T) {
		collector := &mockMetricsCollector{}
		v, err := New(Config{
			BaseURL:          "https://api.example.com",
			MetricsCollector: collector,
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		req, err := v.newRequest("/fail", "GET", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		testErr := &ResponseError{Err: http.ErrServerClosed}
		v.recordMetrics(context.Background(), req, nil, 5*time.Second, testErr)

		if len(collector.requests) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(collector.requests))
		}

		metric := collector.requests[0]
		if metric.Error == nil {
			t.Error("expected error to be set")
		}
		if metric.Success {
			t.Error("expected success to be false")
		}
	})
}

func TestRecordMetricsWithFallback(t *testing.T) {
	t.Run("uses fallback with nil request", func(t *testing.T) {
		collector := &mockMetricsCollector{}
		v, err := New(Config{
			BaseURL:          "https://api.example.com",
			MetricsCollector: collector,
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		res := &Response{
			StatusCode: 404,
			Data:       []byte(`{"error": "not found"}`),
			success:    false,
		}

		method := "GET"
		url := "https://api.example.com/missing"
		v.recordMetricsWithFallback(context.Background(), method, url, nil, res, 200*time.Millisecond, nil)

		if len(collector.requests) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(collector.requests))
		}

		metric := collector.requests[0]
		if metric.Method != method {
			t.Errorf("expected method %s, got %s", method, metric.Method)
		}
		if metric.FullURL != url {
			t.Errorf("expected URL %s, got %s", url, metric.FullURL)
		}
		if metric.StatusCode != 404 {
			t.Errorf("expected status 404, got %d", metric.StatusCode)
		}
	})

	t.Run("uses request data when available", func(t *testing.T) {
		collector := &mockMetricsCollector{}
		v, err := New(Config{
			BaseURL:          "https://api.example.com",
			MetricsCollector: collector,
		})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		req, err := v.newRequest("/users", "POST", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		res := &Response{
			StatusCode: 201,
			Data:       []byte(`{"id": 456}`),
			success:    true,
		}

		v.recordMetricsWithFallback(context.Background(), "POST", "fallback-url", req, res, 100*time.Millisecond, nil)

		if len(collector.requests) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(collector.requests))
		}

		metric := collector.requests[0]
		if metric.FullURL == "fallback-url" {
			t.Error("expected to use request URL, not fallback")
		}
		if metric.StatusCode != 201 {
			t.Errorf("expected status 201, got %d", metric.StatusCode)
		}
	})

	t.Run("does not record when collector is nil", func(t *testing.T) {
		v, err := New(Config{BaseURL: "https://api.example.com"})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		v.recordMetricsWithFallback(context.Background(), "GET", "http://test.com", nil, nil, 0, nil)
	})
}

func TestNormalizeURL(t *testing.T) {
	t.Run("normalizes complete URL", func(t *testing.T) {
		v, err := New(Config{BaseURL: "https://api.example.com"})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		req, err := v.newRequest("/users/123", "GET", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		normalized := v.normalizeURL(req)
		expected := "https://api.example.com/users/123"
		if normalized != expected {
			t.Errorf("expected %s, got %s", expected, normalized)
		}
	})

	t.Run("returns empty for nil request", func(t *testing.T) {
		v, err := New(Config{BaseURL: "https://api.example.com"})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		normalized := v.normalizeURL(nil)
		if normalized != "" {
			t.Errorf("expected empty string, got %s", normalized)
		}
	})

	t.Run("handles path only", func(t *testing.T) {
		v, err := New(Config{})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		req, err := v.newRequest("/path/only", "GET", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		normalized := v.normalizeURL(req)
		if normalized != "/path/only" {
			t.Errorf("expected /path/only, got %s", normalized)
		}
	})
}

