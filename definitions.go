package vecto

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"
)

type CertificateConfig struct {
	Cert string
	Key  string
}

type AdapterFunc func(req Request) (res *Response, err error)

type RequestTransformFunc func(req Request) (data []byte, err error)

type ValidateStatusFunc func(res *Response) bool

type RequestCompletedCallback func(ev RequestCompletedEvent)

type Config struct {
	BaseURL          string
	Timeout          time.Duration
	Headers          map[string]string
	Certificates     []CertificateConfig
	HTTPTransport    *http.Transport
	Adapter          AdapterFunc
	RequestTransform RequestTransformFunc
	ValidateStatus   ValidateStatusFunc
}

type Client interface {
	Do(ctx context.Context, req Request) (res *Response, err error)
}

type requestEvents struct {
	completed []RequestCompletedCallback
}

type Request struct {
	BaseUrl          string               // The Base URL of the request, excluding query parameters
	FullUrl          string               // The Full URL of the request, including query parameters
	Host             string               // The Host component of the URL (e.g., "example.com")
	Scheme           string               // The Scheme of the URL (e.g., "http", "https")
	Path             string               // The Path component of the URL, specifying the specific resource location
	Method           string               // The HTTP method used for the request (e.g., "GET", "POST")
	Params           map[string]any       // A map containing query parameters to be sent with the request
	Headers          map[string]string    // A map of HTTP headers to be sent with the request
	Data             interface{}          // The body of the request, used for POST and PUT requests
	requestTransform RequestTransformFunc // A function to transform the request before sending
	rawRequest       *http.Request        // The raw HTTP request object, used internally
	events           requestEvents        // Internal event hooks for the request lifecycle (e.g., on completed)
}

func (r *Request) Completed(cb RequestCompletedCallback) {
	r.events.completed = append(r.events.completed, cb)
}

func (r *Request) RawRequest() *http.Request {
	return r.rawRequest
}

type RequestCompletedEvent struct {
	response *Response
}

func (r *RequestCompletedEvent) Response() *Response {
	return r.response
}

func (r *Request) ToHTTPRequest(ctx context.Context) (httpReq *http.Request, err error) {
	var httpReqData []byte

	httpReqData, err = r.requestTransform(*r)
	if err != nil {
		return nil, err
	}

	r.rawRequest = r.rawRequest.WithContext(ctx)
	r.rawRequest.Method = r.Method
	r.rawRequest.URL, _ = url.Parse(r.FullUrl)
	r.rawRequest.Body = io.NopCloser(bytes.NewReader(httpReqData))

	r.attachHeadersToHttpReq(r.rawRequest)

	return r.rawRequest, nil
}

func (r *Request) attachHeadersToHttpReq(httpReq *http.Request) {
	header := make(http.Header)

	for key, value := range r.Headers {
		header[key] = []string{value}
	}

	httpReq.Header = header
}

type RequestOptions struct {
	Data             interface{}
	Headers          map[string]string
	Params           map[string]any
	RequestTransform RequestTransformFunc
}

type Response struct {
	Data        []byte
	StatusCode  int
	request     Request
	RawRequest  *http.Request
	RawResponse *http.Response
	success     bool
}

func (r *Response) Success() bool {
	return r.success
}

func (r *Response) RequestFailedError() error {
	if r.success {
		return nil
	}

	return &ResponseError{
		Response: r,
	}
}
