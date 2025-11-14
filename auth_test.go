package vecto

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAuthType_String(t *testing.T) {
	tests := []struct {
		name     string
		authType AuthType
		expected string
	}{
		{
			name:     "basic auth",
			authType: AuthTypeBasic,
			expected: "Basic",
		},
		{
			name:     "bearer auth",
			authType: AuthTypeBearer,
			expected: "Bearer",
		},
		{
			name:     "digest auth",
			authType: AuthTypeDigest,
			expected: "Digest",
		},
		{
			name:     "unknown auth",
			authType: AuthType(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.authType.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEncodeBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		expected string
	}{
		{
			name:     "standard credentials",
			username: "user",
			password: "pass",
			expected: "Basic dXNlcjpwYXNz",
		},
		{
			name:     "empty password",
			username: "user",
			password: "",
			expected: "Basic dXNlcjo=",
		},
		{
			name:     "special characters",
			username: "user@example.com",
			password: "p@ss:word!",
			expected: "Basic dXNlckBleGFtcGxlLmNvbTpwQHNzOndvcmQh",
		},
		{
			name:     "unicode characters",
			username: "usuário",
			password: "señal",
			expected: "Basic dXN1w6FyaW86c2XDsWFs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeBasicAuth(tt.username, tt.password)
			if result != tt.expected {
				t.Errorf("encodeBasicAuth() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDecodeBasicAuth(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectUsername string
		expectPassword string
		expectErr      bool
	}{
		{
			name:           "standard credentials",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectUsername: "user",
			expectPassword: "pass",
			expectErr:      false,
		},
		{
			name:           "empty password",
			authHeader:     "Basic dXNlcjo=",
			expectUsername: "user",
			expectPassword: "",
			expectErr:      false,
		},
		{
			name:           "special characters",
			authHeader:     "Basic dXNlckBleGFtcGxlLmNvbTpwQHNzOndvcmQh",
			expectUsername: "user@example.com",
			expectPassword: "p@ss:word!",
			expectErr:      false,
		},
		{
			name:       "invalid header - no Basic prefix",
			authHeader: "Bearer token123",
			expectErr:  true,
		},
		{
			name:       "invalid header - empty",
			authHeader: "",
			expectErr:  true,
		},
		{
			name:       "invalid header - only prefix",
			authHeader: "Basic",
			expectErr:  true,
		},
		{
			name:       "invalid base64",
			authHeader: "Basic !!!invalid!!!",
			expectErr:  true,
		},
		{
			name:       "invalid format - no colon",
			authHeader: "Basic dXNlcnBhc3M=",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, password, err := DecodeBasicAuth(tt.authHeader)
			if (err != nil) != tt.expectErr {
				t.Errorf("DecodeBasicAuth() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr {
				if username != tt.expectUsername {
					t.Errorf("username = %v, want %v", username, tt.expectUsername)
				}
				if password != tt.expectPassword {
					t.Errorf("password = %v, want %v", password, tt.expectPassword)
				}
			}
		})
	}
}

func TestVecto_SetBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		validate func(t *testing.T, v *Vecto)
	}{
		{
			name:     "valid credentials",
			username: "user",
			password: "pass",
			validate: func(t *testing.T, v *Vecto) {
				auth := v.config.Headers["Authorization"]
				if !strings.HasPrefix(auth, "Basic ") {
					t.Errorf("Authorization header should start with 'Basic '")
				}
				username, password, err := DecodeBasicAuth(auth)
				if err != nil {
					t.Errorf("failed to decode auth: %v", err)
				}
				if username != "user" || password != "pass" {
					t.Errorf("credentials = %v:%v, want user:pass", username, password)
				}
			},
		},
		{
			name:     "empty username",
			username: "",
			password: "pass",
			validate: func(t *testing.T, v *Vecto) {
				if _, exists := v.config.Headers["Authorization"]; exists {
					t.Error("Authorization header should not be set for empty username")
				}
			},
		},
		{
			name:     "empty password",
			username: "user",
			password: "",
			validate: func(t *testing.T, v *Vecto) {
				auth := v.config.Headers["Authorization"]
				username, password, err := DecodeBasicAuth(auth)
				if err != nil {
					t.Errorf("failed to decode auth: %v", err)
				}
				if username != "user" || password != "" {
					t.Errorf("credentials = %v:%v, want user:", username, password)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := New(Config{Timeout: time.Second})
			if err != nil {
				t.Fatalf("failed to create vecto: %v", err)
			}

			v.SetBasicAuth(tt.username, tt.password)
			tt.validate(t, v)
		})
	}
}

func TestVecto_SetBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		validate func(t *testing.T, v *Vecto)
	}{
		{
			name:  "valid token",
			token: "token123",
			validate: func(t *testing.T, v *Vecto) {
				auth := v.config.Headers["Authorization"]
				expected := "Bearer token123"
				if auth != expected {
					t.Errorf("Authorization = %v, want %v", auth, expected)
				}
			},
		},
		{
			name:  "empty token",
			token: "",
			validate: func(t *testing.T, v *Vecto) {
				if _, exists := v.config.Headers["Authorization"]; exists {
					t.Error("Authorization header should not be set for empty token")
				}
			},
		},
		{
			name:  "token with special characters",
			token: "token-123_ABC.xyz",
			validate: func(t *testing.T, v *Vecto) {
				auth := v.config.Headers["Authorization"]
				expected := "Bearer token-123_ABC.xyz"
				if auth != expected {
					t.Errorf("Authorization = %v, want %v", auth, expected)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := New(Config{Timeout: time.Second})
			if err != nil {
				t.Fatalf("failed to create vecto: %v", err)
			}

			v.SetBearerToken(tt.token)
			tt.validate(t, v)
		})
	}
}

func TestVecto_SetAuthToken(t *testing.T) {
	v, err := New(Config{Timeout: time.Second})
	if err != nil {
		t.Fatalf("failed to create vecto: %v", err)
	}

	v.SetAuthToken("token123")
	auth := v.config.Headers["Authorization"]
	expected := "Bearer token123"
	if auth != expected {
		t.Errorf("Authorization = %v, want %v", auth, expected)
	}
}

func TestVecto_ClearAuth(t *testing.T) {
	v, err := New(Config{Timeout: time.Second})
	if err != nil {
		t.Fatalf("failed to create vecto: %v", err)
	}

	v.SetBearerToken("token123")
	if _, exists := v.config.Headers["Authorization"]; !exists {
		t.Error("Authorization header should be set")
	}

	v.ClearAuth()
	if _, exists := v.config.Headers["Authorization"]; exists {
		t.Error("Authorization header should be cleared")
	}
}

func TestRequest_SetBasicAuth(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		expectErr bool
		validate  func(t *testing.T, req *Request)
	}{
		{
			name:      "valid credentials",
			username:  "user",
			password:  "pass",
			expectErr: false,
			validate: func(t *testing.T, req *Request) {
				auth := req.GetAuthHeader()
				if !strings.HasPrefix(auth, "Basic ") {
					t.Errorf("Authorization header should start with 'Basic '")
				}
				username, password, err := DecodeBasicAuth(auth)
				if err != nil {
					t.Errorf("failed to decode auth: %v", err)
				}
				if username != "user" || password != "pass" {
					t.Errorf("credentials = %v:%v, want user:pass", username, password)
				}
			},
		},
		{
			name:      "empty username",
			username:  "",
			password:  "pass",
			expectErr: true,
		},
		{
			name:      "empty password",
			username:  "user",
			password:  "",
			expectErr: false,
			validate: func(t *testing.T, req *Request) {
				auth := req.GetAuthHeader()
				username, password, err := DecodeBasicAuth(auth)
				if err != nil {
					t.Errorf("failed to decode auth: %v", err)
				}
				if username != "user" || password != "" {
					t.Errorf("credentials = %v:%v, want user:", username, password)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := newRequestBuilder("https://example.com", "GET")
			req, err := builder.Build()
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			err = req.SetBasicAuth(tt.username, tt.password)
			if (err != nil) != tt.expectErr {
				t.Errorf("SetBasicAuth() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, req)
			}
		})
	}
}

func TestRequest_SetBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		expectErr bool
		validate  func(t *testing.T, req *Request)
	}{
		{
			name:      "valid token",
			token:     "token123",
			expectErr: false,
			validate: func(t *testing.T, req *Request) {
				auth := req.GetAuthHeader()
				expected := "Bearer token123"
				if auth != expected {
					t.Errorf("Authorization = %v, want %v", auth, expected)
				}
			},
		},
		{
			name:      "empty token",
			token:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := newRequestBuilder("https://example.com", "GET")
			req, err := builder.Build()
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			err = req.SetBearerToken(tt.token)
			if (err != nil) != tt.expectErr {
				t.Errorf("SetBearerToken() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, req)
			}
		})
	}
}

func TestRequest_SetAuthToken(t *testing.T) {
	builder := newRequestBuilder("https://example.com", "GET")
	req, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	err = req.SetAuthToken("token123")
	if err != nil {
		t.Errorf("SetAuthToken() error = %v", err)
	}

	auth := req.GetAuthHeader()
	expected := "Bearer token123"
	if auth != expected {
		t.Errorf("Authorization = %v, want %v", auth, expected)
	}
}

func TestRequest_ClearAuth(t *testing.T) {
	builder := newRequestBuilder("https://example.com", "GET")
	req, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	err = req.SetBearerToken("token123")
	if err != nil {
		t.Errorf("SetBearerToken() error = %v", err)
	}

	if req.GetAuthHeader() == "" {
		t.Error("Authorization header should be set")
	}

	req.ClearAuth()
	if req.GetAuthHeader() != "" {
		t.Error("Authorization header should be cleared")
	}
}

func TestRequest_GetAuthHeader(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(req *Request)
		expected string
	}{
		{
			name: "basic auth",
			setup: func(req *Request) {
				_ = req.SetBasicAuth("user", "pass")
			},
			expected: "Basic dXNlcjpwYXNz",
		},
		{
			name: "bearer token",
			setup: func(req *Request) {
				_ = req.SetBearerToken("token123")
			},
			expected: "Bearer token123",
		},
		{
			name:     "no auth",
			setup:    func(req *Request) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := newRequestBuilder("https://example.com", "GET")
			req, err := builder.Build()
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			tt.setup(req)
			result := req.GetAuthHeader()
			if result != tt.expected {
				t.Errorf("GetAuthHeader() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAuth_Integration(t *testing.T) {
	t.Run("vecto level auth", func(t *testing.T) {
		v, err := New(Config{Timeout: time.Second})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		v.SetBasicAuth("admin", "secret")

		req, err := v.newRequest("/api/users", "GET", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		headers := req.Headers()
		auth, exists := headers["Authorization"]
		if !exists {
			t.Error("Authorization header should be set")
		}

		username, password, err := DecodeBasicAuth(auth)
		if err != nil {
			t.Errorf("failed to decode auth: %v", err)
		}
		if username != "admin" || password != "secret" {
			t.Errorf("credentials = %v:%v, want admin:secret", username, password)
		}
	})

	t.Run("request level auth overrides vecto level", func(t *testing.T) {
		v, err := New(Config{Timeout: time.Second})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		v.SetBasicAuth("admin", "secret")

		req, err := v.newRequest("/api/users", "GET", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		err = req.SetBearerToken("override-token")
		if err != nil {
			t.Errorf("SetBearerToken() error = %v", err)
		}

		auth := req.GetAuthHeader()
		expected := "Bearer override-token"
		if auth != expected {
			t.Errorf("Authorization = %v, want %v", auth, expected)
		}
	})

	t.Run("clear auth", func(t *testing.T) {
		v, err := New(Config{Timeout: time.Second})
		if err != nil {
			t.Fatalf("failed to create vecto: %v", err)
		}

		v.SetBasicAuth("user", "pass")
		if _, exists := v.config.Headers["Authorization"]; !exists {
			t.Error("Authorization should be set")
		}

		v.ClearAuth()
		if _, exists := v.config.Headers["Authorization"]; exists {
			t.Error("Authorization should be cleared")
		}
	})
}

func BenchmarkEncodeBasicAuth(b *testing.B) {
	username := "user@example.com"
	password := "complex_p@ssw0rd_123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = encodeBasicAuth(username, password)
	}
}

func BenchmarkDecodeBasicAuth(b *testing.B) {
	authHeader := "Basic dXNlckBleGFtcGxlLmNvbTpjb21wbGV4X3BAc3N3MHJkXzEyMyE="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeBasicAuth(authHeader)
	}
}

func BenchmarkRequest_SetBasicAuth(b *testing.B) {
	builder := newRequestBuilder("https://example.com", "GET")
	req, _ := builder.Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.SetBasicAuth("user", "pass")
	}
}

func BenchmarkRequest_SetBearerToken(b *testing.B) {
	builder := newRequestBuilder("https://example.com", "GET")
	req, _ := builder.Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.SetBearerToken("token123")
	}
}

func TestAuthConfig(t *testing.T) {
	t.Run("create auth config", func(t *testing.T) {
		config := AuthConfig{
			Type:     AuthTypeBasic,
			Username: "user",
			Password: "pass",
			Token:    "",
		}

		if config.Type != AuthTypeBasic {
			t.Errorf("Type = %v, want %v", config.Type, AuthTypeBasic)
		}
		if config.Username != "user" {
			t.Errorf("Username = %v, want user", config.Username)
		}
		if config.Password != "pass" {
			t.Errorf("Password = %v, want pass", config.Password)
		}
	})

	t.Run("bearer token config", func(t *testing.T) {
		config := AuthConfig{
			Type:  AuthTypeBearer,
			Token: "token123",
		}

		if config.Type != AuthTypeBearer {
			t.Errorf("Type = %v, want %v", config.Type, AuthTypeBearer)
		}
		if config.Token != "token123" {
			t.Errorf("Token = %v, want token123", config.Token)
		}
	})
}

func ExampleVecto_SetBasicAuth() {
	v, _ := New(Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})

	v.SetBasicAuth("username", "password")

	ctx := context.Background()
	_, _ = v.Get(ctx, "/protected-endpoint", nil)
}

func ExampleVecto_SetBearerToken() {
	v, _ := New(Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})

	v.SetBearerToken("your-jwt-token-here")

	ctx := context.Background()
	_, _ = v.Get(ctx, "/protected-endpoint", nil)
}

func ExampleRequest_SetBasicAuth() {
	v, _ := New(Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})

	req, _ := v.newRequest("/users", "GET", nil)
	_ = req.SetBasicAuth("admin", "secret")
}

func ExampleRequest_SetBearerToken() {
	v, _ := New(Config{
		BaseURL: "https://api.example.com",
		Timeout: 30 * time.Second,
	})

	req, _ := v.newRequest("/users", "GET", nil)
	_ = req.SetBearerToken("jwt-token")
}

