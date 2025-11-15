package vecto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type PetMockData struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Owner MockDataOwner `json:"owner"`
}

type MockDataOwner struct {
	Login string `json:"login"`
}

var supportedMethods = map[string]bool{
	http.MethodPost:    true,
	http.MethodGet:     true,
	http.MethodPatch:   true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

type handleReqMethod func(ctx context.Context, url string, options *RequestOptions) (res *Response, err error)

func TestHeaderConfiguration(t *testing.T) {
	customHeaderExpectedValue := "custom"
	anotherHeaderOriginalValue := "another"
	reqCustomHeaderExpectedValue := "req-custom"
	anotherHeaderExpectedValue := "another-replaced"

	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
		Headers: map[string]string{
			"x-custom":  customHeaderExpectedValue,
			"x-another": anotherHeaderOriginalValue,
		},
	})

	res, _ := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{
		Headers: map[string]string{
			"x-req-custom": reqCustomHeaderExpectedValue,
			"x-another":    anotherHeaderExpectedValue,
		},
	})

	xCustomHeaderValue, xCustomHeaderOk := res.request.headers["x-custom"]
	xAnotherHeaderValue, xAnotherHeaderOk := res.request.headers["x-another"]
	xReqCustomHeaderValue, xReqCustomHeaderOk := res.request.headers["x-req-custom"]

	assert.True(t, xCustomHeaderOk)
	assert.True(t, xAnotherHeaderOk)
	assert.True(t, xReqCustomHeaderOk)

	assert.Equal(t, customHeaderExpectedValue, xCustomHeaderValue)
	assert.Equal(t, anotherHeaderExpectedValue, xAnotherHeaderValue)
	assert.Equal(t, reqCustomHeaderExpectedValue, xReqCustomHeaderValue)

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestRequestMethods(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
	})

	methodsMap := map[string]handleReqMethod{
		http.MethodPost:    vecto.Post,
		http.MethodGet:     vecto.Get,
		http.MethodPatch:   vecto.Patch,
		http.MethodPut:     vecto.Put,
		http.MethodDelete:  vecto.Delete,
		http.MethodOptions: vecto.Options,
	}

	for method, funct := range methodsMap {
		res, err := funct(context.Background(), "/test/methods", nil)

		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode, method)
	}

	res, err := vecto.Request(context.Background(), "/test/methods", http.MethodTrace, nil)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode, nil)
}

func TestRequestTimeoutConfiguration(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
		Timeout: 3000,
	})

	res, err := vecto.Post(context.Background(), "/test/slow", nil)
	assert.NotNil(t, err)
	assert.Nil(t, res)
}

func TestConfigHeaderIsolation(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	sharedHeaders := map[string]string{
		"x-shared": "value",
	}

	first, err := New(Config{
		BaseURL: srv.URL,
		Headers: sharedHeaders,
	})
	assert.NoError(t, err)

	sharedHeaders["x-shared"] = "mutated"

	second, err := New(Config{
		BaseURL: srv.URL,
	})
	assert.NoError(t, err)

	first.config.Headers["x-first"] = "value"

	_, presentInSecond := second.config.Headers["x-first"]
	assert.Equal(t, "value", first.config.Headers["x-shared"])
	assert.False(t, presentInSecond)

	_, presentInDefault := defaultConfig.Headers["x-first"]
	assert.False(t, presentInDefault)
}

func TestRequestTransformPrecedence(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	configTransform := func(req *Request) ([]byte, error) {
		return []byte("config"), nil
	}

	requestTransform := func(req *Request) ([]byte, error) {
		return []byte("request"), nil
	}

	vecto, err := New(Config{
		BaseURL:          srv.URL,
		RequestTransform: configTransform,
	})
	assert.NoError(t, err)

	req, err := vecto.newRequest("/test/methods", http.MethodPost, &RequestOptions{})
	assert.NoError(t, err)

	httpReq, err := req.toHTTPRequest(context.Background())
	assert.NoError(t, err)
	defer httpReq.Body.Close()

	body, err := io.ReadAll(httpReq.Body)
	assert.NoError(t, err)
	assert.Equal(t, "config", string(body))

	reqWithOverride, err := vecto.newRequest("/test/methods", http.MethodPost, &RequestOptions{
		RequestTransform: requestTransform,
	})
	assert.NoError(t, err)

	httpReqOverride, err := reqWithOverride.toHTTPRequest(context.Background())
	assert.NoError(t, err)
	defer httpReqOverride.Body.Close()

	overrideBody, err := io.ReadAll(httpReqOverride.Body)
	assert.NoError(t, err)
	assert.Equal(t, "request", string(overrideBody))
}

