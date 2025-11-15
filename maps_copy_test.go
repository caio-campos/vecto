package vecto

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapsCopyWithNilHeaders(t *testing.T) {
	t.Run("nil config headers", func(t *testing.T) {
		srv := newHTTPTestServer()
		defer srv.Close()

		vecto, err := New(Config{
			BaseURL: srv.URL,
			Headers: nil,
		})
		assert.NoError(t, err)

		res, err := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{
			Headers: map[string]string{
				"X-Custom": "value",
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		headers := res.request.Headers()
		assert.Equal(t, "value", headers["X-Custom"])
	})

	t.Run("nil request options headers", func(t *testing.T) {
		srv := newHTTPTestServer()
		defer srv.Close()

		vecto, err := New(Config{
			BaseURL: srv.URL,
			Headers: map[string]string{
				"X-Config": "config-value",
			},
		})
		assert.NoError(t, err)

		res, err := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{
			Headers: nil,
		})

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		headers := res.request.Headers()
		assert.Equal(t, "config-value", headers["X-Config"])
	})

	t.Run("both nil headers", func(t *testing.T) {
		srv := newHTTPTestServer()
		defer srv.Close()

		vecto, err := New(Config{
			BaseURL: srv.URL,
			Headers: nil,
		})
		assert.NoError(t, err)

		res, err := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{
			Headers: nil,
		})

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("empty maps", func(t *testing.T) {
		srv := newHTTPTestServer()
		defer srv.Close()

		vecto, err := New(Config{
			BaseURL: srv.URL,
			Headers: map[string]string{},
		})
		assert.NoError(t, err)

		res, err := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{
			Headers: map[string]string{},
		})

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}
