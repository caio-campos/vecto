package vecto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http URL", "http://example.com", false},
		{"valid https URL", "https://example.com/path", false},
		{"valid URL with query", "https://example.com?foo=bar", false},
		{"empty URL", "", true},
		{"invalid scheme", "ftp://example.com", true},
		{"invalid format", "://invalid", true},
		{"too long URL", string(make([]byte, maxURLLength+1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHeaderName(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{"valid header", "Content-Type", false},
		{"valid header with dash", "X-Custom-Header", false},
		{"valid header with underscore", "X_Custom_Header", false},
		{"empty header", "", true},
		{"invalid character", "Header Name!", true},
		{"too long", string(make([]byte, maxHeaderNameLen+1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaderName(tt.header)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHeaderValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid value", "application/json", false},
		{"empty value", "", false},
		{"value with CR", "value\rwith", true},
		{"value with LF", "value\nwith", true},
		{"too long", string(make([]byte, maxHeaderValueLen+1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaderValue(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		wantErr bool
	}{
		{"valid headers", map[string]string{"Content-Type": "application/json"}, false},
		{"nil headers", nil, false},
		{"empty headers", map[string]string{}, false},
		{"invalid header name", map[string]string{"Invalid Name!": "value"}, true},
		{"invalid header value", map[string]string{"X-Test": "value\nwith"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaders(tt.headers)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"trim spaces", "  http://example.com  ", "http://example.com"},
		{"no change", "http://example.com", "http://example.com"},
		{"leading spaces", "  http://example.com", "http://example.com"},
		{"trailing spaces", "http://example.com  ", "http://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

