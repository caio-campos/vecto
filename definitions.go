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

// RequestMetrics contains metrics information for an HTTP request.
type RequestMetrics struct {
	// Method is the HTTP method (GET, POST, etc.)
	Method string
	
	// URL is the normalized request URL (host + path, without query params).
	URL string
	
	// FullURL is the complete request URL including query params.
	FullURL string
	
	// Duration is the total request duration.
	Duration time.Duration
	
	// StatusCode is the HTTP status code (0 if request failed before receiving response).
	StatusCode int
	
	// Error is the error that occurred, if any.
	Error error
	
	// RequestSize is the size of the request body in bytes (0 if no body).
	RequestSize int64
	
	// ResponseSize is the size of the response body in bytes (0 if no response).
	ResponseSize int64
	
	// Success indicates if the request was considered successful.
	Success bool
}

// MetricsCollector is the interface for collecting HTTP request metrics.
// Implementations should be thread-safe as this interface may be called
// concurrently from multiple goroutines.
type MetricsCollector interface {
	// RecordRequest records metrics for a single HTTP request.
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
