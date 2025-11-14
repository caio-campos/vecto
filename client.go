package vecto

import (
	"context"
	"io"
	"net/http"
)

type DefaultClient struct {
	client http.Client
}

func (c *DefaultClient) Do(ctx context.Context, req Request) (res *Response, err error) {
	reqCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	httpReq, err := req.toHTTPRequest(reqCtx)
	if err != nil {
		return res, err
	}

	req.rawRequest = httpReq

	httpRes, err := c.client.Do(httpReq)
	if err != nil {
		return res, err
	}

	defer httpRes.Body.Close()

	resBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return res, err
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
