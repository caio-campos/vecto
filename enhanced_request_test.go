package vecto

import (
	"testing"
)

func TestReplacePathParams(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		params   map[string]string
		expected string
	}{
		{
			name:     "single param with braces",
			url:      "https://api.example.com/users/{id}",
			params:   map[string]string{"id": "123"},
			expected: "https://api.example.com/users/123",
		},
		{
			name:     "single param with colon",
			url:      "https://api.example.com/users/:id",
			params:   map[string]string{"id": "456"},
			expected: "https://api.example.com/users/456",
		},
		{
			name: "multiple params",
			url:  "https://api.example.com/users/{userId}/posts/{postId}",
			params: map[string]string{
				"userId": "123",
				"postId": "456",
			},
			expected: "https://api.example.com/users/123/posts/456",
		},
		{
			name:     "special characters",
			url:      "https://api.example.com/search/{query}",
			params:   map[string]string{"query": "hello world"},
			expected: "https://api.example.com/search/hello%20world",
		},
		{
			name:     "no params",
			url:      "https://api.example.com/users",
			params:   map[string]string{},
			expected: "https://api.example.com/users",
		},
		{
			name:     "nil params",
			url:      "https://api.example.com/users",
			params:   nil,
			expected: "https://api.example.com/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replacePathParams(tt.url, tt.params)
			if result != tt.expected {
				t.Errorf("replacePathParams() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEncodeFormData(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		validate func(t *testing.T, result string)
	}{
		{
			name: "simple form data",
			data: map[string]string{
				"username": "john",
				"password": "secret",
			},
			validate: func(t *testing.T, result string) {
				if !containsString(result, "username=john") {
					t.Errorf("result should contain username=john")
				}
				if !containsString(result, "password=secret") {
					t.Errorf("result should contain password=secret")
				}
			},
		},
		{
			name: "form data with special characters",
			data: map[string]string{
				"email": "user@example.com",
				"name":  "John Doe",
			},
			validate: func(t *testing.T, result string) {
				if !containsString(result, "email=user%40example.com") {
					t.Errorf("result should encode @ as %%40")
				}
				if !containsString(result, "name=John+Doe") || !containsString(result, "name=John%20Doe") {
					t.Log("result:", result)
				}
			},
		},
		{
			name: "empty form data",
			data: map[string]string{},
			validate: func(t *testing.T, result string) {
				if result != "" {
					t.Errorf("result should be empty, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeFormData(tt.data)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestStructToQueryParams(t *testing.T) {
	type TestQuery struct {
		Name   string `query:"name"`
		Age    int    `query:"age"`
		Active bool   `query:"active"`
	}

	type TestQueryOmitEmpty struct {
		Name   string   `query:"name,omitempty"`
		Age    int      `query:"age,omitempty"`
		Tags   []string `query:"tags,omitempty"`
		Ignore string   `query:"-"`
	}

	tests := []struct {
		name      string
		input     interface{}
		expectErr bool
		validate  func(t *testing.T, params map[string]any)
	}{
		{
			name: "simple struct",
			input: TestQuery{
				Name:   "John",
				Age:    30,
				Active: true,
			},
			expectErr: false,
			validate: func(t *testing.T, params map[string]any) {
				if params["name"] != "John" {
					t.Errorf("name = %v, want John", params["name"])
				}
				if params["age"] != "30" {
					t.Errorf("age = %v, want 30", params["age"])
				}
				if params["active"] != "true" {
					t.Errorf("active = %v, want true", params["active"])
				}
			},
		},
		{
			name: "struct with omitempty",
			input: TestQueryOmitEmpty{
				Name: "John",
				Age:  0,
				Tags: []string{},
			},
			expectErr: false,
			validate: func(t *testing.T, params map[string]any) {
				if params["name"] != "John" {
					t.Errorf("name = %v, want John", params["name"])
				}
				if _, exists := params["age"]; exists {
					t.Error("age should be omitted (zero value)")
				}
				if _, exists := params["tags"]; exists {
					t.Error("tags should be omitted (empty slice)")
				}
				if _, exists := params["Ignore"]; exists {
					t.Error("Ignore field should be ignored")
				}
			},
		},
		{
			name: "struct with slice",
			input: struct {
				IDs []int `query:"ids"`
			}{
				IDs: []int{1, 2, 3},
			},
			expectErr: false,
			validate: func(t *testing.T, params map[string]any) {
				ids, ok := params["ids"].([]string)
				if !ok {
					t.Fatalf("ids should be []string")
				}
				if len(ids) != 3 {
					t.Errorf("len(ids) = %d, want 3", len(ids))
				}
			},
		},
		{
			name:      "nil input",
			input:     nil,
			expectErr: false,
			validate: func(t *testing.T, params map[string]any) {
				if params != nil {
					t.Error("params should be nil")
				}
			},
		},
		{
			name:      "non-struct input",
			input:     "not a struct",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := structToQueryParams(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("structToQueryParams() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, params)
			}
		})
	}
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name         string
		tag          string
		expectedName string
		expectedOpts []string
	}{
		{
			name:         "name only",
			tag:          "field_name",
			expectedName: "field_name",
			expectedOpts: nil,
		},
		{
			name:         "name with omitempty",
			tag:          "field_name,omitempty",
			expectedName: "field_name",
			expectedOpts: []string{"omitempty"},
		},
		{
			name:         "name with multiple options",
			tag:          "field_name,omitempty,required",
			expectedName: "field_name",
			expectedOpts: []string{"omitempty", "required"},
		},
		{
			name:         "empty tag",
			tag:          "",
			expectedName: "",
			expectedOpts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, opts := parseTag(tt.tag)
			if name != tt.expectedName {
				t.Errorf("name = %v, want %v", name, tt.expectedName)
			}
			if len(opts) != len(tt.expectedOpts) {
				t.Errorf("len(opts) = %d, want %d", len(opts), len(tt.expectedOpts))
			}
		})
	}
}

func TestRequest_SetPathParam(t *testing.T) {
	builder := newRequestBuilder("https://api.example.com/users/{id}", "GET")
	req, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.SetPathParam("id", "123")

	params := req.Params()
	pathParams, ok := params["__path_params__"].(map[string]string)
	if !ok {
		t.Fatal("__path_params__ should exist and be map[string]string")
	}

	if pathParams["id"] != "123" {
		t.Errorf("pathParams[id] = %v, want 123", pathParams["id"])
	}
}

func TestRequest_SetPathParams(t *testing.T) {
	builder := newRequestBuilder("https://api.example.com/users/{userId}/posts/{postId}", "GET")
	req, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.SetPathParams(map[string]string{
		"userId": "123",
		"postId": "456",
	})

	params := req.Params()
	pathParams, ok := params["__path_params__"].(map[string]string)
	if !ok {
		t.Fatal("__path_params__ should exist and be map[string]string")
	}

	if pathParams["userId"] != "123" {
		t.Errorf("pathParams[userId] = %v, want 123", pathParams["userId"])
	}
	if pathParams["postId"] != "456" {
		t.Errorf("pathParams[postId] = %v, want 456", pathParams["postId"])
	}
}

func BenchmarkReplacePathParams(b *testing.B) {
	url := "https://api.example.com/users/{userId}/posts/{postId}/comments/{commentId}"
	params := map[string]string{
		"userId":    "123",
		"postId":    "456",
		"commentId": "789",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = replacePathParams(url, params)
	}
}

func BenchmarkEncodeFormData(b *testing.B) {
	data := map[string]string{
		"username": "john_doe",
		"email":    "john@example.com",
		"name":     "John Doe",
		"age":      "30",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = encodeFormData(data)
	}
}

func BenchmarkStructToQueryParams(b *testing.B) {
	type TestQuery struct {
		Name   string   `query:"name"`
		Age    int      `query:"age"`
		Active bool     `query:"active"`
		Tags   []string `query:"tags"`
	}

	input := TestQuery{
		Name:   "John",
		Age:    30,
		Active: true,
		Tags:   []string{"go", "http", "api"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = structToQueryParams(input)
	}
}

