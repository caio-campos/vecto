//go:build example_metrics_collector
// +build example_metrics_collector

// This is a standalone example program. Each example file has its own main function
// and should be run individually: go run -tags example_metrics_collector metrics_collector.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caio-campos/vecto"
)

// PrometheusMetricsCollector is an example implementation of MetricsCollector
// that demonstrates how to integrate with Prometheus or similar metrics systems.
type PrometheusMetricsCollector struct {
	// In a real implementation, you would have Prometheus metrics here:
	// requestDuration *prometheus.HistogramVec
	// requestCount     *prometheus.CounterVec
	// requestSize      *prometheus.HistogramVec
	// responseSize     *prometheus.HistogramVec
}

func NewPrometheusMetricsCollector() *PrometheusMetricsCollector {
	return &PrometheusMetricsCollector{
		// Initialize Prometheus metrics here
	}
}

func (p *PrometheusMetricsCollector) RecordRequest(ctx context.Context, metrics vecto.RequestMetrics) {
	// Extract labels for Prometheus metrics
	labels := map[string]string{
		"method":      metrics.Method,
		"url":         metrics.URL, // Use normalized URL to avoid cardinality explosion
		"status_code": fmt.Sprintf("%d", metrics.StatusCode),
		"success":     fmt.Sprintf("%v", metrics.Success),
	}

	if metrics.Error != nil {
		labels["error"] = metrics.Error.Error()
		labels["error_type"] = getErrorType(metrics.Error)
	} else {
		labels["error"] = ""
		labels["error_type"] = ""
	}

	// Record duration histogram
	// p.requestDuration.WithLabelValues(labels...).Observe(metrics.Duration.Seconds())

	// Record request count
	// p.requestCount.WithLabelValues(labels...).Inc()

	// Record request size histogram
	// if metrics.RequestSize > 0 {
	//     p.requestSize.WithLabelValues(labels...).Observe(float64(metrics.RequestSize))
	// }

	// Record response size histogram
	// if metrics.ResponseSize > 0 {
	//     p.responseSize.WithLabelValues(labels...).Observe(float64(metrics.ResponseSize))
	// }

	// Example: log metrics for demonstration
	log.Printf("Metrics: method=%s url=%s duration=%v status=%d success=%v error=%v",
		metrics.Method,
		metrics.URL,
		metrics.Duration,
		metrics.StatusCode,
		metrics.Success,
		metrics.Error,
	)
}

func getErrorType(err error) string {
	if err == nil {
		return ""
	}
	// Categorize error types (network, timeout, etc.)
	return "unknown"
}

// NoOpMetricsCollector is a no-op implementation useful for testing or when metrics are disabled.
type NoOpMetricsCollector struct{}

func (n *NoOpMetricsCollector) RecordRequest(ctx context.Context, metrics vecto.RequestMetrics) {
	// No-op: do nothing
}

func main() {
	ExampleMetricsCollectorUsage()
}

// ExampleUsage demonstrates how to use MetricsCollector with vecto client
func ExampleMetricsCollectorUsage() {
	// Example usage
	collector := NewPrometheusMetricsCollector()

	client, err := vecto.New(vecto.Config{
		BaseURL:          "https://api.example.com",
		MetricsCollector: collector,
		Timeout:          30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	res, err := client.Get(ctx, "/users/123", nil)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	log.Printf("Response status: %d", res.StatusCode)
}
