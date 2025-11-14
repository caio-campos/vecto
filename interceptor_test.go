package vecto

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReqInterceptor(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
	})

	vecto.Interceptors.Request.Use(func(ctx context.Context, req *Request) (resultReq *Request, err error) {
		req.SetHeader("x-custom", "custom")
		return req, nil
	})

	vecto.Interceptors.Request.Use(func(ctx context.Context, req *Request) (resultReq *Request, err error) {
		req.SetHeader("x-another", "another")
		return req, nil
	})

	assert.Len(t, vecto.Interceptors.Request.interceptors, 2)

	res, err := vecto.Post(context.Background(), "/test/custom-header", &RequestOptions{})

	headers := res.request.Headers()
	assert.Equal(t, headers["x-another"], "another")
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestAsyncMultiInterceptor(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
	})

	wg := sync.WaitGroup{}

	vecto.Interceptors.Request.Use(func(ctx context.Context, req *Request) (resultReq *Request, err error) {
		headers := req.Headers()
		req.SetHeader("x-req-id", headers["x-index"])

		return req, nil
	})

	vecto.Interceptors.Request.Use(func(ctx context.Context, req *Request) (resultReq *Request, err error) {
		headers := req.Headers()
		assert.Equal(t, headers["x-req-id"], headers["x-index"])
		return req, nil
	})

	vecto.Interceptors.Request.Use(func(ctx context.Context, req *Request) (resultReq *Request, err error) {
		statusCodeStr := strings.TrimPrefix(req.FullUrl(), srv.URL+"/test/status/")

		statusCode, _ := strconv.Atoi(statusCodeStr)
		req.Completed(func(event RequestCompletedEvent) {
			assert.Equal(t, statusCode, event.response.StatusCode)
			assert.Equal(t, event.Response().Success(), event.response.StatusCode < 300)
		})

		return req, nil
	})

	rand.Seed(time.Now().UnixNano())

	validStatusCodes := []int{
		200, 201, 202, 203, 204, 205, 206, 207, 208, 226,
		300, 301, 302, 303, 304, 305, 306, 307, 308,
		400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415, 416, 417, 418, 421, 422, 423, 424, 425, 426, 428, 429, 431, 451,
		500, 501, 502, 503, 504, 505, 506, 507, 508, 510, 511,
	}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			randomStatusCode := validStatusCodes[rand.Intn(len(validStatusCodes))]
			vecto.Post(context.Background(), "/test/status/"+strconv.Itoa(randomStatusCode), &RequestOptions{
				Headers: map[string]string{
					"x-index": strconv.Itoa(i),
				},
			})
		}(i)
	}

	wg.Wait()
}

func TestResInterceptor(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	var mockData PetMockData

	vecto, _ := New(Config{
		BaseURL: srv.URL,
	})

	vecto.Interceptors.Response.Use(func(ctx context.Context, res *Response) (resultRes *Response, err error) {
		json.Unmarshal(res.Data, &mockData)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "1", mockData.ID)
		assert.Equal(t, "ccampos", mockData.Owner.Login)

		return res, nil
	})

	assert.Len(t, vecto.Interceptors.Response.interceptors, 1)

	res, err := vecto.Get(context.Background(), "/test/pets/1", &RequestOptions{})

	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestReqInterceptorAddQueryParam(t *testing.T) {
	srv := newHTTPTestServer()
	defer srv.Close()

	vecto, _ := New(Config{
		BaseURL: srv.URL,
	})

	// Adiciona um interceptor que adiciona um query param "added_param=1"
	vecto.Interceptors.Request.Use(func(ctx context.Context, req *Request) (resultReq *Request, err error) {
		req.SetParam("added_param", "1")
		return req, nil
	})

	// Certifica-se de que o interceptor foi adicionado
	assert.Len(t, vecto.Interceptors.Request.interceptors, 1)

	// Executa uma requisição GET para um endpoint de teste
	res, err := vecto.Get(context.Background(), "/test/query", &RequestOptions{})

	// Verifica se o parâmetro foi adicionado na URL final
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.Contains(t, res.request.FullUrl(), "added_param=1")
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
