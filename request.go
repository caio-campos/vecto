package vecto

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
)

// Request represents an HTTP request configuration.
//
// Thread Safety: Request objects are safe for concurrent use.
// All public methods are protected by an internal mutex to ensure
// thread-safe operations across multiple goroutines.
type Request struct {
	mu        sync.RWMutex
	baseURL   string
	url       string
	host      string
	scheme    string
	path      string
	method    string
	params    map[string]any
	headers   map[string]string
	data      interface{}
	transform RequestTransformFunc
	rawReq    *http.Request
	events    requestEvents
}

// Completed adds a callback function that is triggered when the request completes.
// This function allows external code to handle events or perform actions after the request has finished processing.
func (r *Request) Completed(cb RequestCompletedCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events.completed = append(r.events.completed, cb)
}

// RawRequest returns the underlying *http.Request object.
// It provides access to the raw HTTP request used for making the network call, which might be needed for advanced customizations.
func (r *Request) RawRequest() *http.Request {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rawReq
}

// SetParam adds or updates a query parameter for the request and returns
// an error if the resulting URL cannot be constructed.
func (r *Request) SetParam(key string, value any) error {
	if key == "" {
		return fmt.Errorf("param key cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.params == nil {
		r.params = make(map[string]any)
	}
	r.params[key] = value
	return r.refreshUrlUnsafe()
}

func (r *Request) SetHeaders(headers map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range headers {
		r.setHeaderUnsafe(key, value)
	}
}

// SetHeader adds or updates a header for the request.
// This allows custom headers to be set, such as authentication tokens or content types.
func (r *Request) SetHeader(key, value string) {
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setHeaderUnsafe(key, value)
}

func (r *Request) setHeaderUnsafe(key, value string) {
	if r.headers == nil {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
}

// refreshUrl rebuilds the request URL based on the base path and current query parameters.
// It also updates the raw HTTP request object to ensure it reflects the latest URL.
// Returns an error if the URL cannot be constructed.
func (r *Request) refreshUrl() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.refreshUrlUnsafe()
}

func (r *Request) refreshUrlUnsafe() error {
	fullUrl, err := getUrlInstance(r.baseURL, r.params)
	if err != nil {
		return err
	}

	r.url = fullUrl.String()
	r.host = fullUrl.Host
	r.scheme = fullUrl.Scheme
	r.path = fullUrl.Path

	newRequest, err := http.NewRequest(r.method, r.url, bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}

	r.rawReq = newRequest

	return nil
}

// BaseUrl returns the base path of the request, which is the base URL without query parameters.
// This can be useful for logging or debugging the endpoint being accessed.
func (r *Request) BaseUrl() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.baseURL
}

// FullUrl returns the full request URL, including any query parameters.
// This represents the actual URL that will be used for the HTTP request.
func (r *Request) FullUrl() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.url
}

// Host returns the host component of the request URL.
// This includes the domain or IP address where the request is being sent.
func (r *Request) Host() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.host
}

// Scheme returns the scheme of the request URL (e.g., "http" or "https").
// It indicates the protocol being used for the request.
func (r *Request) Scheme() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scheme
}

// Path returns the path component of the request URL.
// It specifies the resource location on the server without including query parameters.
func (r *Request) Path() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.path
}

// Method returns the HTTP method used for the request (e.g., "GET", "POST").
// This indicates the type of action being performed on the specified resource.
func (r *Request) Method() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.method
}

// Data returns the body of the request, typically used for POST and PUT requests.
// It can be any type of data that will be serialized and sent as the request payload.
func (r *Request) Data() interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.data
}

// Headers returns the headers set for the request.
// It provides access to all HTTP headers that will be sent with the request.
// Returns a copy of the headers map to prevent external modifications.
func (r *Request) Headers() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	headersCopy := make(map[string]string, len(r.headers))
	for k, v := range r.headers {
		headersCopy[k] = v
	}
	return headersCopy
}

// Params returns the current set of query parameters for the request.
// It allows access to the key-value pairs that will be included in the request URL.
// Returns a copy of the params map to prevent external modifications.
func (r *Request) Params() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paramsCopy := make(map[string]any, len(r.params))
	for k, v := range r.params {
		paramsCopy[k] = v
	}
	return paramsCopy
}

func (r *Request) toHTTPRequest(ctx context.Context) (*http.Request, error) {
	var httpReqData []byte
	var err error

	if httpReqData, err = r.transform(r); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	newRequest, err := http.NewRequest(r.method, r.url, bytes.NewReader(httpReqData))
	if err != nil {
		return nil, err
	}

	newRequest = newRequest.WithContext(ctx)
	r.rawReq = newRequest

	r.attachHeadersToHttpReqUnsafe(r.rawReq)

	return r.rawReq, nil
}

func (r *Request) attachHeadersToHttpReqUnsafe(httpReq *http.Request) {
	header := make(http.Header)

	for key, value := range r.headers {
		header[key] = []string{value}
	}

	httpReq.Header = header
}
