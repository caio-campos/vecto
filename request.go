package vecto

import (
	"bytes"
	"context"
	"net/http"
)

type Request struct {
	basePath         string
	requestUrl       string
	host             string
	scheme           string
	path             string
	method           string
	params           map[string]any
	headers          map[string]string
	data             interface{}
	requestTransform RequestTransformFunc
	rawRequest       *http.Request
	events           requestEvents
}

// Completed adds a callback function that is triggered when the request completes.
// This function allows external code to handle events or perform actions after the request has finished processing.
func (r *Request) Completed(cb RequestCompletedCallback) {
	r.events.completed = append(r.events.completed, cb)
}

// RawRequest returns the underlying *http.Request object.
// It provides access to the raw HTTP request used for making the network call, which might be needed for advanced customizations.
func (r *Request) RawRequest() *http.Request {
	return r.rawRequest
}

// SetParam adds or updates a query parameter for the request.
// The key-value pair provided is added to the list of query parameters,
// and the URL is refreshed to include the updated parameters.
func (r *Request) SetParam(key string, value any) {
	if r.params == nil {
		r.params = make(map[string]any)
	}
	r.params[key] = value
	r.refreshUrl()
}

func (r *Request) SetHeaders(headers map[string]string) {
	for key, value := range headers {
		r.SetHeader(key, value)
	}
}

// SetHeader adds or updates a header for the request.
// This allows custom headers to be set, such as authentication tokens or content types.
func (r *Request) SetHeader(key, value string) {
	if r.headers == nil {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
}

// refreshUrl rebuilds the request URL based on the base path and current query parameters.
// It also updates the raw HTTP request object to ensure it reflects the latest URL.
// Returns an error if the URL cannot be constructed.
func (r *Request) refreshUrl() error {
	fullUrl, err := getUrlInstance(r.basePath, r.params)
	if err != nil {
		return err
	}

	r.requestUrl = fullUrl.String()
	r.host = fullUrl.Host
	r.scheme = fullUrl.Scheme
	r.path = fullUrl.Path

	newRequest, err := http.NewRequest(r.method, r.requestUrl, bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}

	r.rawRequest = newRequest

	return nil
}

// BaseUrl returns the base path of the request, which is the base URL without query parameters.
// This can be useful for logging or debugging the endpoint being accessed.
func (r *Request) BaseUrl() string {
	return r.basePath
}

// FullUrl returns the full request URL, including any query parameters.
// This represents the actual URL that will be used for the HTTP request.
func (r *Request) FullUrl() string {
	return r.requestUrl
}

// Host returns the host component of the request URL.
// This includes the domain or IP address where the request is being sent.
func (r *Request) Host() string {
	return r.host
}

// Scheme returns the scheme of the request URL (e.g., "http" or "https").
// It indicates the protocol being used for the request.
func (r *Request) Scheme() string {
	return r.scheme
}

// Path returns the path component of the request URL.
// It specifies the resource location on the server without including query parameters.
func (r *Request) Path() string {
	return r.path
}

// Method returns the HTTP method used for the request (e.g., "GET", "POST").
// This indicates the type of action being performed on the specified resource.
func (r *Request) Method() string {
	return r.method
}

// Data returns the body of the request, typically used for POST and PUT requests.
// It can be any type of data that will be serialized and sent as the request payload.
func (r *Request) Data() interface{} {
	return r.data
}

// Headers returns the headers set for the request.
// It provides access to all HTTP headers that will be sent with the request.
func (r *Request) Headers() map[string]string {
	return r.headers
}

// Params returns the current set of query parameters for the request.
// It allows access to the key-value pairs that will be included in the request URL.
func (r *Request) Params() map[string]any {
	return r.params
}

func (r *Request) toHTTPRequest(ctx context.Context) (*http.Request, error) {
	var httpReqData []byte
	var err error

	if httpReqData, err = r.requestTransform(*r); err != nil {
		return nil, err
	}

	newRequest, err := http.NewRequest(r.method, r.requestUrl, bytes.NewReader(httpReqData))
	if err != nil {
		return nil, err
	}

	newRequest = newRequest.WithContext(ctx)
	r.rawRequest = newRequest

	r.attachHeadersToHttpReq(r.rawRequest)

	return r.rawRequest, nil
}

func (r *Request) attachHeadersToHttpReq(httpReq *http.Request) {
	header := make(http.Header)

	for key, value := range r.headers {
		header[key] = []string{value}
	}

	httpReq.Header = header
}
