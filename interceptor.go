package vecto

import "context"

type ReqInterceptorFunc func(ctx context.Context, req *Request) (resultReq *Request, err error)
type ResInterceptorFunc func(ctx context.Context, res *Response) (resultRes *Response, err error)

type interceptorCollectionWrapper struct {
	Request  reqInterceptorCollection
	Response resInterceptorCollection
}

type reqInterceptorCollection struct {
	interceptors []ReqInterceptorFunc
}

func (c *reqInterceptorCollection) Use(interceptor ReqInterceptorFunc) {
	c.interceptors = append(c.interceptors, interceptor)
}

type resInterceptorCollection struct {
	interceptors []ResInterceptorFunc
}

func (c *resInterceptorCollection) Use(interceptor ResInterceptorFunc) {
	c.interceptors = append(c.interceptors, interceptor)
}
