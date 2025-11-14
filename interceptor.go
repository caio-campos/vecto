package vecto

import (
	"context"
	"sync"
)

type ReqInterceptorFunc func(ctx context.Context, req *Request) (resultReq *Request, err error)
type ResInterceptorFunc func(ctx context.Context, res *Response) (resultRes *Response, err error)

type interceptorCollectionWrapper struct {
	Request  reqInterceptorCollection
	Response resInterceptorCollection
}

type reqInterceptorCollection struct {
	mu           sync.RWMutex
	interceptors []ReqInterceptorFunc
}

func (c *reqInterceptorCollection) Use(interceptor ReqInterceptorFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, interceptor)
}

func (c *reqInterceptorCollection) getAll() []ReqInterceptorFunc {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Retorna cópia para evitar race conditions
	result := make([]ReqInterceptorFunc, len(c.interceptors))
	copy(result, c.interceptors)
	return result
}

type resInterceptorCollection struct {
	mu           sync.RWMutex
	interceptors []ResInterceptorFunc
}

func (c *resInterceptorCollection) Use(interceptor ResInterceptorFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, interceptor)
}

func (c *resInterceptorCollection) getAll() []ResInterceptorFunc {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Retorna cópia para evitar race conditions
	result := make([]ResInterceptorFunc, len(c.interceptors))
	copy(result, c.interceptors)
	return result
}
