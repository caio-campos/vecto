package vecto

import (
	"context"
	"net/http"
	"time"
)

type CertificateConfig struct {
	Cert string
	Key  string
}

type AdapterFunc func(req *Request) (res *Response, err error)

type RequestTransformFunc func(req *Request) (data []byte, err error)

type ValidateStatusFunc func(res *Response) bool

type RequestCompletedCallback func(ev RequestCompletedEvent)

// RequestMetrics contains all metrics information for an HTTP request.
// This structure follows observability best practices by grouping related metrics
// and providing normalized labels to avoid high cardinality issues.
type RequestMetrics struct {
	// Method is the HTTP method (GET, POST, etc.)
	Method string
	
	// URL is the normalized request URL (host + path pattern, not full URL with query params)
	// This helps avoid cardinality explosion in metrics systems
	URL string
	
	// FullURL is the complete request URL (including query params) - use with caution
	// Only include if needed for specific use cases
	FullURL string
	
	// Duration is the total request duration
	Duration time.Duration
	
	// StatusCode is the HTTP status code (0 if request failed before receiving response)
	StatusCode int
	
	// Error is the error that occurred, if any
	Error error
	
	// RequestSize is the size of the request body in bytes (0 if no body)
	RequestSize int64
	
	// ResponseSize is the size of the response body in bytes (0 if no response)
	ResponseSize int64
	
	// Success indicates if the request was considered successful
	// (based on ValidateStatus function)
	Success bool
}

// MetricsCollector is the interface for collecting HTTP request metrics.
// Implementations should be thread-safe as this interface may be called
// concurrently from multiple goroutines.
//
// Best practices for implementations:
//   - Use histograms for duration metrics with appropriate buckets
//   - Normalize URLs to avoid high cardinality (use URL field, not FullURL)
//   - Group metrics by method, status code, and error type
//   - Consider rate limiting or sampling for high-volume scenarios
type MetricsCollector interface {
	// RecordRequest records all metrics for a single HTTP request.
	// This method should be fast and non-blocking to avoid impacting request performance.
	// Consider using async processing or buffering for expensive operations.
	RecordRequest(ctx context.Context, metrics RequestMetrics)
}

type Config struct {
	BaseURL                string
	Timeout                time.Duration
	Headers                map[string]string
	Certificates           []CertificateConfig
	HTTPTransport          *http.Transport
	Adapter                AdapterFunc
	RequestTransform       RequestTransformFunc
	ValidateStatus         ValidateStatusFunc
	InsecureSkipVerify     bool
	Logger                 Logger
	MetricsCollector       MetricsCollector
	MaxResponseBodySize    int64
	MaxConcurrentCallbacks int
	CallbackTimeout        time.Duration
}

type Client interface {
	Do(ctx context.Context, req *Request) (res *Response, err error)
}

type requestEvents struct {
	completed []RequestCompletedCallback
}

type RequestCompletedEvent struct {
	response *Response
}

func (r *RequestCompletedEvent) Response() *Response {
	return r.response
}

type RequestOptions struct {
	Data             interface{}
	Headers          map[string]string
	Params           map[string]any
	RequestTransform RequestTransformFunc
}