func TestNilContextHandling(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{
		BaseURL: srv.URL,
	})
	assert.NoError(t, err)

	res, err := vecto.Get(context.TODO(), "/test/pets/1", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestResponseBodySizeLimit(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{
		BaseURL: srv.URL,
	})
	assert.NoError(t, err)

	_, err = vecto.Get(context.Background(), "/test/large-response", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded maximum size")
}

func TestConfigValidation(t *testing.T) {
	t.Run("negative timeout", func(t *testing.T) {
		_, err := New(Config{
			Timeout: -1 * time.Second,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout cannot be negative")
	})

	t.Run("negative max response body size", func(t *testing.T) {
		_, err := New(Config{
			MaxResponseBodySize: -1,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max response body size cannot be negative")
	})

	t.Run("negative max concurrent callbacks", func(t *testing.T) {
		_, err := New(Config{
			MaxConcurrentCallbacks: -1,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max concurrent callbacks cannot be negative")
	})

	t.Run("negative callback timeout", func(t *testing.T) {
		_, err := New(Config{
			CallbackTimeout: -1 * time.Second,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "callback timeout cannot be negative")
	})

	t.Run("invalid base URL", func(t *testing.T) {
		_, err := New(Config{
			BaseURL: "://invalid-url",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid base URL")
	})

	t.Run("valid config", func(t *testing.T) {
		srv := newHTTPTestServer()
		defer srv.Close()

		_, err := New(Config{
			BaseURL:                srv.URL,
			Timeout:                10 * time.Second,
			MaxResponseBodySize:    50 * 1024 * 1024,
			MaxConcurrentCallbacks: 50,
			CallbackTimeout:        15 * time.Second,
		})
		assert.NoError(t, err)
	})
}

func TestConfigurableLimits(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	customMaxSize := int64(50 * 1024 * 1024)
	customMaxCallbacks := 50
	customCallbackTimeout := 15 * time.Second

	vecto, err := New(Config{
		BaseURL:                srv.URL,
		MaxResponseBodySize:    customMaxSize,
		MaxConcurrentCallbacks: customMaxCallbacks,
		CallbackTimeout:        customCallbackTimeout,
	})
	assert.NoError(t, err)
	assert.Equal(t, customMaxSize, vecto.config.MaxResponseBodySize)
	assert.Equal(t, customMaxCallbacks, vecto.config.MaxConcurrentCallbacks)
	assert.Equal(t, customCallbackTimeout, vecto.config.CallbackTimeout)
}

func TestResponseErrorUnwrap(t *testing.T) {
	t.Run("without underlying error", func(t *testing.T) {
		err := &ResponseError{
			Response: &Response{
				StatusCode: 404,
				Data:       []byte("Not Found"),
			},
			Err: nil,
		}

		assert.Nil(t, err.Unwrap())
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("with underlying error", func(t *testing.T) {
		underlyingErr := fmt.Errorf("network error")
		err := &ResponseError{
			Response: &Response{
				StatusCode: 500,
				Data:       []byte("Internal Server Error"),
			},
			Err: underlyingErr,
		}

		assert.Equal(t, underlyingErr, err.Unwrap())
		assert.Contains(t, err.Error(), "500")
		assert.Contains(t, err.Error(), "network error")
	})
}

func TestHeadMethod(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{BaseURL: srv.URL})
	assert.Nil(t, err)

	t.Run("successful HEAD request", func(t *testing.T) {
		res, err := vecto.Head(context.Background(), "/test/methods", nil)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("HEAD with headers", func(t *testing.T) {
		res, err := vecto.Head(context.Background(), "/test/methods", &RequestOptions{
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}

func TestShouldUseRetry(t *testing.T) {
	t.Run("should use retry when circuit breaker is nil", func(t *testing.T) {
		result := shouldUseRetry(nil)
		assert.True(t, result)
	})

	t.Run("should use retry when circuit breaker is closed", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		cb := NewCircuitBreaker("test", config)

		result := shouldUseRetry(cb)
		assert.True(t, result)
	})

	t.Run("should not use retry when circuit breaker is open", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		config.FailureThreshold = 1
		cb := NewCircuitBreaker("test", config)

		res := &Response{StatusCode: 500}
		cb.RecordResult(res, nil)

		result := shouldUseRetry(cb)
		assert.False(t, result)
	})

	t.Run("should use retry when circuit breaker is half-open", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		config.FailureThreshold = 1
		config.Timeout = 10 * time.Millisecond
		cb := NewCircuitBreaker("test", config)

		res := &Response{StatusCode: 500}
		_, _ = cb.Execute(context.Background(), func() (*Response, error) {
			return res, nil
		})
		cb.RecordResult(res, nil)

		time.Sleep(20 * time.Millisecond)

		state := cb.GetState()
		if state == StateHalfOpen {
			result := shouldUseRetry(cb)
			assert.True(t, result)
		}
	})
}

func TestWriteDebugOutput(t *testing.T) {
	t.Run("writes debug output with logger", func(t *testing.T) {
		logger := &mockLogger{}
		vecto, err := New(Config{
			BaseURL: "https://api.example.com",
			Logger:  logger,
		})
		assert.Nil(t, err)

		req, _ := vecto.newRequest("/users/123", "GET", &RequestOptions{
			Headers: map[string]string{"X-Request-Id": "test"},
		})

		res := &Response{
			StatusCode: 200,
			Data:       []byte(`{"id": 123}`),
			success:    true,
		}

		vecto.writeDebugOutput(req, res)

		assert.True(t, len(logger.debugCalls) > 0)
		if len(logger.debugCalls) > 0 {
			assert.Contains(t, logger.debugCalls[0].msg, "DEBUG INFO")
			assert.Contains(t, logger.debugCalls[0].msg, "GET")
		}
	})

	t.Run("writes debug output with trace info", func(t *testing.T) {
		logger := &mockLogger{}
		vecto, err := New(Config{
			BaseURL:     "https://api.example.com",
			Logger:      logger,
			EnableTrace: true,
		})
		assert.Nil(t, err)

		req, _ := vecto.newRequest("/users/123", "GET", nil)

		trace := &TraceInfo{
			DNSLookup:     10 * time.Millisecond,
			TCPConnection: 20 * time.Millisecond,
			TLSHandshake:  30 * time.Millisecond,
		}

		res := &Response{
			StatusCode: 200,
			Data:       []byte(`{"id": 123}`),
			success:    true,
			TraceInfo:  trace,
		}

		vecto.writeDebugOutput(req, res)

		assert.True(t, len(logger.debugCalls) > 0)
	})

	t.Run("does not panic with nil request", func(t *testing.T) {
		logger := &mockLogger{}
		vecto, _ := New(Config{
			BaseURL: "https://api.example.com",
			Logger:  logger,
		})

		res := &Response{StatusCode: 200}
		vecto.writeDebugOutput(nil, res)

		assert.Equal(t, 0, len(logger.debugCalls))
	})

	t.Run("does not panic with nil response", func(t *testing.T) {
		logger := &mockLogger{}
		vecto, _ := New(Config{
			BaseURL: "https://api.example.com",
			Logger:  logger,
		})

		req, _ := vecto.newRequest("/test", "GET", nil)
		vecto.writeDebugOutput(req, nil)

		assert.Equal(t, 0, len(logger.debugCalls))
	})

	t.Run("does not write when logger is noop", func(t *testing.T) {
		vecto, _ := New(Config{BaseURL: "https://api.example.com"})

		req, _ := vecto.newRequest("/test", "GET", nil)
		res := &Response{StatusCode: 200}

		vecto.writeDebugOutput(req, res)
	})
}
