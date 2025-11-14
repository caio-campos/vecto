package vecto

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	tests := []struct {
		name     string
		attempt  int
		config   *RetryConfig
		expected time.Duration
	}{
		{
			name:    "attempt 1",
			attempt: 1,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
			},
			expected: time.Second,
		},
		{
			name:    "attempt 2",
			attempt: 2,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
			},
			expected: 2 * time.Second,
		},
		{
			name:    "attempt 3",
			attempt: 3,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
			},
			expected: 4 * time.Second,
		},
		{
			name:    "attempt 4",
			attempt: 4,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
			},
			expected: 8 * time.Second,
		},
		{
			name:    "max wait time exceeded",
			attempt: 10,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: 10 * time.Second,
			},
			expected: 10 * time.Second,
		},
		{
			name:    "nil config",
			attempt: 1,
			config:  nil,
			expected: time.Second,
		},
		{
			name:    "zero wait time",
			attempt: 1,
			config: &RetryConfig{
				WaitTime: 0,
			},
			expected: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExponentialBackoff(tt.attempt, tt.config)
			if result != tt.expected {
				t.Errorf("ExponentialBackoff() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLinearBackoff(t *testing.T) {
	tests := []struct {
		name     string
		attempt  int
		config   *RetryConfig
		expected time.Duration
	}{
		{
			name:    "attempt 1",
			attempt: 1,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
			},
			expected: time.Second,
		},
		{
			name:    "attempt 2",
			attempt: 2,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
			},
			expected: 2 * time.Second,
		},
		{
			name:    "attempt 5",
			attempt: 5,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
			},
			expected: 5 * time.Second,
		},
		{
			name:    "max wait time exceeded",
			attempt: 20,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: 10 * time.Second,
			},
			expected: 10 * time.Second,
		},
		{
			name:    "nil config",
			attempt: 1,
			config:  nil,
			expected: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LinearBackoff(tt.attempt, tt.config)
			if result != tt.expected {
				t.Errorf("LinearBackoff() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFixedBackoff(t *testing.T) {
	tests := []struct {
		name     string
		attempt  int
		config   *RetryConfig
		expected time.Duration
	}{
		{
			name:    "attempt 1",
			attempt: 1,
			config: &RetryConfig{
				WaitTime: time.Second,
			},
			expected: time.Second,
		},
		{
			name:    "attempt 5",
			attempt: 5,
			config: &RetryConfig{
				WaitTime: 2 * time.Second,
			},
			expected: 2 * time.Second,
		},
		{
			name:    "nil config",
			attempt: 1,
			config:  nil,
			expected: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixedBackoff(tt.attempt, tt.config)
			if result != tt.expected {
				t.Errorf("FixedBackoff() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultRetryCondition(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		err      error
		expected bool
	}{
		{
			name: "500 internal server error",
			response: &Response{
				StatusCode: 500,
			},
			err:      nil,
			expected: true,
		},
		{
			name: "502 bad gateway",
			response: &Response{
				StatusCode: 502,
			},
			err:      nil,
			expected: true,
		},
		{
			name: "503 service unavailable",
			response: &Response{
				StatusCode: 503,
			},
			err:      nil,
			expected: true,
		},
		{
			name: "429 too many requests",
			response: &Response{
				StatusCode: 429,
			},
			err:      nil,
			expected: true,
		},
		{
			name: "200 ok",
			response: &Response{
				StatusCode: 200,
			},
			err:      nil,
			expected: false,
		},
		{
			name: "404 not found",
			response: &Response{
				StatusCode: 404,
			},
			err:      nil,
			expected: false,
		},
		{
			name:     "network error",
			response: nil,
			err: &net.OpError{
				Op:  "dial",
				Err: errors.New("connection refused"),
			},
			expected: true,
		},
		{
			name:     "url error",
			response: nil,
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: errors.New("timeout"),
			},
			expected: true,
		},
		{
			name:     "generic error",
			response: nil,
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "nil response and no error",
			response: nil,
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultRetryCondition(tt.response, tt.err)
			if result != tt.expected {
				t.Errorf("DefaultRetryCondition() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "net.OpError",
			err: &net.OpError{
				Op:  "dial",
				Err: errors.New("connection refused"),
			},
			expected: true,
		},
		{
			name: "url.Error with timeout",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &timeoutError{},
			},
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("isNetworkError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestParseRetryAfterHeader(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		expected time.Duration
	}{
		{
			name: "retry after in seconds",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Retry-After": []string{"5"},
					},
				},
			},
			expected: 5 * time.Second,
		},
		{
			name: "no retry after header",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{},
				},
			},
			expected: 0,
		},
		{
			name:     "nil response",
			response: nil,
			expected: 0,
		},
		{
			name: "nil raw response",
			response: &Response{
				RawResponse: nil,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryAfterHeader(tt.response)
			if result != tt.expected {
				t.Errorf("parseRetryAfterHeader() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		attempt  int
		config   *RetryConfig
		response *Response
		err      error
		expected bool
	}{
		{
			name:    "first attempt with 500 error",
			attempt: 1,
			config: &RetryConfig{
				MaxAttempts: 3,
			},
			response: &Response{
				StatusCode: 500,
			},
			err:      nil,
			expected: true,
		},
		{
			name:    "max attempts reached",
			attempt: 3,
			config: &RetryConfig{
				MaxAttempts: 3,
			},
			response: &Response{
				StatusCode: 500,
			},
			err:      nil,
			expected: false,
		},
		{
			name:    "success response",
			attempt: 1,
			config: &RetryConfig{
				MaxAttempts: 3,
			},
			response: &Response{
				StatusCode: 200,
			},
			err:      nil,
			expected: false,
		},
		{
			name:    "nil config",
			attempt: 1,
			config:  nil,
			response: &Response{
				StatusCode: 500,
			},
			err:      nil,
			expected: false,
		},
		{
			name:    "custom retry condition - always retry",
			attempt: 1,
			config: &RetryConfig{
				MaxAttempts: 3,
				RetryCondition: func(res *Response, err error) bool {
					return true
				},
			},
			response: &Response{
				StatusCode: 200,
			},
			err:      nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.attempt, tt.config, tt.response, tt.err)
			if result != tt.expected {
				t.Errorf("shouldRetry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRetryWaitTime(t *testing.T) {
	tests := []struct {
		name     string
		attempt  int
		config   *RetryConfig
		response *Response
		err      error
		validate func(t *testing.T, result time.Duration)
	}{
		{
			name:    "exponential backoff",
			attempt: 2,
			config: &RetryConfig{
				WaitTime:    time.Second,
				MaxWaitTime: time.Minute,
				Backoff:     ExponentialBackoff,
			},
			validate: func(t *testing.T, result time.Duration) {
				if result != 2*time.Second {
					t.Errorf("expected 2 seconds, got %v", result)
				}
			},
		},
		{
			name:    "custom retry after",
			attempt: 1,
			config: &RetryConfig{
				RetryAfter: func(attempt int, res *Response, err error) time.Duration {
					return 5 * time.Second
				},
			},
			validate: func(t *testing.T, result time.Duration) {
				if result != 5*time.Second {
					t.Errorf("expected 5 seconds, got %v", result)
				}
			},
		},
		{
			name:    "respect retry-after header",
			attempt: 1,
			config: &RetryConfig{
				WaitTime:                time.Second,
				RespectRetryAfterHeader: true,
			},
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Retry-After": []string{"3"},
					},
				},
			},
			validate: func(t *testing.T, result time.Duration) {
				if result != 3*time.Second {
					t.Errorf("expected 3 seconds, got %v", result)
				}
			},
		},
		{
			name:    "nil config",
			attempt: 1,
			config:  nil,
			validate: func(t *testing.T, result time.Duration) {
				if result != time.Second {
					t.Errorf("expected 1 second, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRetryWaitTime(tt.attempt, tt.config, tt.response, tt.err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestValidateRetryConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *RetryConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: &RetryConfig{
				MaxAttempts: 3,
				WaitTime:    time.Second,
				MaxWaitTime: 10 * time.Second,
			},
			expectErr: false,
		},
		{
			name:      "nil config",
			config:    nil,
			expectErr: true,
		},
		{
			name: "negative max attempts",
			config: &RetryConfig{
				MaxAttempts: -2,
				WaitTime:    time.Second,
			},
			expectErr: true,
		},
		{
			name: "negative wait time",
			config: &RetryConfig{
				MaxAttempts: 3,
				WaitTime:    -time.Second,
			},
			expectErr: true,
		},
		{
			name: "negative max wait time",
			config: &RetryConfig{
				MaxAttempts: 3,
				WaitTime:    time.Second,
				MaxWaitTime: -time.Second,
			},
			expectErr: true,
		},
		{
			name: "wait time greater than max wait time",
			config: &RetryConfig{
				MaxAttempts: 3,
				WaitTime:    10 * time.Second,
				MaxWaitTime: 5 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "unlimited retries (-1)",
			config: &RetryConfig{
				MaxAttempts: -1,
				WaitTime:    time.Second,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRetryConfig(tt.config)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateRetryConfig() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRetry_Integration(t *testing.T) {
	t.Run("retry on 500 error", func(t *testing.T) {
		attempts := 0
		mockClient := &mockRetryClient{
			doFunc: func(ctx context.Context, req *Request) (*Response, error) {
				attempts++
				if attempts < 3 {
					return &Response{
						StatusCode: 500,
						Data:       []byte("error"),
					}, nil
				}
				return &Response{
					StatusCode: 200,
					Data:       []byte("success"),
					success:    true,
				}, nil
			},
		}

		v := &Vecto{
			client: mockClient,
			logger: newNoopLogger(),
			config: Config{
				Retry: &RetryConfig{
					MaxAttempts: 5,
					WaitTime:    10 * time.Millisecond,
					MaxWaitTime: 100 * time.Millisecond,
					Backoff:     FixedBackoff,
				},
			},
		}

		ctx := context.Background()
		req, _ := newRequestBuilder("https://example.com", "GET").Build()
		
		res, err := v.executeWithRetry(ctx, req, v.config.Retry)
		if err != nil {
			t.Errorf("executeWithRetry() error = %v", err)
		}

		if res == nil || res.StatusCode != 200 {
			t.Error("expected successful response")
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("max attempts reached", func(t *testing.T) {
		attempts := 0
		mockClient := &mockRetryClient{
			doFunc: func(ctx context.Context, req *Request) (*Response, error) {
				attempts++
				return &Response{
					StatusCode: 500,
					Data:       []byte("error"),
				}, nil
			},
		}

		v := &Vecto{
			client: mockClient,
			logger: newNoopLogger(),
			config: Config{
				Retry: &RetryConfig{
					MaxAttempts: 3,
					WaitTime:    10 * time.Millisecond,
					Backoff:     FixedBackoff,
				},
			},
		}

		ctx := context.Background()
		req, _ := newRequestBuilder("https://example.com", "GET").Build()
		
		res, err := v.executeWithRetry(ctx, req, v.config.Retry)
		
		if res == nil || res.StatusCode != 500 {
			t.Error("expected 500 response")
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}

		if err != nil {
			t.Logf("Returned with error (optional): %v", err)
		}
	})

	t.Run("no retry on success", func(t *testing.T) {
		attempts := 0
		mockClient := &mockRetryClient{
			doFunc: func(ctx context.Context, req *Request) (*Response, error) {
				attempts++
				return &Response{
					StatusCode: 200,
					Data:       []byte("success"),
					success:    true,
				}, nil
			},
		}

		v := &Vecto{
			client: mockClient,
			logger: newNoopLogger(),
			config: Config{
				Retry: &RetryConfig{
					MaxAttempts: 3,
					WaitTime:    time.Second,
				},
			},
		}

		ctx := context.Background()
		req, _ := newRequestBuilder("https://example.com", "GET").Build()
		
		res, err := v.executeWithRetry(ctx, req, v.config.Retry)
		if err != nil {
			t.Errorf("executeWithRetry() error = %v", err)
		}

		if res == nil || res.StatusCode != 200 {
			t.Error("expected successful response")
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockClient := &mockRetryClient{
			doFunc: func(ctx context.Context, req *Request) (*Response, error) {
				return &Response{
					StatusCode: 500,
				}, nil
			},
		}

		v := &Vecto{
			client: mockClient,
			logger: newNoopLogger(),
			config: Config{
				Retry: &RetryConfig{
					MaxAttempts: 5,
					WaitTime:    100 * time.Millisecond,
					Backoff:     FixedBackoff,
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		req, _ := newRequestBuilder("https://example.com", "GET").Build()
		
		_, err := v.executeWithRetry(ctx, req, v.config.Retry)
		if err == nil {
			t.Error("expected context cancellation error")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}

type mockRetryClient struct {
	doFunc func(ctx context.Context, req *Request) (*Response, error)
}

func (m *mockRetryClient) Do(ctx context.Context, req *Request) (*Response, error) {
	return m.doFunc(ctx, req)
}

func BenchmarkExponentialBackoff(b *testing.B) {
	config := &RetryConfig{
		WaitTime:    time.Second,
		MaxWaitTime: time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExponentialBackoff(3, config)
	}
}

func BenchmarkDefaultRetryCondition(b *testing.B) {
	res := &Response{
		StatusCode: 500,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultRetryCondition(res, nil)
	}
}

func ExampleRetryConfig() {
	config := RetryConfig{
		MaxAttempts: 3,
		WaitTime:    time.Second,
		MaxWaitTime: 10 * time.Second,
		Backoff:     ExponentialBackoff,
		OnRetry: func(attempt int, err error) {
			fmt.Printf("Retrying... Attempt %d\n", attempt)
		},
	}

	_ = config
}

func ExampleExponentialBackoff() {
	config := &RetryConfig{
		WaitTime:    time.Second,
		MaxWaitTime: time.Minute,
	}

	wait1 := ExponentialBackoff(1, config)
	wait2 := ExponentialBackoff(2, config)
	wait3 := ExponentialBackoff(3, config)

	fmt.Printf("Attempt 1: %v\n", wait1)
	fmt.Printf("Attempt 2: %v\n", wait2)
	fmt.Printf("Attempt 3: %v\n", wait3)
}

