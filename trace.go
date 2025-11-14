package vecto

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http/httptrace"
	"strings"
	"time"
)

// TraceInfo contains detailed timing information about an HTTP request.
type TraceInfo struct {
	// DNSLookup is the time spent performing DNS lookup.
	DNSLookup time.Duration

	// TCPConnection is the time spent establishing TCP connection.
	TCPConnection time.Duration

	// TLSHandshake is the time spent performing TLS handshake.
	TLSHandshake time.Duration

	// ServerProcessing is the time spent waiting for server to process request.
	ServerProcessing time.Duration

	// ContentTransfer is the time spent transferring response content.
	ContentTransfer time.Duration

	// Total is the total request time.
	Total time.Duration

	// ConnReused indicates if the connection was reused.
	ConnReused bool

	// ConnWasIdle indicates if the connection was previously idle.
	ConnWasIdle bool

	// ConnIdleTime is how long the connection was idle before this request.
	ConnIdleTime time.Duration
}

// String returns a formatted string representation of the trace info.
func (t *TraceInfo) String() string {
	if t == nil {
		return "TraceInfo: <nil>"
	}

	var b strings.Builder
	b.WriteString("Request Trace:\n")
	b.WriteString(fmt.Sprintf("  DNS Lookup:        %v\n", t.DNSLookup))
	b.WriteString(fmt.Sprintf("  TCP Connection:    %v\n", t.TCPConnection))
	b.WriteString(fmt.Sprintf("  TLS Handshake:     %v\n", t.TLSHandshake))
	b.WriteString(fmt.Sprintf("  Server Processing: %v\n", t.ServerProcessing))
	b.WriteString(fmt.Sprintf("  Content Transfer:  %v\n", t.ContentTransfer))
	b.WriteString(fmt.Sprintf("  Total Time:        %v\n", t.Total))
	b.WriteString(fmt.Sprintf("  Conn Reused:       %v\n", t.ConnReused))
	if t.ConnReused {
		b.WriteString(fmt.Sprintf("  Conn Was Idle:     %v\n", t.ConnWasIdle))
		b.WriteString(fmt.Sprintf("  Conn Idle Time:    %v\n", t.ConnIdleTime))
	}

	return b.String()
}

// ToCurl generates a cURL command equivalent to the request.
func (r *Request) ToCurl() string {
	if r == nil {
		return ""
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var b strings.Builder
	
	b.WriteString("curl -X ")
	b.WriteString(r.method)
	b.WriteString(" '")
	b.WriteString(r.url)
	b.WriteString("'")

	for key, value := range r.headers {
		b.WriteString(" \\\n  -H '")
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(value)
		b.WriteString("'")
	}

	if r.data != nil {
		dataStr := formatDataForCurl(r.data)
		if dataStr != "" {
			b.WriteString(" \\\n  -d '")
			b.WriteString(dataStr)
			b.WriteString("'")
		}
	}

	return b.String()
}

// formatDataForCurl formats request data for cURL output.
func formatDataForCurl(data interface{}) string {
	if data == nil {
		return ""
	}

	switch v := data.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", data)
	}
}

// traceContext holds timing information during request execution.
type traceContext struct {
	dnsStart         time.Time
	dnsDone          time.Time
	connectStart     time.Time
	connectDone      time.Time
	tlsStart         time.Time
	tlsDone          time.Time
	gotFirstResponse time.Time
	requestStart     time.Time
	requestEnd       time.Time
	connReused       bool
	connWasIdle      bool
	connIdleTime     time.Duration
}

// createClientTrace creates an httptrace.ClientTrace for collecting timing information.
func createClientTrace(tc *traceContext) *httptrace.ClientTrace {
	if tc == nil {
		return nil
	}

	return &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			tc.dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			tc.dnsDone = time.Now()
		},
		ConnectStart: func(network, addr string) {
			tc.connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			tc.connectDone = time.Now()
		},
		TLSHandshakeStart: func() {
			tc.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tc.tlsDone = time.Now()
		},
		GotFirstResponseByte: func() {
			tc.gotFirstResponse = time.Now()
		},
		GetConn: func(hostPort string) {
		},
		GotConn: func(info httptrace.GotConnInfo) {
			tc.connReused = info.Reused
			tc.connWasIdle = info.WasIdle
			tc.connIdleTime = info.IdleTime
		},
	}
}

// computeTraceInfo computes TraceInfo from traceContext.
func computeTraceInfo(tc *traceContext) *TraceInfo {
	if tc == nil {
		return nil
	}

	info := &TraceInfo{
		ConnReused:   tc.connReused,
		ConnWasIdle:  tc.connWasIdle,
		ConnIdleTime: tc.connIdleTime,
	}

	if !tc.dnsStart.IsZero() && !tc.dnsDone.IsZero() {
		info.DNSLookup = tc.dnsDone.Sub(tc.dnsStart)
	}

	if !tc.connectStart.IsZero() && !tc.connectDone.IsZero() {
		info.TCPConnection = tc.connectDone.Sub(tc.connectStart)
	}

	if !tc.tlsStart.IsZero() && !tc.tlsDone.IsZero() {
		info.TLSHandshake = tc.tlsDone.Sub(tc.tlsStart)
	}

	if !tc.requestStart.IsZero() && !tc.gotFirstResponse.IsZero() {
		info.ServerProcessing = tc.gotFirstResponse.Sub(tc.requestStart)
	}

	if !tc.gotFirstResponse.IsZero() && !tc.requestEnd.IsZero() {
		info.ContentTransfer = tc.requestEnd.Sub(tc.gotFirstResponse)
	}

	if !tc.requestStart.IsZero() && !tc.requestEnd.IsZero() {
		info.Total = tc.requestEnd.Sub(tc.requestStart)
	}

	return info
}

// DebugWriter writes debug information to the configured writer.
func writeDebugInfo(writer io.Writer, req *Request, res *Response, trace *TraceInfo) {
	if writer == nil {
		return
	}

	fmt.Fprintf(writer, "\n=== DEBUG INFO ===\n")
	fmt.Fprintf(writer, "Request: %s %s\n", req.Method(), req.FullUrl())
	
	if headers := req.Headers(); len(headers) > 0 {
		fmt.Fprintf(writer, "\nRequest Headers:\n")
		for key, value := range headers {
			fmt.Fprintf(writer, "  %s: %s\n", key, value)
		}
	}

	if req.Data() != nil {
		fmt.Fprintf(writer, "\nRequest Body:\n  %v\n", req.Data())
	}

	if res != nil {
		fmt.Fprintf(writer, "\nResponse Status: %d\n", res.StatusCode)
		
		if headers := res.Headers(); len(headers) > 0 {
			fmt.Fprintf(writer, "\nResponse Headers:\n")
			for key, values := range headers {
				for _, value := range values {
					fmt.Fprintf(writer, "  %s: %s\n", key, value)
				}
			}
		}

		if len(res.Data) > 0 {
			bodyPreview := string(res.Data)
			if len(bodyPreview) > 500 {
				bodyPreview = bodyPreview[:500] + "... (truncated)"
			}
			fmt.Fprintf(writer, "\nResponse Body:\n%s\n", bodyPreview)
		}
	}

	if trace != nil {
		fmt.Fprintf(writer, "\n%s", trace.String())
	}

	fmt.Fprintf(writer, "\nCurl Equivalent:\n%s\n", req.ToCurl())
	fmt.Fprintf(writer, "==================\n\n")
}

