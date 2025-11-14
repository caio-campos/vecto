package vecto

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"dario.cat/mergo"
)

// Vecto is the main HTTP client wrapper.
//
// Thread Safety: The Vecto instance itself is safe for concurrent use.
// Multiple goroutines can safely call Request methods on the same Vecto instance.
// However, Request and Response objects should not be shared between goroutines
// or modified after being passed to request methods.
type Vecto struct {
	config       Config
	client       Client
	logger       Logger
	Interceptors interceptorCollectionWrapper
}

var defaultConfig = Config{
	Timeout: time.Second * 30,
	Headers: map[string]string{
		"Content-Type": "application/json",
	},
	ValidateStatus: func(res *Response) bool {
		if res == nil {
			return false
		}

		return res.StatusCode >= 200 && res.StatusCode < 300
	},
}

func New(config Config) (v *Vecto, err error) {
	if err = mergo.Merge(&config, defaultConfig); err != nil {
		return nil, err
	}

	instance := Vecto{
		config: config,
	}

	if config.Logger == nil {
		instance.logger = newNoopLogger()
	} else {
		instance.logger = config.Logger
	}

	err = instance.setHttpClient()
	if err != nil {
		return nil, err
	}

	return &instance, nil
}

func (v *Vecto) Post(ctx context.Context, url string, options *RequestOptions) (res *Response, err error) {
	return v.Request(ctx, url, http.MethodPost, options)
}

func (v *Vecto) Patch(ctx context.Context, url string, options *RequestOptions) (res *Response, err error) {
	return v.Request(ctx, url, http.MethodPatch, options)
}

func (v *Vecto) Put(ctx context.Context, url string, options *RequestOptions) (res *Response, err error) {
	return v.Request(ctx, url, http.MethodPut, options)
}

func (v *Vecto) Get(ctx context.Context, url string, options *RequestOptions) (res *Response, err error) {
	return v.Request(ctx, url, http.MethodGet, options)
}

func (v *Vecto) Delete(ctx context.Context, url string, options *RequestOptions) (res *Response, err error) {
	return v.Request(ctx, url, http.MethodDelete, options)
}

func (v *Vecto) Head(ctx context.Context, url string, options *RequestOptions) (res *Response, err error) {
	return v.Request(ctx, url, http.MethodHead, options)
}

func (v *Vecto) Options(ctx context.Context, url string, options *RequestOptions) (res *Response, err error) {
	return v.Request(ctx, url, http.MethodOptions, options)
}

func (v *Vecto) Request(ctx context.Context, url string, method string, options *RequestOptions) (res *Response, err error) {
	request, err := v.newRequest(url, method, options)
	if err != nil {
		v.logger.Error(ctx, "failed to create request", map[string]interface{}{
			"url":    url,
			"method": method,
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	v.logger.Debug(ctx, "request created", map[string]interface{}{
		"url":    request.FullUrl(),
		"method": request.Method(),
	})

	if v.config.Adapter != nil {
		return v.config.Adapter(request)
	}

	request, err = v.interceptRequest(ctx, request)
	if err != nil {
		v.logger.Error(ctx, "request interceptor failed", map[string]interface{}{
			"url":   request.FullUrl(),
			"error": err.Error(),
		})
		return nil, fmt.Errorf("request interceptor failed: %w", err)
	}

	res, err = v.client.Do(ctx, request)
	if err != nil {
		v.logger.Error(ctx, "http request failed", map[string]interface{}{
			"url":    request.FullUrl(),
			"method": request.Method(),
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	v.logger.Info(ctx, "request completed", map[string]interface{}{
		"url":         request.FullUrl(),
		"method":      request.Method(),
		"status_code": res.StatusCode,
	})

	res.success = v.config.ValidateStatus(res)
	resultRes, err := v.interceptResponse(ctx, res)
	if err != nil {
		v.logger.Error(ctx, "response interceptor failed", map[string]interface{}{
			"url":         request.FullUrl(),
			"status_code": res.StatusCode,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("response interceptor failed: %w", err)
	}

	v.dispatchRequestCompleted(resultRes)

	return resultRes, nil
}

func (v *Vecto) dispatchRequestCompleted(res *Response) {
	event := RequestCompletedEvent{
		response: res,
	}

	requestUrl := res.request.FullUrl()
	
	res.request.mu.RLock()
	callbacks := make([]RequestCompletedCallback, len(res.request.events.completed))
	copy(callbacks, res.request.events.completed)
	res.request.mu.RUnlock()

	for _, cb := range callbacks {
		go func(callback RequestCompletedCallback, url string) {
			defer func() {
				if r := recover(); r != nil {
					v.logger.Error(context.Background(), "panic in request completed callback", map[string]interface{}{
						"panic": fmt.Sprintf("%v", r),
						"url":   url,
					})
				}
			}()
			callback(event)
		}(cb, requestUrl)
	}
}

func (v *Vecto) interceptRequest(ctx context.Context, req *Request) (resultReq *Request, err error) {
	resultReq = req
	for _, interceptor := range v.Interceptors.Request.interceptors {
		resultReq, err = interceptor(ctx, resultReq)
		if err != nil {
			return req, err
		}
	}

	return resultReq, nil
}

func (v *Vecto) interceptResponse(ctx context.Context, res *Response) (resultRes *Response, err error) {
	if res == nil {
		return res, nil
	}

	resultRes = res
	for _, interceptor := range v.Interceptors.Response.interceptors {
		resultRes, err = interceptor(ctx, resultRes)
		if err != nil {
			return resultRes, err
		}
	}

	return resultRes, nil
}

func (v *Vecto) newRequest(urlStr string, method string, options *RequestOptions) (*Request, error) {
	reqOptions := RequestOptions{}
	if options != nil {
		reqOptions = *options
	}

	fullUrlStr := fmt.Sprintf("%s%s", v.config.BaseURL, urlStr)

	builder := newRequestBuilder(fullUrlStr, method).
		SetHeaders(v.config.Headers).
		SetHeaders(reqOptions.Headers).
		SetData(reqOptions.Data).
		SetTransform(ApplicationJsonReqTransformer)

	if reqOptions.RequestTransform != nil {
		builder.SetTransform(reqOptions.RequestTransform)
	}

	for key, value := range reqOptions.Params {
		builder.SetParam(key, value)
	}

	req, err := builder.Build()
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (v *Vecto) setHttpClient() (err error) {
	client, err := newDefaultClient(v)
	if err != nil {
		return err
	}

	v.client = client

	return nil
}
