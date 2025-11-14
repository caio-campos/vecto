package vecto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

type DefaultClient struct {
	client              http.Client
	maxResponseBodySize int64
	enableTrace         bool
}

func (c *DefaultClient) Do(ctx context.Context, req *Request) (res *Response, err error) {
	var tc *traceContext
	var traceInfo *TraceInfo

	if c.enableTrace {
		tc = &traceContext{}
		trace := createClientTrace(tc)
		ctx = httptrace.WithClientTrace(ctx, trace)
		tc.requestStart = time.Now()
	}

	httpReq, err := req.toHTTPRequest(ctx)
	if err != nil {
		return res, err
	}

	httpRes, err := c.client.Do(httpReq)
	if err != nil {
		if tc != nil {
			tc.requestEnd = time.Now()
		}
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
		if tc != nil {
			tc.requestEnd = time.Now()
		}
		return res, err
	}

	if tc != nil {
		tc.requestEnd = time.Now()
		traceInfo = computeTraceInfo(tc)
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
		TraceInfo:   traceInfo,
	}

	return res, nil
}
