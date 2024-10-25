package vecto

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type PetMockData struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Owner MockDataOwner `json:"owner"`
}

type MockDataOwner struct {
	Login string `json:"login"`
}

var supportedMethods = map[string]bool{
	http.MethodPost:    true,
	http.MethodGet:     true,
	http.MethodPatch:   true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

type handleReqMethod func(ctx context.Context, url string, options *RequestOptions) (res *Response, err error)

func TestHeaderConfiguration(t *testing.T) {
	customHeaderExpectedValue := "custom"
	anotherHeaderOriginalValue := "another"
	reqCustomHeaderExpectedValue := "req-custom"
	anotherHeaderExpectedValue := "another-replaced"

	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
		Headers: map[string]string{
			"x-custom":  customHeaderExpectedValue,
			"x-another": anotherHeaderOriginalValue,
		},
	})

	res, _ := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{
		Headers: map[string]string{
			"x-req-custom": reqCustomHeaderExpectedValue,
			"x-another":    anotherHeaderExpectedValue,
		},
	})

	xCustomHeaderValue, xCustomHeaderOk := res.request.headers["x-custom"]
	xAnotherHeaderValue, xAnotherHeaderOk := res.request.headers["x-another"]
	xReqCustomHeaderValue, xReqCustomHeaderOk := res.request.headers["x-req-custom"]

	assert.True(t, xCustomHeaderOk)
	assert.True(t, xAnotherHeaderOk)
	assert.True(t, xReqCustomHeaderOk)

	assert.Equal(t, customHeaderExpectedValue, xCustomHeaderValue)
	assert.Equal(t, anotherHeaderExpectedValue, xAnotherHeaderValue)
	assert.Equal(t, reqCustomHeaderExpectedValue, xReqCustomHeaderValue)

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestRequestMethods(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
	})

	methodsMap := map[string]handleReqMethod{
		http.MethodPost:    vecto.Post,
		http.MethodGet:     vecto.Get,
		http.MethodPatch:   vecto.Patch,
		http.MethodPut:     vecto.Put,
		http.MethodDelete:  vecto.Delete,
		http.MethodOptions: vecto.Options,
	}

	for method, funct := range methodsMap {
		res, err := funct(context.Background(), "/test/methods", nil)

		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode, method)
	}

	res, err := vecto.Request(context.Background(), "/test/methods", http.MethodTrace, nil)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode, nil)
}

func TestRequestTimeoutConfiguration(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
		Timeout: 3000,
	})

	res, err := vecto.Post(context.Background(), "/test/slow", nil)
	assert.NotNil(t, err)
	assert.Nil(t, res)
}
