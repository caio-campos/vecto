package vecto

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestSetParamError(t *testing.T) {
	req := &Request{
		baseURL: "://bad url",
		method:  http.MethodGet,
	}

	err := req.SetParam("foo", "bar")
	assert.Error(t, err)
}

func TestRequestSetParamEmptyKey(t *testing.T) {
	req := &Request{
		baseURL: "http://example.com",
		method:  http.MethodGet,
	}

	err := req.SetParam("", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "param key cannot be empty")
}

func TestRequestSetHeaderEmptyKey(t *testing.T) {
	req := &Request{
		baseURL: "http://example.com",
		method:  http.MethodGet,
	}

	initialHeaders := len(req.Headers())
	req.SetHeader("", "value")
	
	assert.Equal(t, initialHeaders, len(req.Headers()))
}

