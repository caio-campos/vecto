package vecto

import "fmt"

type ResponseError struct {
	Response *Response
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("request failed: %d - %s", e.Response.StatusCode, e.Response.Data)
}

func (e *ResponseError) Unwrap() error {
	return nil
}
