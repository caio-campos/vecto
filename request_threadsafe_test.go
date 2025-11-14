package vecto

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestThreadSafety(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
	})

	t.Run("concurrent reads", func(t *testing.T) {
		req, err := vecto.newRequest("/test/pets/1", "GET", nil)
		assert.NoError(t, err)

		wg := sync.WaitGroup{}
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = req.FullUrl()
				_ = req.Method()
				_ = req.Host()
				_ = req.Scheme()
				_ = req.Path()
				_ = req.BaseUrl()
				_ = req.Headers()
				_ = req.Params()
				_ = req.Data()
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent writes", func(t *testing.T) {
		req, err := vecto.newRequest("/test/pets/1", "GET", nil)
		assert.NoError(t, err)

		initialHeaders := len(req.Headers())

		wg := sync.WaitGroup{}
		errCh := make(chan error, 200)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				req.SetHeader("x-thread-"+strconv.Itoa(idx), strconv.Itoa(idx))
				if err := req.SetParam("param-"+strconv.Itoa(idx), idx); err != nil {
					errCh <- err
				}
			}(i)
		}
		wg.Wait()
		close(errCh)

		for err := range errCh {
			assert.NoError(t, err)
		}

		headers := req.Headers()
		params := req.Params()

		assert.Equal(t, 100+initialHeaders, len(headers))
		assert.Equal(t, 100, len(params))
	})

	t.Run("mixed reads and writes", func(t *testing.T) {
		req, err := vecto.newRequest("/test/pets/1", "GET", nil)
		assert.NoError(t, err)

		wg := sync.WaitGroup{}

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				req.SetHeader("x-writer-"+strconv.Itoa(idx), strconv.Itoa(idx))
			}(i)
		}

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = req.Headers()
				_ = req.Method()
				_ = req.FullUrl()
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent requests", func(t *testing.T) {
		wg := sync.WaitGroup{}
		errCh := make(chan error, 200)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				req, err := vecto.newRequest("/test/status/200", "GET", &RequestOptions{
					Headers: map[string]string{
						"x-thread": strconv.Itoa(idx),
					},
				})
				if err != nil {
					errCh <- err
					return
				}

				if err := req.SetParam("thread_id", idx); err != nil {
					errCh <- err
					return
				}

				res, err := vecto.client.Do(context.Background(), req)
				if err != nil {
					errCh <- err
					return
				}
				if res.StatusCode != 200 {
					errCh <- fmt.Errorf("unexpected status: %d", res.StatusCode)
				}
			}(i)
		}

		wg.Wait()
		close(errCh)

		for err := range errCh {
			assert.NoError(t, err)
		}
	})
}
