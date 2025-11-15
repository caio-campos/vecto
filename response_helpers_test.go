package vecto

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestResponse_String(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		expected string
	}{
		{
			name: "valid response",
			response: &Response{
				Data: []byte("hello world"),
			},
			expected: "hello world",
		},
		{
			name: "empty response",
			response: &Response{
				Data: []byte{},
			},
			expected: "",
		},
		{
			name:     "nil response",
			response: nil,
			expected: "",
		},
		{
			name: "nil data",
			response: &Response{
				Data: nil,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResponse_JSON(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name      string
		response  *Response
		dest      interface{}
		expectErr bool
		validate  func(t *testing.T, dest interface{})
	}{
		{
			name: "valid json",
			response: &Response{
				Data: []byte(`{"name":"John","age":30}`),
			},
			dest:      &Person{},
			expectErr: false,
			validate: func(t *testing.T, dest interface{}) {
				p := dest.(*Person)
				if p.Name != "John" || p.Age != 30 {
					t.Errorf("JSON() parsed incorrectly: got %+v", p)
				}
			},
		},
		{
			name: "invalid json",
			response: &Response{
				Data: []byte(`{"name":"John",`),
			},
			dest:      &Person{},
			expectErr: true,
		},
		{
			name:      "nil response",
			response:  nil,
			dest:      &Person{},
			expectErr: true,
		},
		{
			name: "empty data",
			response: &Response{
				Data: []byte{},
			},
			dest:      &Person{},
			expectErr: true,
		},
		{
			name: "nil destination",
			response: &Response{
				Data: []byte(`{"name":"John","age":30}`),
			},
			dest:      nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.JSON(tt.dest)
			if (err != nil) != tt.expectErr {
				t.Errorf("JSON() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, tt.dest)
			}
		})
	}
}

func TestResponse_XML(t *testing.T) {
	type Person struct {
		Name string `xml:"name"`
		Age  int    `xml:"age"`
	}

	tests := []struct {
		name      string
		response  *Response
		dest      interface{}
		expectErr bool
		validate  func(t *testing.T, dest interface{})
	}{
		{
			name: "valid xml",
			response: &Response{
				Data: []byte(`<Person><name>John</name><age>30</age></Person>`),
			},
			dest:      &Person{},
			expectErr: false,
			validate: func(t *testing.T, dest interface{}) {
				p := dest.(*Person)
				if p.Name != "John" || p.Age != 30 {
					t.Errorf("XML() parsed incorrectly: got %+v", p)
				}
			},
		},
		{
			name: "invalid xml",
			response: &Response{
				Data: []byte(`<Person><name>John`),
			},
			dest:      &Person{},
			expectErr: true,
		},
		{
			name:      "nil response",
			response:  nil,
			dest:      &Person{},
			expectErr: true,
		},
		{
			name: "empty data",
			response: &Response{
				Data: []byte{},
			},
			dest:      &Person{},
			expectErr: true,
		},
		{
			name: "nil destination",
			response: &Response{
				Data: []byte(`<Person><name>John</name><age>30</age></Person>`),
			},
			dest:      nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.XML(tt.dest)
			if (err != nil) != tt.expectErr {
				t.Errorf("XML() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, tt.dest)
			}
		})
	}
}

func TestResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		wantResult bool
	}{
		{
			name:       "200 OK",
			response:   &Response{StatusCode: 200},
			wantResult: true,
		},
		{
			name:       "201 Created",
			response:   &Response{StatusCode: 201},
			wantResult: true,
		},
		{
			name:       "204 No Content",
			response:   &Response{StatusCode: 204},
			wantResult: true,
		},
		{
			name:       "299 edge case",
			response:   &Response{StatusCode: 299},
			wantResult: true,
		},
		{
			name:       "199 edge case",
			response:   &Response{StatusCode: 199},
			wantResult: false,
		},
		{
			name:       "300 Redirect",
			response:   &Response{StatusCode: 300},
			wantResult: false,
		},
		{
			name:       "400 Bad Request",
			response:   &Response{StatusCode: 400},
			wantResult: false,
		},
		{
			name:       "404 Not Found",
			response:   &Response{StatusCode: 404},
			wantResult: false,
		},
		{
			name:       "500 Internal Server Error",
			response:   &Response{StatusCode: 500},
			wantResult: false,
		},
		{
			name:       "nil response",
			response:   nil,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.IsSuccess()
			if result != tt.wantResult {
				t.Errorf("IsSuccess() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestResponse_IsError(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		wantResult bool
	}{
		{
			name:       "400 Bad Request",
			response:   &Response{StatusCode: 400},
			wantResult: true,
		},
		{
			name:       "404 Not Found",
			response:   &Response{StatusCode: 404},
			wantResult: true,
		},
		{
			name:       "499 edge case",
			response:   &Response{StatusCode: 499},
			wantResult: true,
		},
		{
			name:       "500 Internal Server Error",
			response:   &Response{StatusCode: 500},
			wantResult: true,
		},
		{
			name:       "503 Service Unavailable",
			response:   &Response{StatusCode: 503},
			wantResult: true,
		},
		{
			name:       "599 edge case",
			response:   &Response{StatusCode: 599},
			wantResult: true,
		},
		{
			name:       "200 OK",
			response:   &Response{StatusCode: 200},
			wantResult: false,
		},
		{
			name:       "300 Redirect",
			response:   &Response{StatusCode: 300},
			wantResult: false,
		},
		{
			name:       "399 edge case",
			response:   &Response{StatusCode: 399},
			wantResult: false,
		},
		{
			name:       "600 edge case",
			response:   &Response{StatusCode: 600},
			wantResult: false,
		},
		{
			name:       "nil response",
			response:   nil,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.IsError()
			if result != tt.wantResult {
				t.Errorf("IsError() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestResponse_IsClientError(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		wantResult bool
	}{
		{
			name:       "400 Bad Request",
			response:   &Response{StatusCode: 400},
			wantResult: true,
		},
		{
			name:       "404 Not Found",
			response:   &Response{StatusCode: 404},
			wantResult: true,
		},
		{
			name:       "499 edge case",
			response:   &Response{StatusCode: 499},
			wantResult: true,
		},
		{
			name:       "500 Internal Server Error",
			response:   &Response{StatusCode: 500},
			wantResult: false,
		},
		{
			name:       "200 OK",
			response:   &Response{StatusCode: 200},
			wantResult: false,
		},
		{
			name:       "nil response",
			response:   nil,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.IsClientError()
			if result != tt.wantResult {
				t.Errorf("IsClientError() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestResponse_IsServerError(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		wantResult bool
	}{
		{
			name:       "500 Internal Server Error",
			response:   &Response{StatusCode: 500},
			wantResult: true,
		},
		{
			name:       "503 Service Unavailable",
			response:   &Response{StatusCode: 503},
			wantResult: true,
		},
		{
			name:       "599 edge case",
			response:   &Response{StatusCode: 599},
			wantResult: true,
		},
		{
			name:       "400 Bad Request",
			response:   &Response{StatusCode: 400},
			wantResult: false,
		},
		{
			name:       "200 OK",
			response:   &Response{StatusCode: 200},
			wantResult: false,
		},
		{
			name:       "600 edge case",
			response:   &Response{StatusCode: 600},
			wantResult: false,
		},
		{
			name:       "nil response",
			response:   nil,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.IsServerError()
			if result != tt.wantResult {
				t.Errorf("IsServerError() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestResponse_Header(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		headerKey  string
		wantResult string
	}{
		{
			name: "existing header",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				},
			},
			headerKey:  "Content-Type",
			wantResult: "application/json",
		},
		{
			name: "non-existing header",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				},
			},
			headerKey:  "Authorization",
			wantResult: "",
		},
		{
			name: "case insensitive header",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				},
			},
			headerKey:  "content-type",
			wantResult: "application/json",
		},
		{
			name:       "nil response",
			response:   nil,
			headerKey:  "Content-Type",
			wantResult: "",
		},
		{
			name: "nil raw response",
			response: &Response{
				RawResponse: nil,
			},
			headerKey:  "Content-Type",
			wantResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.Header(tt.headerKey)
			if result != tt.wantResult {
				t.Errorf("Header() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestResponse_Headers(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		wantNil    bool
		wantLength int
		validate   func(t *testing.T, headers map[string][]string)
	}{
		{
			name: "valid headers",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type":   []string{"application/json"},
						"Content-Length": []string{"123"},
						"X-Custom":       []string{"value1", "value2"},
					},
				},
			},
			wantNil:    false,
			wantLength: 3,
			validate: func(t *testing.T, headers map[string][]string) {
				if len(headers["Content-Type"]) != 1 || headers["Content-Type"][0] != "application/json" {
					t.Errorf("Content-Type header incorrect")
				}
				if len(headers["X-Custom"]) != 2 {
					t.Errorf("X-Custom header should have 2 values")
				}
			},
		},
		{
			name:       "nil response",
			response:   nil,
			wantNil:    true,
			wantLength: 0,
		},
		{
			name: "nil raw response",
			response: &Response{
				RawResponse: nil,
			},
			wantNil:    true,
			wantLength: 0,
		},
		{
			name: "empty headers",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{},
				},
			},
			wantNil:    false,
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.Headers()
			if tt.wantNil {
				if result != nil {
					t.Errorf("Headers() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("Headers() = nil, want non-nil")
				}
				if len(result) != tt.wantLength {
					t.Errorf("Headers() length = %d, want %d", len(result), tt.wantLength)
				}
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestResponse_Cookies(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		wantNil    bool
		wantLength int
	}{
		{
			name: "valid cookies",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Set-Cookie": []string{
							"session=abc123; Path=/",
							"user=john; Path=/; HttpOnly",
						},
					},
				},
			},
			wantNil:    false,
			wantLength: 2,
		},
		{
			name: "no cookies",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{},
				},
			},
			wantNil:    false,
			wantLength: 0,
		},
		{
			name:       "nil response",
			response:   nil,
			wantNil:    true,
			wantLength: 0,
		},
		{
			name: "nil raw response",
			response: &Response{
				RawResponse: nil,
			},
			wantNil:    true,
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.Cookies()
			if tt.wantNil {
				if result != nil {
					t.Errorf("Cookies() = %v, want nil", result)
				}
			} else {
				if len(result) != tt.wantLength {
					t.Errorf("Cookies() length = %d, want %d", len(result), tt.wantLength)
				}
			}
		})
	}
}

func TestResponse_Cookie(t *testing.T) {
	tests := []struct {
		name       string
		response   *Response
		cookieName string
		wantNil    bool
		wantValue  string
	}{
		{
			name: "existing cookie",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Set-Cookie": []string{
							"session=abc123; Path=/",
							"user=john; Path=/; HttpOnly",
						},
					},
				},
			},
			cookieName: "session",
			wantNil:    false,
			wantValue:  "abc123",
		},
		{
			name: "non-existing cookie",
			response: &Response{
				RawResponse: &http.Response{
					Header: http.Header{
						"Set-Cookie": []string{
							"session=abc123; Path=/",
						},
					},
				},
			},
			cookieName: "nonexistent",
			wantNil:    true,
		},
		{
			name:       "nil response",
			response:   nil,
			cookieName: "session",
			wantNil:    true,
		},
		{
			name: "nil raw response",
			response: &Response{
				RawResponse: nil,
			},
			cookieName: "session",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.Cookie(tt.cookieName)
			if tt.wantNil {
				if result != nil {
					t.Errorf("Cookie() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("Cookie() = nil, want non-nil")
					return
				}
				if result.Value != tt.wantValue {
					t.Errorf("Cookie().Value = %v, want %v", result.Value, tt.wantValue)
				}
			}
		})
	}
}

