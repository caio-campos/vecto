package vecto

import "net/http"

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
