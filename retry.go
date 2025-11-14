package vecto

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/url"
	"time"
)

// RetryConditionFunc defines when a retry should be attempted.
// Returns true if the request should be retried.
type RetryConditionFunc func(res *Response, err error) bool

// RetryAfterFunc defines custom logic to determine the wait time before next retry.
// Returns the duration to wait before the next retry attempt.
type RetryAfterFunc func(attempt int, res *Response, err error) time.Duration

// BackoffFunc defines the backoff strategy.
// Returns the wait time for the given attempt number.
type BackoffFunc func(attempt int, config *RetryConfig) time.Duration

// RetryConfig configures the retry mechanism.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts (0 = no retries, -1 = unlimited).
	MaxAttempts int

	// WaitTime is the initial wait time between retries.
	WaitTime time.Duration

	// MaxWaitTime is the maximum wait time between retries.
	MaxWaitTime time.Duration

	// Backoff defines the backoff strategy (defaults to ExponentialBackoff).
	Backoff BackoffFunc

	// RetryCondition determines when to retry (defaults to retry on 5xx and network errors).
	RetryCondition RetryConditionFunc

	// RetryAfter allows custom wait time calculation (overrides Backoff if provided).
	RetryAfter RetryAfterFunc

	// RespectRetryAfterHeader respects the Retry-After header in responses.
	RespectRetryAfterHeader bool

	// OnRetry is called before each retry attempt.
	OnRetry func(attempt int, err error)
}

// ExponentialBackoff implements exponential backoff strategy (2^n * WaitTime).
func ExponentialBackoff(attempt int, config *RetryConfig) time.Duration {
	if config == nil || config.WaitTime <= 0 {
		return time.Second
	}

	wait := config.WaitTime * time.Duration(math.Pow(2, float64(attempt-1)))

	if config.MaxWaitTime > 0 && wait > config.MaxWaitTime {
		return config.MaxWaitTime
	}

	return wait
}

// LinearBackoff implements linear backoff strategy (attempt * WaitTime).
func LinearBackoff(attempt int, config *RetryConfig) time.Duration {
	if config == nil || config.WaitTime <= 0 {
		return time.Second
	}

	wait := config.WaitTime * time.Duration(attempt)

	if config.MaxWaitTime > 0 && wait > config.MaxWaitTime {
		return config.MaxWaitTime
	}

	return wait
}

// FixedBackoff implements fixed delay backoff strategy (always WaitTime).
func FixedBackoff(attempt int, config *RetryConfig) time.Duration {
	if config == nil || config.WaitTime <= 0 {
		return time.Second
	}

	return config.WaitTime
}

// DefaultRetryCondition is the default condition for retrying requests.
// Retries on 5xx status codes, 429 (Too Many Requests), and network errors.
func DefaultRetryCondition(res *Response, err error) bool {
	if err != nil {
		if isNetworkError(err) {
			return true
		}
		if _, ok := err.(*url.Error); ok {
			return true
		}
		return false
	}

	if res == nil {
		return false
	}

	return res.StatusCode >= 500 || res.StatusCode == 429
}

// isNetworkError checks if the error is a network-related error.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	if _, ok := err.(net.Error); ok {
		return true
	}

	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Temporary() || urlErr.Timeout()
	}

	return false
}

// parseRetryAfterHeader parses the Retry-After header.
// Returns the duration to wait, or 0 if the header is not present or invalid.
func parseRetryAfterHeader(res *Response) time.Duration {
	if res == nil || res.RawResponse == nil {
		return 0
	}

	retryAfter := res.Header("Retry-After")
	if retryAfter == "" {
		return 0
	}

	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}

	if t, err := time.Parse(time.RFC1123, retryAfter); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	return 0
}

// shouldRetry determines if a request should be retried based on the retry config.
func shouldRetry(attempt int, config *RetryConfig, res *Response, err error) bool {
	if config == nil {
		return false
	}

	if config.MaxAttempts >= 0 && attempt >= config.MaxAttempts {
		return false
	}

	condition := config.RetryCondition
	if condition == nil {
		condition = DefaultRetryCondition
	}

	return condition(res, err)
}

// getRetryWaitTime calculates the wait time before the next retry.
func getRetryWaitTime(attempt int, config *RetryConfig, res *Response, err error) time.Duration {
	if config == nil {
		return time.Second
	}

	if config.RetryAfter != nil {
		return config.RetryAfter(attempt, res, err)
	}

	if config.RespectRetryAfterHeader && res != nil {
		if retryAfter := parseRetryAfterHeader(res); retryAfter > 0 {
			if config.MaxWaitTime > 0 && retryAfter > config.MaxWaitTime {
				return config.MaxWaitTime
			}
			return retryAfter
		}
	}

	backoff := config.Backoff
	if backoff == nil {
		backoff = ExponentialBackoff
	}

	return backoff(attempt, config)
}

// executeWithRetry executes a request with retry logic.
func (v *Vecto) executeWithRetry(
	ctx context.Context,
	req *Request,
	retryConfig *RetryConfig,
) (*Response, error) {
	if retryConfig == nil || retryConfig.MaxAttempts == 0 {
		return v.client.Do(ctx, req)
	}

	var lastResponse *Response
	var lastErr error
	attempt := 0

	for {
		attempt++

		res, err := v.client.Do(ctx, req)
		lastResponse = res
		lastErr = err

		if err == nil && res != nil && res.success {
			return res, nil
		}

		if !shouldRetry(attempt, retryConfig, res, err) {
			break
		}

		if ctx.Err() != nil {
			return res, ctx.Err()
		}

		waitTime := getRetryWaitTime(attempt, retryConfig, res, err)

		if retryConfig.OnRetry != nil {
			retryConfig.OnRetry(attempt, err)
		}

		if !v.logger.IsNoop() {
			v.logger.Warn(ctx, "retrying request", map[string]interface{}{
				"attempt":   attempt,
				"wait_time": waitTime.String(),
				"url":       req.FullUrl(),
				"error":     formatErrorForLog(err),
			})
		}

		select {
		case <-ctx.Done():
			return lastResponse, ctx.Err()
		case <-time.After(waitTime):
		}
	}

	if lastErr != nil {
		return lastResponse, fmt.Errorf("request failed after %d attempts: %w", attempt, lastErr)
	}

	return lastResponse, nil
}

// formatErrorForLog formats an error for logging.
func formatErrorForLog(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ValidateRetryConfig validates the retry configuration.
func ValidateRetryConfig(config *RetryConfig) error {
	if config == nil {
		return fmt.Errorf("retry config is nil")
	}

	if config.MaxAttempts < -1 {
		return fmt.Errorf("max attempts cannot be less than -1")
	}

	if config.WaitTime < 0 {
		return fmt.Errorf("wait time cannot be negative")
	}

	if config.MaxWaitTime < 0 {
		return fmt.Errorf("max wait time cannot be negative")
	}

	if config.MaxWaitTime > 0 && config.WaitTime > config.MaxWaitTime {
		return fmt.Errorf("wait time cannot be greater than max wait time")
	}

	return nil
}
