package vecto

import (
	"context"
	"fmt"
	"time"
)

type requestHandler struct {
	vecto *Vecto
}

func newRequestHandler(v *Vecto) *requestHandler {
	return &requestHandler{vecto: v}
}

func (h *requestHandler) handleRequestError(
	ctx context.Context,
	req *Request,
	method string,
	startTime time.Time,
	err error,
) (*Response, error) {
	duration := time.Since(startTime)

	if !h.vecto.logger.IsNoop() {
		h.vecto.logger.Error(ctx, "http request failed", map[string]interface{}{
			"url":    req.FullUrl(),
			"method": method,
			"error":  err.Error(),
		})
	}

	h.vecto.recordMetrics(ctx, req, nil, duration, err)
	return nil, fmt.Errorf("http request failed: %w", err)
}

func (h *requestHandler) handleCircuitBreakerError(
	ctx context.Context,
	req *Request,
	cbKey string,
	breaker *CircuitBreaker,
	startTime time.Time,
	err error,
) (*Response, error) {
	duration := time.Since(startTime)

	if !h.vecto.logger.IsNoop() {
		h.vecto.logger.Warn(ctx, "request blocked by circuit breaker", map[string]interface{}{
			"url":    req.FullUrl(),
			"method": req.Method(),
			"key":    cbKey,
			"state":  breaker.GetState().String(),
		})
	}

	h.vecto.recordMetrics(ctx, req, nil, duration, err)
	return nil, err
}

func (h *requestHandler) getOrSetCircuitBreakerKey(req *Request) string {
	req.mu.RLock()
	if req.cbKeyCached {
		key := req.cbKey
		req.mu.RUnlock()
		return key
	}
	req.mu.RUnlock()

	req.mu.Lock()
	defer req.mu.Unlock()

	if !req.cbKeyCached {
		req.cbKey = h.vecto.getCircuitBreakerKey(req)
		req.cbKeyCached = true
	}

	return req.cbKey
}

func (h *requestHandler) executeRequest(
	ctx context.Context,
	req *Request,
	retryConfig *RetryConfig,
	breaker *CircuitBreaker,
) (*Response, error) {
	if retryConfig != nil && shouldUseRetry(breaker) {
		return h.vecto.executeWithRetry(ctx, req, retryConfig)
	}
	return h.vecto.client.Do(ctx, req)
}

