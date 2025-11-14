package vecto

import (
	"context"
	"io"
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

func TestConfigHeaderIsolation(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	sharedHeaders := map[string]string{
		"x-shared": "value",
	}

	first, err := New(Config{
		BaseURL: srv.URL,
		Headers: sharedHeaders,
	})
	assert.NoError(t, err)

	sharedHeaders["x-shared"] = "mutated"

	second, err := New(Config{
		BaseURL: srv.URL,
	})
	assert.NoError(t, err)

	first.config.Headers["x-first"] = "value"

	_, presentInSecond := second.config.Headers["x-first"]
	assert.Equal(t, "value", first.config.Headers["x-shared"])
	assert.False(t, presentInSecond)

	_, presentInDefault := defaultConfig.Headers["x-first"]
	assert.False(t, presentInDefault)
}

func TestRequestTransformPrecedence(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	configTransform := func(req *Request) ([]byte, error) {
		return []byte("config"), nil
	}

	requestTransform := func(req *Request) ([]byte, error) {
		return []byte("request"), nil
	}

	vecto, err := New(Config{
		BaseURL:          srv.URL,
		RequestTransform: configTransform,
	})
	assert.NoError(t, err)

	req, err := vecto.newRequest("/test/methods", http.MethodPost, &RequestOptions{})
	assert.NoError(t, err)

	httpReq, err := req.toHTTPRequest(context.Background())
	assert.NoError(t, err)
	defer httpReq.Body.Close()

	body, err := io.ReadAll(httpReq.Body)
	assert.NoError(t, err)
	assert.Equal(t, "config", string(body))

	reqWithOverride, err := vecto.newRequest("/test/methods", http.MethodPost, &RequestOptions{
		RequestTransform: requestTransform,
	})
	assert.NoError(t, err)

	httpReqOverride, err := reqWithOverride.toHTTPRequest(context.Background())
	assert.NoError(t, err)
	defer httpReqOverride.Body.Close()

	overrideBody, err := io.ReadAll(httpReqOverride.Body)
	assert.NoError(t, err)
	assert.Equal(t, "request", string(overrideBody))
}

func TestNilContextHandling(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{
		BaseURL: srv.URL,
	})
	assert.NoError(t, err)

	res, err := vecto.Get(nil, "/test/pets/1", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestResponseBodySizeLimit(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, err := New(Config{
		BaseURL: srv.URL,
	})
	assert.NoError(t, err)

	_, err = vecto.Get(context.Background(), "/test/large-response", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded maximum size")
}
