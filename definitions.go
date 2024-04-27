package vecto

import (
	"bytes"
	"context"
	"io/ioutil"
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
	Url              string
	Method           string
	Params           map[string]any
	Headers          map[string]string
	Data             interface{}
	requestTransform RequestTransformFunc
	rawRequest       *http.Request
	events           requestEvents
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
	r.rawRequest.URL, _ = url.Parse(r.Url)
	r.rawRequest.Body = ioutil.NopCloser(bytes.NewReader(httpReqData))

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
