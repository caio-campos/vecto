package vecto

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the current state of the circuit breaker.
type CircuitBreakerState int

const (
	// StateClosed indicates the circuit breaker is closed and requests are allowed.
	StateClosed CircuitBreakerState = iota
	// StateOpen indicates the circuit breaker is open and requests are blocked.
	StateOpen
	// StateHalfOpen indicates the circuit breaker is in half-open state, allowing limited requests to test recovery.
	StateHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for a circuit breaker instance.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures required to open the circuit.
	// Default: 5
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes required in half-open state to close the circuit.
	// Default: 2
	SuccessThreshold int

	// Timeout is the duration to wait before transitioning from open to half-open state.
	// Default: 60 seconds
	Timeout time.Duration

	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state.
	// Default: 1
	HalfOpenMaxRequests int

	// WindowSize is the duration of the sliding window for counting failures.
	// Default: 60 seconds
	WindowSize time.Duration

	// ShouldTrip is a function that determines if a request should be considered a failure.
	// If nil, defaults to checking for non-nil errors and unsuccessful responses.
	ShouldTrip func(res *Response, err error) bool

	// OnStateChange is an optional callback invoked when the circuit breaker state changes.
	OnStateChange func(from, to CircuitBreakerState, key string)

	// Logger is an optional logger for circuit breaker events.
	Logger Logger
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             60 * time.Second,
		HalfOpenMaxRequests: 1,
		WindowSize:          60 * time.Second,
		ShouldTrip:          defaultShouldTrip,
	}
}

func defaultShouldTrip(res *Response, err error) bool {
	if err != nil {
		return true
	}
	if res == nil {
		return true
	}
	return res.StatusCode >= 500 || res.StatusCode < 200
}

// failureRecord represents a single failure event within the sliding window.
type failureRecord struct {
	timestamp time.Time
}

// CircuitBreaker implements a thread-safe circuit breaker pattern with sliding window.
type CircuitBreaker struct {
	mu                   sync.RWMutex
	config               CircuitBreakerConfig
	key                  string
	state                CircuitBreakerState
	failures             []failureRecord
	lastFailureTime      time.Time
	halfOpenRequests     int
	consecutiveSuccesses int
	stateChangeTime      time.Time
}

// NewCircuitBreaker creates a new circuit breaker instance with the given key and configuration.
func NewCircuitBreaker(key string, config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = 1
	}
	if config.WindowSize <= 0 {
		config.WindowSize = 60 * time.Second
	}
	if config.ShouldTrip == nil {
		config.ShouldTrip = defaultShouldTrip
	}

	cb := &CircuitBreaker{
		config:          config,
		key:             key,
		state:           StateClosed,
		failures:        make([]failureRecord, 0, 10),
		stateChangeTime: time.Now(),
	}

	return cb
}

// Execute wraps a function call with circuit breaker logic.
// Note: The result must be recorded separately using RecordResult after validation.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() (*Response, error)) (*Response, error) {
	if !cb.allowRequest() {
		return nil, &CircuitBreakerError{
			State: cb.getState(),
			Key:   cb.key,
		}
	}

	res, err := fn()

	if err != nil {
		cb.recordResult(res, err)
	}

	return res, err
}

// RecordResult records the result of a request after validation.
// This should be called after validating the response status.
func (cb *CircuitBreaker) RecordResult(res *Response, err error) {
	cb.recordResult(res, err)
}

// allowRequest checks if a request should be allowed based on the current state.
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		cb.cleanupOldFailures(now)
		return true

	case StateOpen:
		if now.Sub(cb.stateChangeTime) >= cb.config.Timeout {
			cb.transitionToHalfOpen(now)
			return true
		}
		return false

	case StateHalfOpen:
		if cb.halfOpenRequests >= cb.config.HalfOpenMaxRequests {
			return false
		}
		cb.halfOpenRequests++
		return true

	default:
		return false
	}
}

// recordResult records the result of a request and updates the circuit breaker state accordingly.
func (cb *CircuitBreaker) recordResult(res *Response, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	isFailure := cb.config.ShouldTrip(res, err)

	switch cb.state {
	case StateClosed:
		if isFailure {
			cb.recordFailure(now)
			if cb.shouldOpen() {
				cb.transitionToOpen(now)
			}
		} else {
			cb.cleanupOldFailures(now)
		}

	case StateHalfOpen:
		if isFailure {
			cb.transitionToOpen(now)
			cb.halfOpenRequests = 0
			cb.consecutiveSuccesses = 0
		} else {
			cb.consecutiveSuccesses++
			cb.halfOpenRequests--
			if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
				cb.transitionToClosed(now)
			}
		}
	}
}

// recordFailure adds a failure record to the sliding window.
func (cb *CircuitBreaker) recordFailure(now time.Time) {
	cb.failures = append(cb.failures, failureRecord{timestamp: now})
	cb.lastFailureTime = now
	cb.cleanupOldFailures(now)
}

// cleanupOldFailures removes failure records outside the sliding window.
func (cb *CircuitBreaker) cleanupOldFailures(now time.Time) {
	cutoff := now.Add(-cb.config.WindowSize)
	validFailures := cb.failures[:0]
	for _, f := range cb.failures {
		if f.timestamp.After(cutoff) {
			validFailures = append(validFailures, f)
		}
	}
	cb.failures = validFailures
}

