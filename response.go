package vecto

import (
	"bytes"
	"io"
	"net/http"
)

// Response represents an HTTP response.
//
// Thread Safety: Response objects are safe to read concurrently but
// should not be modified after being returned from a request.
type Response struct {
	Data        []byte
	StatusCode  int
	request     *Request
	RawRequest  *http.Request
	RawResponse *http.Response
	success     bool
}

func (r *Response) deepCopy() *Response {
	if r == nil {
		return nil
	}

	dataCopy := make([]byte, len(r.Data))
	copy(dataCopy, r.Data)

	var rawReqCopy *http.Request
	if r.RawRequest != nil {
		rawReqCopy = r.RawRequest.Clone(r.RawRequest.Context())
	}

	var rawResCopy *http.Response
	if r.RawResponse != nil {
		rawResCopy = &http.Response{
			Status:           r.RawResponse.Status,
			StatusCode:       r.RawResponse.StatusCode,
			Proto:            r.RawResponse.Proto,
			ProtoMajor:       r.RawResponse.ProtoMajor,
			ProtoMinor:       r.RawResponse.ProtoMinor,
			Header:           cloneHTTPHeaders(r.RawResponse.Header),
			ContentLength:    r.RawResponse.ContentLength,
			TransferEncoding: cloneStringSlice(r.RawResponse.TransferEncoding),
			Close:            r.RawResponse.Close,
			Uncompressed:     r.RawResponse.Uncompressed,
			Trailer:          cloneHTTPHeaders(r.RawResponse.Trailer),
			Request:          rawReqCopy,
			TLS:              r.RawResponse.TLS,
		}
		
		rawResCopy.Body = io.NopCloser(bytes.NewReader(dataCopy))
	}

	return &Response{
		Data:        dataCopy,
		StatusCode:  r.StatusCode,
		RawRequest:  rawReqCopy,
		RawResponse: rawResCopy,
		request:     r.request,
		success:     r.success,
	}
}

func cloneHTTPHeaders(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}
	
	result := make(http.Header, len(headers))
	for k, v := range headers {
		result[k] = cloneStringSlice(v)
	}
	return result
}

func cloneStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	
	result := make([]string, len(s))
	copy(result, s)
	return result
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
