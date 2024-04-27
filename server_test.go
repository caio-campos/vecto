package vecto

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"
)

func newHTTPTestServer() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/test/status/") {
				statusCodeStr := strings.TrimPrefix(r.URL.Path, "/test/status/")
				statusCode, err := strconv.Atoi(statusCodeStr)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if statusCode < 100 || statusCode > 599 {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if statusCode == 200 {
					w.WriteHeader(http.StatusOK)
					return
				}

				w.WriteHeader(statusCode)
				return
			}

			if r.URL.Path == "/test/methods" {
				w.Header().Add("Content-Type", "application/json")

				if _, ok := supportedMethods[r.Method]; ok {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusBadRequest)
				}

				return
			}

			if r.Method == http.MethodPost && r.URL.Path == "/test/custom-header" {
				w.Header().Add("Content-Type", "application/json")

				if r.Header.Get("x-custom") == "custom" {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusBadRequest)
				}

				return
			}

			if r.Method == http.MethodGet && r.URL.Path == "/test/pets/1" {
				w.Header().Add("Content-Type", "application/json")
				w.Write([]byte(`{"id": "1", "name":"Little Tony","owner":{"login": "ccampos"}}`))
				return
			}

			if r.URL.Path == "/test/slow" {
				time.Sleep(time.Second * 6)
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}),
	)
}
