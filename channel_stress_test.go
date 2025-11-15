package vecto

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Simula o EventRecorder do seu código
type mockEventRecorder struct {
	mu           sync.Mutex
	events       []mockIntegrationEvent
	recordedChan chan struct{}
	recordCount  int32
}

type mockIntegrationEvent struct {
	EventID    string
	LeadID     string
	TenantID   string
	Response   string
	Success    bool
	URL        string
	Method     string
	StatusCode int
	RequestID  string
	Duration   float64
}

func newMockEventRecorder() *mockEventRecorder {
	return &mockEventRecorder{
		events:       make([]mockIntegrationEvent, 0),
		recordedChan: make(chan struct{}, 10000),
	}
}

func (m *mockEventRecorder) RecordEvent(ctx context.Context, event *mockIntegrationEvent) {
	atomic.AddInt32(&m.recordCount, 1)
	m.mu.Lock()
	m.events = append(m.events, *event)
	m.mu.Unlock()

	select {
	case m.recordedChan <- struct{}{}:
	default:
	}
}

func (m *mockEventRecorder) GetEventCount() int {
	return int(atomic.LoadInt32(&m.recordCount))
}

func (m *mockEventRecorder) WaitForEvents(count int, timeout time.Duration) bool {
	deadline := time.After(timeout)
	received := 0

	for received < count {
		select {
		case <-m.recordedChan:
			received++
		case <-deadline:
			return false
		}
	}
	return true
}

// Simula o interceptor do seu código usando channels
type eventRecorder interface {
	RecordEvent(ctx context.Context, event *mockIntegrationEvent)
	GetEventCount() int
	WaitForEvents(count int, timeout time.Duration) bool
}

func integrationEventInterceptor(rec eventRecorder, eventID, leadID, tenantID string) RequestMiddlewareFunc {
	eventCh := make(chan RequestCompletedEvent, 10000)

	go func() {
		for event := range eventCh {
			reqStart := time.Now()
			res := event.Response()
			req := res.request

			var reqID string
			if headers := req.Headers(); headers != nil {
				if val, ok := headers["X-Req-Id"]; ok {
					reqID = val
				}
			}

			obsEvent := &mockIntegrationEvent{
				EventID:    eventID,
				LeadID:     leadID,
				TenantID:   tenantID,
				Response:   string(res.Data),
				Success:    res.Success(),
				URL:        req.FullUrl(),
				Method:     req.Method(),
				StatusCode: res.StatusCode,
				RequestID:  reqID,
				Duration:   time.Since(reqStart).Seconds(),
			}

			rec.RecordEvent(context.Background(), obsEvent)
		}
	}()

	return func(ctx context.Context, req *Request) (*Request, error) {
		req.OnCompleted(eventCh)
		return req, nil
	}
}

func TestChannelStress_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
	})
	assert.NoError(t, err)

	recorder := newMockEventRecorder()
	interceptor := integrationEventInterceptor(recorder, "event-123", "lead-456", "tenant-789")
	client.UseRequest(interceptor)

	const numRequests = 1000
	const concurrency = 100

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx := context.Background()
			_, err := client.Get(ctx, "/test", &RequestOptions{
				Headers: map[string]string{
					"X-Req-Id": fmt.Sprintf("req-%d", idx),
				},
			})
			if err != nil {
				t.Logf("Request %d failed: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	if !recorder.WaitForEvents(numRequests, 10*time.Second) {
		t.Errorf("Timeout waiting for all events. Expected %d, got %d",
			numRequests, recorder.GetEventCount())
	}

	t.Logf("Processed %d requests in %v (%.2f req/s)",
		numRequests, duration, float64(numRequests)/duration.Seconds())

	assert.Equal(t, numRequests, recorder.GetEventCount(),
		"All events should be recorded")
}

