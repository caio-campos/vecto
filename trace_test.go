package vecto

import (
	"testing"
	"time"
)

func TestTraceInfo_String(t *testing.T) {
	tests := []struct {
		name  string
		trace *TraceInfo
	}{
		{
			name: "complete trace info",
			trace: &TraceInfo{
				DNSLookup:        10 * time.Millisecond,
				TCPConnection:    20 * time.Millisecond,
				TLSHandshake:     30 * time.Millisecond,
				ServerProcessing: 100 * time.Millisecond,
				ContentTransfer:  50 * time.Millisecond,
				Total:            210 * time.Millisecond,
				ConnReused:       false,
				ConnWasIdle:      false,
			},
		},
		{
			name: "with connection reuse",
			trace: &TraceInfo{
				DNSLookup:        0,
				TCPConnection:    0,
				TLSHandshake:     0,
				ServerProcessing: 50 * time.Millisecond,
				ContentTransfer:  25 * time.Millisecond,
				Total:            75 * time.Millisecond,
				ConnReused:       true,
				ConnWasIdle:      true,
				ConnIdleTime:     1 * time.Second,
			},
		},
		{
			name:  "nil trace",
			trace: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.trace.String()
			if tt.trace == nil {
				if result != "TraceInfo: <nil>" {
					t.Errorf("String() for nil should return 'TraceInfo: <nil>', got %v", result)
				}
			} else {
				if result == "" {
					t.Error("String() should not return empty string")
				}
				t.Logf("TraceInfo String output:\n%s", result)
			}
		})
	}
}

func TestRequest_ToCurl(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Request
		contains []string
	}{
		{
			name: "simple GET request",
			setup: func() *Request {
				builder := newRequestBuilder("https://api.example.com/users", "GET")
				req, _ := builder.Build()
				return req
			},
			contains: []string{
				"curl -X GET",
				"https://api.example.com/users",
			},
		},
		{
			name: "POST with headers",
			setup: func() *Request {
				builder := newRequestBuilder("https://api.example.com/users", "POST").
					SetHeaders(map[string]string{
						"Content-Type":  "application/json",
						"Authorization": "Bearer token123",
					})
				req, _ := builder.Build()
				return req
			},
			contains: []string{
				"curl -X POST",
				"Content-Type: application/json",
				"Authorization: Bearer token123",
			},
		},
		{
			name: "request with data",
			setup: func() *Request {
				builder := newRequestBuilder("https://api.example.com/users", "POST").
					SetData(map[string]string{"name": "John"})
				req, _ := builder.Build()
				return req
			},
			contains: []string{
				"curl -X POST",
				"-d",
			},
		},
		{
			name: "nil request",
			setup: func() *Request {
				return nil
			},
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setup()
			result := req.ToCurl()

			if req == nil {
				if result != "" {
					t.Errorf("ToCurl() for nil should return empty string, got %v", result)
				}
				return
			}

			for _, expected := range tt.contains {
				if !containsString(result, expected) {
					t.Errorf("ToCurl() should contain %q, got:\n%s", expected, result)
				}
			}

			t.Logf("cURL output:\n%s", result)
		})
	}
}

