package vecto

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
)

// String returns the response body as a string.
func (r *Response) String() string {
	if r == nil || r.Data == nil {
		return ""
	}
	return string(r.Data)
}

// JSON unmarshals the response body as JSON into the provided interface.
// Returns an error if the response is nil, empty, or if JSON unmarshaling fails.
func (r *Response) JSON(v interface{}) error {
	if r == nil {
		return fmt.Errorf("response is nil")
	}

	if len(r.Data) == 0 {
		return fmt.Errorf("response body is empty")
	}

	if v == nil {
		return fmt.Errorf("destination variable is nil")
	}

	if err := json.Unmarshal(r.Data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// XML unmarshals the response body as XML into the provided interface.
// Returns an error if the response is nil, empty, or if XML unmarshaling fails.
func (r *Response) XML(v interface{}) error {
	if r == nil {
		return fmt.Errorf("response is nil")
	}

	if len(r.Data) == 0 {
		return fmt.Errorf("response body is empty")
	}

	if v == nil {
		return fmt.Errorf("destination variable is nil")
	}

	if err := xml.Unmarshal(r.Data, v); err != nil {
		return fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	return nil
}

// IsSuccess returns true if the HTTP status code is in the 2xx range.
func (r *Response) IsSuccess() bool {
	if r == nil {
		return false
	}
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsError returns true if the HTTP status code is in the 4xx or 5xx range.
func (r *Response) IsError() bool {
	if r == nil {
		return false
	}
	return r.StatusCode >= 400 && r.StatusCode < 600
}

// IsClientError returns true if the HTTP status code is in the 4xx range.
func (r *Response) IsClientError() bool {
	if r == nil {
		return false
	}
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the HTTP status code is in the 5xx range.
func (r *Response) IsServerError() bool {
	if r == nil {
		return false
	}
	return r.StatusCode >= 500 && r.StatusCode < 600
}

// Header returns the value of a specific header from the response.
// Returns an empty string if the header is not found or if the response is nil.
func (r *Response) Header(key string) string {
	if r == nil || r.RawResponse == nil || r.RawResponse.Header == nil {
		return ""
	}
	return r.RawResponse.Header.Get(key)
}

// Headers returns all headers from the response as a map.
// Returns nil if the response or headers are nil.
func (r *Response) Headers() map[string][]string {
	if r == nil || r.RawResponse == nil || r.RawResponse.Header == nil {
		return nil
	}

	headers := make(map[string][]string, len(r.RawResponse.Header))
	for key, values := range r.RawResponse.Header {
		headerValues := make([]string, len(values))
		copy(headerValues, values)
		headers[key] = headerValues
	}

	return headers
}

// Cookies returns all cookies from the response.
// Returns nil if the response is nil.
func (r *Response) Cookies() []*http.Cookie {
	if r == nil || r.RawResponse == nil {
		return nil
	}
	return r.RawResponse.Cookies()
}

// Cookie returns a specific cookie by name from the response.
// Returns nil if the cookie is not found or if the response is nil.
func (r *Response) Cookie(name string) *http.Cookie {
	if r == nil || r.RawResponse == nil {
		return nil
	}

	cookies := r.RawResponse.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

// Result unmarshals the response body into the provided interface if the request was successful.
// This combines success checking with unmarshaling in a single call.
// Returns an error if the response failed or if unmarshaling fails.
func (r *Response) Result(v interface{}) error {
	if r == nil {
		return fmt.Errorf("response is nil")
	}

	if !r.success {
		return &ResponseError{
			Response: r,
		}
	}

	if v == nil {
		return nil
	}

	if len(r.Data) == 0 {
		return nil
	}

	contentType := r.Header("Content-Type")
	if contentType == "" && r.RawResponse != nil {
		contentType = r.RawResponse.Header.Get("Content-Type")
	}

	if containsString(contentType, "application/json") {
		if err := json.Unmarshal(r.Data, v); err != nil {
			return fmt.Errorf("failed to unmarshal JSON result: %w", err)
		}
		return nil
	}

	if containsString(contentType, "application/xml") || containsString(contentType, "text/xml") {
		if err := xml.Unmarshal(r.Data, v); err != nil {
			return fmt.Errorf("failed to unmarshal XML result: %w", err)
		}
		return nil
	}

	if err := json.Unmarshal(r.Data, v); err != nil {
		return fmt.Errorf("failed to unmarshal result (assumed JSON): %w", err)
	}

	return nil
}

func containsString(s, substr string) bool {
	if s == "" || substr == "" {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