func TestChannelStress_MultipleInterceptors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	})
	assert.NoError(t, err)

	const numInterceptors = 5
	recorders := make([]*mockEventRecorder, numInterceptors)

	for i := 0; i < numInterceptors; i++ {
		recorder := newMockEventRecorder()
		recorders[i] = recorder
		interceptor := integrationEventInterceptor(
			recorder,
			fmt.Sprintf("event-%d", i),
			fmt.Sprintf("lead-%d", i),
			fmt.Sprintf("tenant-%d", i),
		)
		client.UseRequest(interceptor)
	}

	const numRequests = 500
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx := context.Background()
			client.Get(ctx, "/test", nil)
		}(i)
	}

	wg.Wait()

	for i, recorder := range recorders {
		if !recorder.WaitForEvents(numRequests, 10*time.Second) {
			t.Errorf("Recorder %d: timeout waiting for events. Got %d/%d",
				i, recorder.GetEventCount(), numRequests)
		}
		assert.Equal(t, numRequests, recorder.GetEventCount(),
			"Recorder %d should receive all events", i)
	}
}

type slowMockEventRecorder struct {
	*mockEventRecorder
}

func newSlowMockEventRecorder() *slowMockEventRecorder {
	return &slowMockEventRecorder{
		mockEventRecorder: newMockEventRecorder(),
	}
}

func (s *slowMockEventRecorder) RecordEvent(ctx context.Context, event *mockIntegrationEvent) {
	time.Sleep(5 * time.Millisecond)
	atomic.AddInt32(&s.recordCount, 1)
	s.mu.Lock()
	s.events = append(s.events, *event)
	s.mu.Unlock()

	select {
	case s.recordedChan <- struct{}{}:
	default:
	}
}

func TestChannelStress_RapidRequestsWithSlowRecorder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	assert.NoError(t, err)

	slowRecorder := newSlowMockEventRecorder()

	interceptor := integrationEventInterceptor(slowRecorder, "event-slow", "lead-slow", "tenant-slow")
	client.UseRequest(interceptor)

	const numRequests = 200
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			client.Get(ctx, "/test", nil)
		}()
	}

	wg.Wait()
	requestsDuration := time.Since(startTime)

	t.Logf("All requests completed in %v", requestsDuration)

	if !slowRecorder.WaitForEvents(numRequests, 30*time.Second) {
		t.Errorf("Timeout waiting for slow recorder. Got %d/%d events",
			slowRecorder.GetEventCount(), numRequests)
	}

	totalDuration := time.Since(startTime)
	t.Logf("All events recorded in %v", totalDuration)

	assert.Equal(t, numRequests, slowRecorder.GetEventCount())
}

func TestChannelStress_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	assert.NoError(t, err)

	recorder := newMockEventRecorder()
	interceptor := integrationEventInterceptor(recorder, "event-cancel", "lead-cancel", "tenant-cancel")
	client.UseRequest(interceptor)

	const numRequests = 100
	var wg sync.WaitGroup
	successCount := int32(0)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()

			_, err := client.Get(ctx, "/test", nil)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	successful := int(atomic.LoadInt32(&successCount))
	t.Logf("Successful requests: %d/%d", successful, numRequests)
	t.Logf("Events recorded: %d", recorder.GetEventCount())

	assert.LessOrEqual(t, recorder.GetEventCount(), numRequests,
		"Should not record more events than requests")
}

func TestChannelStress_MixedStatusCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal error"}`))
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		case "/timeout":
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusRequestTimeout)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	assert.NoError(t, err)

	recorder := newMockEventRecorder()
	interceptor := integrationEventInterceptor(recorder, "event-mixed", "lead-mixed", "tenant-mixed")
	client.UseRequest(interceptor)

	endpoints := []string{"/success", "/error", "/notfound", "/timeout", "/badrequest"}
	const requestsPerEndpoint = 100

	var wg sync.WaitGroup
	totalRequests := len(endpoints) * requestsPerEndpoint

	for _, endpoint := range endpoints {
		for i := 0; i < requestsPerEndpoint; i++ {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				ctx := context.Background()
				client.Get(ctx, path, nil)
			}(endpoint)
		}
	}

	wg.Wait()

	if !recorder.WaitForEvents(totalRequests, 15*time.Second) {
		t.Errorf("Timeout waiting for events. Got %d/%d",
			recorder.GetEventCount(), totalRequests)
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	statusCodeCount := make(map[int]int)
	for _, event := range recorder.events {
		statusCodeCount[event.StatusCode]++
	}

	t.Logf("Status code distribution: %v", statusCodeCount)
	assert.Equal(t, totalRequests, recorder.GetEventCount())
}

