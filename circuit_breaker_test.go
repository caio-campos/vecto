package vecto

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 3
	config.SuccessThreshold = 2
	config.Timeout = 100 * time.Millisecond
	config.HalfOpenMaxRequests = 1

	cb := NewCircuitBreaker("test", config)
	assert.Equal(t, StateClosed, cb.GetState())

	res, err := cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 500}, nil
	})
	require.NoError(t, err)
	cb.RecordResult(res, nil)

	res, err = cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 500}, nil
	})
	require.NoError(t, err)
	cb.RecordResult(res, nil)

	res, err = cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 500}, nil
	})
	require.NoError(t, err)
	cb.RecordResult(res, nil)

	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreaker_BlocksRequestsWhenOpen(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.Timeout = 100 * time.Millisecond

	cb := NewCircuitBreaker("test", config)

	res, err := cb.Execute(context.Background(), func() (*Response, error) {
		return nil, errors.New("test error")
	})
	require.Error(t, err)
	cb.RecordResult(res, err)

	assert.Equal(t, StateOpen, cb.GetState())

	_, err = cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 200}, nil
	})
	assert.Error(t, err)
	assert.IsType(t, &CircuitBreakerError{}, err)
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.SuccessThreshold = 1
	config.Timeout = 50 * time.Millisecond
	config.HalfOpenMaxRequests = 1

	cb := NewCircuitBreaker("test", config)

	res, err := cb.Execute(context.Background(), func() (*Response, error) {
		return nil, errors.New("test error")
	})
	require.Error(t, err)
	cb.RecordResult(res, err)

	assert.Equal(t, StateOpen, cb.GetState())

	time.Sleep(60 * time.Millisecond)

	res, err = cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 200}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.GetState())

	cb.RecordResult(res, nil)
	assert.Equal(t, StateClosed, cb.GetState())
}


func TestCircuitBreaker_SlidingWindow(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 3
	config.WindowSize = 100 * time.Millisecond

	cb := NewCircuitBreaker("test", config)

	for i := 0; i < 3; i++ {
		res, err := cb.Execute(context.Background(), func() (*Response, error) {
			return &Response{StatusCode: 500}, nil
		})
		require.NoError(t, err)
		cb.RecordResult(res, nil)
	}

	assert.Equal(t, StateOpen, cb.GetState())

	time.Sleep(110 * time.Millisecond)

	stats := cb.GetStats()
	assert.Equal(t, 0, stats.FailureCount)
}

func TestCircuitBreaker_CustomShouldTrip(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 2
	config.ShouldTrip = func(res *Response, err error) bool {
		if err != nil {
			return true
		}
		return res != nil && res.StatusCode == 404
	}

	cb := NewCircuitBreaker("test", config)

	res, err := cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 404}, nil
	})
	require.NoError(t, err)
	cb.RecordResult(res, nil)

	res, err = cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 404}, nil
	})
	require.NoError(t, err)
	cb.RecordResult(res, nil)

	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreaker_Stats(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 2

	cb := NewCircuitBreaker("test", config)

	stats := cb.GetStats()
	assert.Equal(t, StateClosed, stats.State)
	assert.Equal(t, 0, stats.FailureCount)

	res, err := cb.Execute(context.Background(), func() (*Response, error) {
		return &Response{StatusCode: 500}, nil
	})
	require.NoError(t, err)
	cb.RecordResult(res, nil)

	stats = cb.GetStats()
	assert.Equal(t, 1, stats.FailureCount)
}

func TestCircuitBreakerManager_GetOrCreate(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	mgr := NewCircuitBreakerManager(config, nil)

	cb1 := mgr.GetOrCreate("key1", nil)
	cb2 := mgr.GetOrCreate("key1", nil)

	assert.Same(t, cb1, cb2)

	cb3 := mgr.GetOrCreate("key2", nil)
	assert.NotSame(t, cb1, cb3)
}

func TestCircuitBreakerManager_CustomConfig(t *testing.T) {
	defaultConfig := DefaultCircuitBreakerConfig()
	mgr := NewCircuitBreakerManager(defaultConfig, nil)

	customConfig := DefaultCircuitBreakerConfig()
	customConfig.FailureThreshold = 10

	cb := mgr.GetOrCreate("key1", &customConfig)
	stats := cb.GetStats()
	assert.Equal(t, StateClosed, stats.State)

	for i := 0; i < 10; i++ {
		res, err := cb.Execute(context.Background(), func() (*Response, error) {
			return &Response{StatusCode: 500}, nil
		})
		require.NoError(t, err)
		cb.RecordResult(res, nil)
	}

	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreakerError(t *testing.T) {
	err := &CircuitBreakerError{
		State: StateOpen,
		Key:   "test-key",
	}

	assert.Contains(t, err.Error(), "open")
	assert.Contains(t, err.Error(), "test-key")
}

func TestDefaultShouldTrip(t *testing.T) {
	tests := []struct {
		name     string
		res      *Response
		err      error
		expected bool
	}{
		{
			name:     "nil response",
			res:      nil,
			err:      nil,
			expected: true,
		},
		{
			name:     "error present",
			res:      nil,
			err:      errors.New("test error"),
			expected: true,
		},
		{
			name:     "500 status code",
			res:      &Response{StatusCode: 500},
			err:      nil,
			expected: true,
		},
		{
			name:     "200 status code",
			res:      &Response{StatusCode: 200},
			err:      nil,
			expected: false,
		},
		{
			name:     "299 status code",
			res:      &Response{StatusCode: 299},
			err:      nil,
			expected: false,
		},
		{
			name:     "400 status code",
			res:      &Response{StatusCode: 400},
			err:      nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultShouldTrip(tt.res, tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 100
	cb := NewCircuitBreaker("test", config)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				res, err := cb.Execute(context.Background(), func() (*Response, error) {
					return &Response{StatusCode: 200}, nil
				})
				if err == nil {
					cb.RecordResult(res, nil)
				}
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := cb.GetStats()
	assert.Equal(t, StateClosed, stats.State)
}