func TestResponse_Result(t *testing.T) {
	type Person struct {
		Name string `json:"name" xml:"name"`
		Age  int    `json:"age" xml:"age"`
	}

	tests := []struct {
		name      string
		response  *Response
		dest      interface{}
		expectErr bool
		validate  func(t *testing.T, dest interface{})
	}{
		{
			name: "successful json response",
			response: &Response{
				Data:       []byte(`{"name":"John","age":30}`),
				StatusCode: 200,
				success:    true,
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				},
			},
			dest:      &Person{},
			expectErr: false,
			validate: func(t *testing.T, dest interface{}) {
				p := dest.(*Person)
				if p.Name != "John" || p.Age != 30 {
					t.Errorf("Result() parsed incorrectly: got %+v", p)
				}
			},
		},
		{
			name: "successful xml response",
			response: &Response{
				Data:       []byte(`<Person><name>Jane</name><age>25</age></Person>`),
				StatusCode: 200,
				success:    true,
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/xml"},
					},
				},
			},
			dest:      &Person{},
			expectErr: false,
			validate: func(t *testing.T, dest interface{}) {
				p := dest.(*Person)
				if p.Name != "Jane" || p.Age != 25 {
					t.Errorf("Result() parsed incorrectly: got %+v", p)
				}
			},
		},
		{
			name: "failed response",
			response: &Response{
				Data:       []byte(`{"error":"not found"}`),
				StatusCode: 404,
				success:    false,
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				},
			},
			dest:      &Person{},
			expectErr: true,
		},
		{
			name:      "nil response",
			response:  nil,
			dest:      &Person{},
			expectErr: true,
		},
		{
			name: "empty data with success",
			response: &Response{
				Data:       []byte{},
				StatusCode: 204,
				success:    true,
				RawResponse: &http.Response{
					Header: http.Header{},
				},
			},
			dest:      &Person{},
			expectErr: false,
		},
		{
			name: "nil destination with success",
			response: &Response{
				Data:       []byte(`{"name":"John","age":30}`),
				StatusCode: 200,
				success:    true,
				RawResponse: &http.Response{
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
				},
			},
			dest:      nil,
			expectErr: false,
		},
		{
			name: "json without content-type",
			response: &Response{
				Data:       []byte(`{"name":"John","age":30}`),
				StatusCode: 200,
				success:    true,
				RawResponse: &http.Response{
					Header: http.Header{},
				},
			},
			dest:      &Person{},
			expectErr: false,
			validate: func(t *testing.T, dest interface{}) {
				p := dest.(*Person)
				if p.Name != "John" || p.Age != 30 {
					t.Errorf("Result() parsed incorrectly: got %+v", p)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.Result(tt.dest)
			if (err != nil) != tt.expectErr {
				t.Errorf("Result() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, tt.dest)
			}
		})
	}
}

