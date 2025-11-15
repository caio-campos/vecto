package vecto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequest_OnCompleted_SingleChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	eventCh := make(chan RequestCompletedEvent, 1)

	ctx := context.Background()
	res, err := client.Request(ctx, "/test", http.MethodGet, &RequestOptions{
		Data: nil,
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	res.request.OnCompleted(eventCh)

	client.channelDispatcher.dispatch(ctx, res)

	select {
	case event := <-eventCh:
		if event.Response() == nil {
			t.Error("expected response in event, got nil")
		}
		if event.Response().StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", event.Response().StatusCode)
		}
		if string(event.Response().Data) != `{"status": "ok"}` {
			t.Errorf("unexpected response body: %s", string(event.Response().Data))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestRequest_OnCompleted_MultipleChannels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	eventCh1 := make(chan RequestCompletedEvent, 1)
	eventCh2 := make(chan RequestCompletedEvent, 1)
	eventCh3 := make(chan RequestCompletedEvent, 1)

	ctx := context.Background()
	res, err := client.Request(ctx, "/test", http.MethodGet, &RequestOptions{})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	res.request.OnCompleted(eventCh1).OnCompleted(eventCh2).OnCompleted(eventCh3)

	client.channelDispatcher.dispatch(ctx, res)

	received := 0
	timeout := time.After(2 * time.Second)

	for received < 3 {
		select {
		case event := <-eventCh1:
			received++
			if event.Response().StatusCode != http.StatusOK {
				t.Errorf("channel 1: expected status 200, got %d", event.Response().StatusCode)
			}
		case event := <-eventCh2:
			received++
			if event.Response().StatusCode != http.StatusOK {
				t.Errorf("channel 2: expected status 200, got %d", event.Response().StatusCode)
			}
		case event := <-eventCh3:
			received++
			if event.Response().StatusCode != http.StatusOK {
				t.Errorf("channel 3: expected status 200, got %d", event.Response().StatusCode)
			}
		case <-timeout:
			t.Fatalf("timeout waiting for events, received %d out of 3", received)
		}
	}
}

func TestRequest_OnCompleted_ChannelNotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	logger := &testLogger{logs: make([]logEntry, 0)}
	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	eventCh := make(chan RequestCompletedEvent)

	ctx := context.Background()
	res, err := client.Request(ctx, "/test", http.MethodGet, &RequestOptions{})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	res.request.OnCompleted(eventCh)

	client.channelDispatcher.dispatch(ctx, res)

	time.Sleep(100 * time.Millisecond)

	foundWarning := false
	for _, entry := range logger.logs {
		if entry.level == "warn" && entry.msg == "channel receiver not ready, skipping event" {
			foundWarning = true
			break
		}
	}

	if !foundWarning {
		t.Error("expected warning log when channel is not ready")
	}
}

func TestRequest_OnCompleted_WithContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	eventCh := make(chan RequestCompletedEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := client.Request(context.Background(), "/test", http.MethodGet, &RequestOptions{})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	res.request.OnCompleted(eventCh)

	client.channelDispatcher.dispatch(ctx, res)

	select {
	case <-eventCh:
		t.Error("should not receive event when context is cancelled")
	case <-time.After(100 * time.Millisecond):
	}
}


func TestRequest_OnCompleted_Chaining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	eventCh1 := make(chan RequestCompletedEvent, 1)
	eventCh2 := make(chan RequestCompletedEvent, 1)

	ctx := context.Background()
	res, err := client.Request(ctx, "/test", http.MethodGet, &RequestOptions{})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	returnedReq := res.request.OnCompleted(eventCh1).OnCompleted(eventCh2)

	if returnedReq != res.request {
		t.Error("OnCompleted should return the same request for chaining")
	}

	client.channelDispatcher.dispatch(ctx, res)

	receivedCount := 0
	timeout := time.After(2 * time.Second)

	for receivedCount < 2 {
		select {
		case <-eventCh1:
			receivedCount++
		case <-eventCh2:
			receivedCount++
		case <-timeout:
			t.Fatalf("timeout: received %d events out of 2", receivedCount)
		}
	}
}

type testLogger struct {
	logs []logEntry
}

type logEntry struct {
	level string
	msg   string
	data  map[string]interface{}
}

func (l *testLogger) Debug(ctx context.Context, msg string, data map[string]interface{}) {
	l.logs = append(l.logs, logEntry{level: "debug", msg: msg, data: data})
}

func (l *testLogger) Info(ctx context.Context, msg string, data map[string]interface{}) {
	l.logs = append(l.logs, logEntry{level: "info", msg: msg, data: data})
}

func (l *testLogger) Warn(ctx context.Context, msg string, data map[string]interface{}) {
	l.logs = append(l.logs, logEntry{level: "warn", msg: msg, data: data})
}

func (l *testLogger) Error(ctx context.Context, msg string, data map[string]interface{}) {
	l.logs = append(l.logs, logEntry{level: "error", msg: msg, data: data})
}

func (l *testLogger) IsNoop() bool {
	return false
}

