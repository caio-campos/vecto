package vecto

import (
	"context"
	"sync"
)

type RequestMiddlewareFunc func(ctx context.Context, req *Request) (resultReq *Request, err error)
type ResponseMiddlewareFunc func(ctx context.Context, res *Response) (resultRes *Response, err error)

type middlewareCollection struct {
	mu         sync.RWMutex
	requestMW  []RequestMiddlewareFunc
	responseMW []ResponseMiddlewareFunc
}

func newMiddlewareCollection() *middlewareCollection {
	return &middlewareCollection{
		requestMW:  make([]RequestMiddlewareFunc, 0, 4),
		responseMW: make([]ResponseMiddlewareFunc, 0, 4),
	}
}

func (c *middlewareCollection) addRequest(mw RequestMiddlewareFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestMW = append(c.requestMW, mw)
}

func (c *middlewareCollection) addResponse(mw ResponseMiddlewareFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responseMW = append(c.responseMW, mw)
}

func (c *middlewareCollection) getRequest() []RequestMiddlewareFunc {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.requestMW) == 0 {
		return nil
	}

	result := make([]RequestMiddlewareFunc, len(c.requestMW))
	copy(result, c.requestMW)
	return result
}

func (c *middlewareCollection) getResponse() []ResponseMiddlewareFunc {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.responseMW) == 0 {
		return nil
	}

	result := make([]ResponseMiddlewareFunc, len(c.responseMW))
	copy(result, c.responseMW)
	return result
}
