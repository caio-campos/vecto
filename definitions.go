package vecto

import (
	"context"
	"net/http"
	"time"
)

type CertificateConfig struct {
	Cert string
	Key  string
}

type AdapterFunc func(req *Request) (res *Response, err error)

type RequestTransformFunc func(req *Request) (data []byte, err error)

type ValidateStatusFunc func(res *Response) bool

type RequestCompletedCallback func(ev RequestCompletedEvent)

type Config struct {
	BaseURL            string
	Timeout            time.Duration
	Headers            map[string]string
	Certificates       []CertificateConfig
	HTTPTransport      *http.Transport
	Adapter            AdapterFunc
	RequestTransform   RequestTransformFunc
	ValidateStatus     ValidateStatusFunc
	InsecureSkipVerify bool
	Logger             Logger
}

type Client interface {
	Do(ctx context.Context, req *Request) (res *Response, err error)
}

type requestEvents struct {
	completed []RequestCompletedCallback
}

type RequestCompletedEvent struct {
	response *Response
}

func (r *RequestCompletedEvent) Response() *Response {
	return r.response
}

type RequestOptions struct {
	Data             interface{}
	Headers          map[string]string
	Params           map[string]any
	RequestTransform RequestTransformFunc
}
