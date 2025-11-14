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