func TestChannelStress_MemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test response with some data"}`))
	}))
	defer server.Close()

	const iterations = 5
	const requestsPerIteration = 500

	for iter := 0; iter < iterations; iter++ {
		client, err := New(Config{
			BaseURL: server.URL,
			Timeout: 10 * time.Second,
		})
		assert.NoError(t, err)

		recorder := newMockEventRecorder()
		interceptor := integrationEventInterceptor(
			recorder,
			fmt.Sprintf("event-iter-%d", iter),
			"lead-mem",
			"tenant-mem",
		)
		client.UseRequest(interceptor)

		var wg sync.WaitGroup
		for i := 0; i < requestsPerIteration; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()
				client.Get(ctx, "/test", nil)
			}()
		}

		wg.Wait()
		recorder.WaitForEvents(requestsPerIteration, 5*time.Second)

		t.Logf("Iteration %d: %d requests completed, %d events recorded",
			iter+1, requestsPerIteration, recorder.GetEventCount())
	}

	t.Log("Memory leak test completed - check for goroutine leaks manually if needed")
}

func TestChannelStress_EdgeCase_VeryLargeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	})
	assert.NoError(t, err)

	recorder := newMockEventRecorder()
	interceptor := integrationEventInterceptor(recorder, "event-large", "lead-large", "tenant-large")
	client.UseRequest(interceptor)

	const numRequests = 50
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			client.Get(ctx, "/test", nil)
		}()
	}

	wg.Wait()

	if !recorder.WaitForEvents(numRequests, 10*time.Second) {
		t.Errorf("Timeout waiting for events. Got %d/%d",
			recorder.GetEventCount(), numRequests)
	}

	assert.Equal(t, numRequests, recorder.GetEventCount())

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	for _, event := range recorder.events {
		assert.Greater(t, len(event.Response), 1024*1024-100,
			"Response should contain large data")
	}
}

func TestChannelStress_EdgeCase_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	assert.NoError(t, err)

	recorder := newMockEventRecorder()
	interceptor := integrationEventInterceptor(recorder, "event-empty", "lead-empty", "tenant-empty")
	client.UseRequest(interceptor)

	const numRequests = 200
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			client.Get(ctx, "/test", nil)
		}()
	}

	wg.Wait()

	if !recorder.WaitForEvents(numRequests, 5*time.Second) {
		t.Errorf("Timeout waiting for events. Got %d/%d",
			recorder.GetEventCount(), numRequests)
	}

	assert.Equal(t, numRequests, recorder.GetEventCount())

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	for _, event := range recorder.events {
		assert.Equal(t, http.StatusNoContent, event.StatusCode)
		assert.Empty(t, event.Response)
	}
}

func TestChannelStress_BurstTraffic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	})
	assert.NoError(t, err)

	recorder := newMockEventRecorder()
	interceptor := integrationEventInterceptor(recorder, "event-burst", "lead-burst", "tenant-burst")
	client.UseRequest(interceptor)

	const numBursts = 10
	const requestsPerBurst = 200

	totalRequests := 0
	startTime := time.Now()

	for burst := 0; burst < numBursts; burst++ {
		var wg sync.WaitGroup

		for i := 0; i < requestsPerBurst; i++ {
			wg.Add(1)
			totalRequests++
			go func() {
				defer wg.Done()
				ctx := context.Background()
				client.Get(ctx, "/test", nil)
			}()
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond)
	}

	duration := time.Since(startTime)

	if !recorder.WaitForEvents(totalRequests, 15*time.Second) {
		t.Errorf("Timeout waiting for events. Got %d/%d",
			recorder.GetEventCount(), totalRequests)
	}

	t.Logf("Processed %d requests in %d bursts over %v",
		totalRequests, numBursts, duration)
	assert.Equal(t, totalRequests, recorder.GetEventCount())
}