func TestFormatDataForCurl(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{
			name:     "string data",
			data:     "hello world",
			expected: "hello world",
		},
		{
			name:     "byte slice",
			data:     []byte("hello"),
			expected: "hello",
		},
		{
			name:     "nil data",
			data:     nil,
			expected: "",
		},
		{
			name:     "map data",
			data:     map[string]string{"key": "value"},
			expected: "map[key:value]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDataForCurl(tt.data)
			if result != tt.expected {
				t.Errorf("formatDataForCurl() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCreateClientTrace(t *testing.T) {
	tc := &traceContext{}
	trace := createClientTrace(tc)

	if trace == nil {
		t.Fatal("createClientTrace() should not return nil")
	}

	if trace.DNSStart == nil {
		t.Error("DNSStart callback should be set")
	}
	if trace.DNSDone == nil {
		t.Error("DNSDone callback should be set")
	}
	if trace.ConnectStart == nil {
		t.Error("ConnectStart callback should be set")
	}
	if trace.ConnectDone == nil {
		t.Error("ConnectDone callback should be set")
	}
	if trace.TLSHandshakeStart == nil {
		t.Error("TLSHandshakeStart callback should be set")
	}
	if trace.TLSHandshakeDone == nil {
		t.Error("TLSHandshakeDone callback should be set")
	}
	if trace.GotFirstResponseByte == nil {
		t.Error("GotFirstResponseByte callback should be set")
	}

	t.Run("nil trace context", func(t *testing.T) {
		trace := createClientTrace(nil)
		if trace != nil {
			t.Error("createClientTrace(nil) should return nil")
		}
	})
}

func TestComputeTraceInfo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		tc       *traceContext
		validate func(t *testing.T, info *TraceInfo)
	}{
		{
			name: "complete trace",
			tc: &traceContext{
				dnsStart:         now,
				dnsDone:          now.Add(10 * time.Millisecond),
				connectStart:     now.Add(10 * time.Millisecond),
				connectDone:      now.Add(30 * time.Millisecond),
				tlsStart:         now.Add(30 * time.Millisecond),
				tlsDone:          now.Add(60 * time.Millisecond),
				requestStart:     now,
				gotFirstResponse: now.Add(100 * time.Millisecond),
				requestEnd:       now.Add(150 * time.Millisecond),
				connReused:       false,
			},
			validate: func(t *testing.T, info *TraceInfo) {
				if info == nil {
					t.Fatal("TraceInfo should not be nil")
				}
				if info.DNSLookup != 10*time.Millisecond {
					t.Errorf("DNSLookup = %v, want 10ms", info.DNSLookup)
				}
				if info.TCPConnection != 20*time.Millisecond {
					t.Errorf("TCPConnection = %v, want 20ms", info.TCPConnection)
				}
				if info.TLSHandshake != 30*time.Millisecond {
					t.Errorf("TLSHandshake = %v, want 30ms", info.TLSHandshake)
				}
			},
		},
		{
			name: "connection reused",
			tc: &traceContext{
				requestStart:     now,
				gotFirstResponse: now.Add(50 * time.Millisecond),
				requestEnd:       now.Add(75 * time.Millisecond),
				connReused:       true,
				connWasIdle:      true,
				connIdleTime:     time.Second,
			},
			validate: func(t *testing.T, info *TraceInfo) {
				if info == nil {
					t.Fatal("TraceInfo should not be nil")
				}
				if !info.ConnReused {
					t.Error("ConnReused should be true")
				}
				if !info.ConnWasIdle {
					t.Error("ConnWasIdle should be true")
				}
				if info.ConnIdleTime != time.Second {
					t.Errorf("ConnIdleTime = %v, want 1s", info.ConnIdleTime)
				}
			},
		},
		{
			name: "nil context",
			tc:   nil,
			validate: func(t *testing.T, info *TraceInfo) {
				if info != nil {
					t.Error("computeTraceInfo(nil) should return nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := computeTraceInfo(tt.tc)
			if tt.validate != nil {
				tt.validate(t, info)
			}
		})
	}
}

func TestFormatDebugInfo(t *testing.T) {
	builder := newRequestBuilder("https://api.example.com/users", "GET").
		SetHeaders(map[string]string{
			"Authorization": "Bearer token",
		})
	req, _ := builder.Build()

	res := &Response{
		StatusCode: 200,
		Data:       []byte(`{"id": 1, "name": "John"}`),
	}

	trace := &TraceInfo{
		Total:      100 * time.Millisecond,
		DNSLookup:  10 * time.Millisecond,
		ConnReused: false,
	}

	result := formatDebugInfo(req, res, trace)

	expectedStrings := []string{
		"DEBUG INFO",
		"GET",
		"https://api.example.com/users",
		"Authorization: Bearer token",
		"Response Status: 200",
		"Curl Equivalent",
	}

	for _, expected := range expectedStrings {
		if !containsString(result, expected) {
			t.Errorf("formatDebugInfo() should contain %q", expected)
		}
	}

	t.Logf("Debug info:\n%s", result)
}

func BenchmarkTraceInfo_String(b *testing.B) {
	trace := &TraceInfo{
		DNSLookup:        10 * time.Millisecond,
		TCPConnection:    20 * time.Millisecond,
		TLSHandshake:     30 * time.Millisecond,
		ServerProcessing: 100 * time.Millisecond,
		ContentTransfer:  50 * time.Millisecond,
		Total:            210 * time.Millisecond,
		ConnReused:       false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trace.String()
	}
}

func BenchmarkRequest_ToCurl(b *testing.B) {
	builder := newRequestBuilder("https://api.example.com/users", "POST").
		SetHeaders(map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token123",
		}).
		SetData(map[string]string{"name": "John"})
	req, _ := builder.Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.ToCurl()
	}
}

func ExampleTraceInfo_String() {
	trace := &TraceInfo{
		DNSLookup:        10 * time.Millisecond,
		TCPConnection:    20 * time.Millisecond,
		TLSHandshake:     30 * time.Millisecond,
		ServerProcessing: 100 * time.Millisecond,
		ContentTransfer:  50 * time.Millisecond,
		Total:            210 * time.Millisecond,
		ConnReused:       false,
	}

	_ = trace.String()
}

func ExampleRequest_ToCurl() {
	builder := newRequestBuilder("https://api.example.com/users", "POST").
		SetHeaders(map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token123",
		})
	req, _ := builder.Build()

	curlCmd := req.ToCurl()
	_ = curlCmd
}