func TestResponse_ContainsString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "substring exists",
			s:        "application/json; charset=utf-8",
			substr:   "application/json",
			expected: true,
		},
		{
			name:     "substring not exists",
			s:        "application/json",
			substr:   "xml",
			expected: false,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "test",
			substr:   "",
			expected: false,
		},
		{
			name:     "both empty",
			s:        "",
			substr:   "",
			expected: false,
		},
		{
			name:     "substring at beginning",
			s:        "application/json",
			substr:   "application",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "application/json",
			substr:   "json",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("containsString(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func BenchmarkResponse_String(b *testing.B) {
	response := &Response{
		Data: []byte("hello world from benchmark"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = response.String()
	}
}

func BenchmarkResponse_JSON(b *testing.B) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	response := &Response{
		Data: []byte(`{"name":"John","age":30}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var p Person
		_ = response.JSON(&p)
	}
}

func BenchmarkResponse_IsSuccess(b *testing.B) {
	response := &Response{
		StatusCode: 200,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = response.IsSuccess()
	}
}

func BenchmarkResponse_Headers(b *testing.B) {
	response := &Response{
		RawResponse: &http.Response{
			Header: http.Header{
				"Content-Type":   []string{"application/json"},
				"Content-Length": []string{"123"},
				"X-Custom":       []string{"value1", "value2"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = response.Headers()
	}
}

func createTestResponse(statusCode int, body string, contentType string) *Response {
	return &Response{
		Data:       []byte(body),
		StatusCode: statusCode,
		success:    statusCode >= 200 && statusCode < 300,
		RawResponse: &http.Response{
			StatusCode: statusCode,
			Header: http.Header{
				"Content-Type": []string{contentType},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(body))),
		},
	}
}

func TestResponse_Integration(t *testing.T) {
	t.Run("complete json workflow", func(t *testing.T) {
		type User struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		response := createTestResponse(200, `{"id":1,"name":"John","email":"john@example.com"}`, "application/json")

		if !response.IsSuccess() {
			t.Error("expected success response")
		}

		if response.IsError() {
			t.Error("expected non-error response")
		}

		var user User
		if err := response.Result(&user); err != nil {
			t.Errorf("Result() error = %v", err)
		}

		if user.ID != 1 || user.Name != "John" || user.Email != "john@example.com" {
			t.Errorf("user data incorrect: %+v", user)
		}

		contentType := response.Header("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", contentType)
		}
	})

	t.Run("complete error workflow", func(t *testing.T) {
		response := createTestResponse(404, `{"error":"not found"}`, "application/json")

		if response.IsSuccess() {
			t.Error("expected non-success response")
		}

		if !response.IsError() {
			t.Error("expected error response")
		}

		if !response.IsClientError() {
			t.Error("expected client error")
		}

		if response.IsServerError() {
			t.Error("expected non-server error")
		}

		var data map[string]string
		err := response.Result(&data)
		if err == nil {
			t.Error("expected error from Result()")
		}
	})
}
