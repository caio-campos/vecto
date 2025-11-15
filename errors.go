package vecto

import "fmt"

type ResponseError struct {
	Response *Response
	Err      error
}

func (e *ResponseError) Error() string {
	if e.Response == nil {
		if e.Err != nil {
			return fmt.Sprintf("request failed: %v", e.Err)
		}
		return "request failed"
	}

	statusCode := e.Response.StatusCode
	data := e.Response.Data

	if e.Err != nil {
		return fmt.Sprintf("request failed: %d - %s: %v", statusCode, data, e.Err)
	}

	return fmt.Sprintf("request failed: %d - %s", statusCode, data)
}

func (e *ResponseError) Unwrap() error {
	return e.Err
}