// shouldOpen determines if the circuit breaker should transition to open state.
func (cb *CircuitBreaker) shouldOpen() bool {
	return len(cb.failures) >= cb.config.FailureThreshold
}

// transitionToOpen transitions the circuit breaker to open state.
func (cb *CircuitBreaker) transitionToOpen(now time.Time) {
	if cb.state == StateOpen {
		return
	}

	oldState := cb.state
	cb.state = StateOpen
	cb.stateChangeTime = now
	cb.halfOpenRequests = 0
	cb.consecutiveSuccesses = 0

	cb.notifyStateChange(oldState, StateOpen)
	cb.logStateChange(oldState, StateOpen)
}

// transitionToHalfOpen transitions the circuit breaker to half-open state.
func (cb *CircuitBreaker) transitionToHalfOpen(now time.Time) {
	if cb.state == StateHalfOpen {
		return
	}

	oldState := cb.state
	cb.state = StateHalfOpen
	cb.stateChangeTime = now
	cb.halfOpenRequests = 0
	cb.consecutiveSuccesses = 0

	cb.notifyStateChange(oldState, StateHalfOpen)
	cb.logStateChange(oldState, StateHalfOpen)
}

// transitionToClosed transitions the circuit breaker to closed state.
func (cb *CircuitBreaker) transitionToClosed(now time.Time) {
	if cb.state == StateClosed {
		return
	}

	oldState := cb.state
	cb.state = StateClosed
	cb.stateChangeTime = now
	cb.failures = cb.failures[:0]
	cb.halfOpenRequests = 0
	cb.consecutiveSuccesses = 0

	cb.notifyStateChange(oldState, StateClosed)
	cb.logStateChange(oldState, StateClosed)
}

// notifyStateChange invokes the OnStateChange callback if configured.
func (cb *CircuitBreaker) notifyStateChange(from, to CircuitBreakerState) {
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(from, to, cb.key)
	}
}

// logStateChange logs the state change if a logger is configured.
func (cb *CircuitBreaker) logStateChange(from, to CircuitBreakerState) {
	if cb.config.Logger == nil {
		return
	}

	ctx := context.Background()
	cb.config.Logger.Info(ctx, "circuit breaker state changed", map[string]interface{}{
		"key":      cb.key,
		"from":     from.String(),
		"to":       to.String(),
		"failures": len(cb.failures),
	})
}

// getState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) getState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetState returns the current state of the circuit breaker (thread-safe).
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.getState()
}

// GetStats returns statistics about the circuit breaker.
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	now := time.Now()
	cbCopy := &CircuitBreaker{
		config:   cb.config,
		failures: make([]failureRecord, len(cb.failures)),
	}
	copy(cbCopy.failures, cb.failures)
	cbCopy.cleanupOldFailures(now)

	return CircuitBreakerStats{
		State:                cb.state,
		FailureCount:         len(cbCopy.failures),
		LastFailureTime:      cb.lastFailureTime,
		HalfOpenRequests:     cb.halfOpenRequests,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		StateChangeTime:      cb.stateChangeTime,
	}
}

// CircuitBreakerStats contains statistics about a circuit breaker instance.
type CircuitBreakerStats struct {
	State                CircuitBreakerState
	FailureCount         int
	LastFailureTime      time.Time
	HalfOpenRequests     int
	ConsecutiveSuccesses int
	StateChangeTime      time.Time
}

// CircuitBreakerError is returned when a request is blocked by the circuit breaker.
type CircuitBreakerError struct {
	State CircuitBreakerState
	Key   string
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker is %s for key: %s", e.State.String(), e.Key)
}

// CircuitBreakerManager manages multiple circuit breaker instances, one per key.
type CircuitBreakerManager struct {
	mu            sync.RWMutex
	breakers      map[string]*CircuitBreaker
	defaultConfig CircuitBreakerConfig
	logger        Logger
}

// NewCircuitBreakerManager creates a new circuit breaker manager.
func NewCircuitBreakerManager(defaultConfig CircuitBreakerConfig, logger Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers:      make(map[string]*CircuitBreaker, 16),
		defaultConfig: defaultConfig,
		logger:        logger,
	}
}

// GetOrCreate returns an existing circuit breaker for the key or creates a new one.
func (m *CircuitBreakerManager) GetOrCreate(key string, config *CircuitBreakerConfig) *CircuitBreaker {
	m.mu.RLock()
	if breaker, exists := m.breakers[key]; exists {
		m.mu.RUnlock()
		return breaker
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if breaker, exists := m.breakers[key]; exists {
		return breaker
	}

	cbConfig := m.defaultConfig
	if config != nil {
		cbConfig = *config
		if cbConfig.Logger == nil {
			cbConfig.Logger = m.logger
		}
	} else if cbConfig.Logger == nil {
		cbConfig.Logger = m.logger
	}

	breaker := NewCircuitBreaker(key, cbConfig)
	m.breakers[key] = breaker

	return breaker
}

// Get returns the circuit breaker for the given key, or nil if it doesn't exist.
func (m *CircuitBreakerManager) Get(key string) *CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.breakers[key]
}

// Remove removes a circuit breaker from the manager.
func (m *CircuitBreakerManager) Remove(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakers, key)
}

// Clear removes all circuit breakers from the manager.
func (m *CircuitBreakerManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.breakers = make(map[string]*CircuitBreaker, 16)
}
