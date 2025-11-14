package vecto

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	// MaxResponseBodySize limita o tamanho do response body para prevenir DoS
	// Default: 100MB
	MaxResponseBodySize = 100 * 1024 * 1024
)

type DefaultClient struct {
	client http.Client
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

	// Limita o tamanho do response body para prevenir DoS
	limitedReader := io.LimitReader(httpRes.Body, MaxResponseBodySize)
	resBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return res, err
	}

	// Verifica se atingiu o limite
	if len(resBody) >= MaxResponseBodySize {
		// Tenta ler mais 1 byte para confirmar que excedeu
		var oneByte [1]byte
		n, _ := httpRes.Body.Read(oneByte[:])
		if n > 0 {
			return res, fmt.Errorf("response body exceeded maximum size of %d bytes", MaxResponseBodySize)
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
