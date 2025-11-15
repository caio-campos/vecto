package vecto

import (
	"context"
	"time"
)

func (v *Vecto) recordMetrics(ctx context.Context, req *Request, res *Response, duration time.Duration, err error) {
	if v.config.MetricsCollector == nil {
		return
	}

	var normalizedURL, fullURL string
	var method string
	var requestSize int64
	var statusCode int
	var responseSize int64
	var success bool

	if req != nil {
		method = req.Method()
		fullURL = req.FullUrl()
		normalizedURL = v.normalizeURL(req)

		if req.RawRequest() != nil && req.RawRequest().Body != nil {
			if req.RawRequest().ContentLength > 0 {
				requestSize = req.RawRequest().ContentLength
			}
		}
	}

	if res != nil {
		statusCode = res.StatusCode
		responseSize = int64(len(res.Data))
		success = res.success
	}

	metrics := RequestMetrics{
		Method:       method,
		URL:          normalizedURL,
		FullURL:      fullURL,
		Duration:     duration,
		StatusCode:   statusCode,
		Error:        err,
		RequestSize:  requestSize,
		ResponseSize: responseSize,
		Success:      success,
	}

	v.config.MetricsCollector.RecordRequest(ctx, metrics)
}

func (v *Vecto) recordMetricsWithFallback(ctx context.Context, method, url string, req *Request, res *Response, duration time.Duration, err error) {
	if v.config.MetricsCollector == nil {
		return
	}

	var normalizedURL, fullURL string
	var requestSize int64
	var statusCode int
	var responseSize int64
	var success bool

	if req != nil {
		fullURL = req.FullUrl()
		normalizedURL = v.normalizeURL(req)
		if req.RawRequest() != nil && req.RawRequest().Body != nil {
			if req.RawRequest().ContentLength > 0 {
				requestSize = req.RawRequest().ContentLength
			}
		}
	} else {
		fullURL = url
		normalizedURL = url
	}

	if res != nil {
		statusCode = res.StatusCode
		responseSize = int64(len(res.Data))
		success = res.success
	}

	metrics := RequestMetrics{
		Method:       method,
		URL:          normalizedURL,
		FullURL:      fullURL,
		Duration:     duration,
		StatusCode:   statusCode,
		Error:        err,
		RequestSize:  requestSize,
		ResponseSize: responseSize,
		Success:      success,
	}

	v.config.MetricsCollector.RecordRequest(ctx, metrics)
}

func (v *Vecto) normalizeURL(req *Request) string {
	if req == nil {
		return ""
	}

	scheme := req.Scheme()
	host := req.Host()
	path := req.Path()

	if scheme == "" && host == "" {
		return path
	}

	if host == "" {
		return path
	}

	return scheme + "://" + host + path
}
