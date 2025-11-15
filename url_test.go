package vecto

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUrlInstance(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, *url.URL)
	}{
		{
			name:    "valid URL without params",
			baseURL: "http://example.com",
			params:  nil,
			wantErr: false,
			check: func(t *testing.T, u *url.URL) {
				assert.Equal(t, "http", u.Scheme)
				assert.Equal(t, "example.com", u.Host)
			},
		},
		{
			name:    "valid URL with params",
			baseURL: "http://example.com",
			params:  map[string]any{"foo": "bar", "baz": 123},
			wantErr: false,
			check: func(t *testing.T, u *url.URL) {
				assert.Contains(t, u.RawQuery, "foo=bar")
				assert.Contains(t, u.RawQuery, "baz=123")
			},
		},
		{
			name:    "invalid URL",
			baseURL: "://invalid",
			params:  nil,
			wantErr: true,
		},
		{
			name:    "URL with existing query params",
			baseURL: "http://example.com?existing=value",
			params:  map[string]any{"new": "param"},
			wantErr: false,
			check: func(t *testing.T, u *url.URL) {
				assert.Contains(t, u.RawQuery, "existing=value")
				assert.Contains(t, u.RawQuery, "new=param")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getUrlInstance(tt.baseURL, tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.check != nil {
					tt.check(t, result)
				}
			}
		})
	}
}

