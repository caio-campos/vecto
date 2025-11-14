package vecto

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type DefaultClient struct {
	client            http.Client
	maxResponseBodySize int64
}

func (c *DefaultClient) Do(ctx context.Context, req *Request) (res *Response, err error) {
	httpReq, err := req.toHTTPRequest(ctx)
	if err != nil {
		return res, err
	}

	httpRes, err := c.client.Do(httpReq)
	if err != nil {
		return res, err
	}

	defer httpRes.Body.Close()

	maxSize := c.maxResponseBodySize
	if maxSize <= 0 {
		maxSize = 100 * 1024 * 1024
	}

	limitedReader := io.LimitReader(httpRes.Body, maxSize)
	resBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return res, err
	}

	if int64(len(resBody)) >= maxSize {
		var oneByte [1]byte
		n, _ := httpRes.Body.Read(oneByte[:])
		if n > 0 {
			return res, fmt.Errorf("response body exceeded maximum size of %d bytes", maxSize)
		}
	}

	res = &Response{
		Data:        resBody,
		StatusCode:  httpRes.StatusCode,
		RawRequest:  httpReq,
		RawResponse: httpRes,
		request:     req,
	}

	return res, nil
}
