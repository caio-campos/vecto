package vecto

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"
)

// Vecto is the main HTTP client wrapper.
//
// Thread Safety: The Vecto instance itself is safe for concurrent use.
// Multiple goroutines can safely call Request methods on the same Vecto instance.
// However, Request and Response objects should not be shared between goroutines
// or modified after being passed to request methods.
type Vecto struct {
	config             Config
	client             Client
	logger             Logger
	middleware         *middlewareCollection
	circuitBreakerMgr  *CircuitBreakerManager
	callbackDispatcher *callbackDispatcher
	requestHandler     *requestHandler
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
	MaxResponseBodySize:    100 * 1024 * 1024,
	MaxConcurrentCallbacks: 100,
	CallbackTimeout:        30 * time.Second,
}

func New(config Config) (v *Vecto, err error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	mergedConfig := mergeConfig(config, defaultConfig)

	instance := Vecto{
		config: mergedConfig,
	}

	if mergedConfig.Logger == nil {
		instance.logger = newNoopLogger()
	} else {
		instance.logger = mergedConfig.Logger
	}

	if mergedConfig.CircuitBreaker != nil {
		cbConfig := *mergedConfig.CircuitBreaker
		if cbConfig.Logger == nil {
			cbConfig.Logger = instance.logger
		}
		instance.circuitBreakerMgr = NewCircuitBreakerManager(cbConfig, instance.logger)
	}

	err = instance.setHTTPClient()
	if err != nil {
		return nil, err
	}

	instance.middleware = newMiddlewareCollection()
	instance.callbackDispatcher = newCallbackDispatcher(instance.logger, mergedConfig)
	instance.requestHandler = newRequestHandler(&instance)

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
	startTime := time.Now()

	if ctx == nil {
		ctx = context.Background()
	}

	request, err := v.newRequest(url, method, options)
	if err != nil {
		if !v.logger.IsNoop() {
			v.logger.Error(ctx, "failed to create request", map[string]interface{}{
				"url":    url,
				"method": method,
				"error":  err.Error(),
			})
		}
		v.recordMetricsWithFallback(ctx, method, url, nil, nil, time.Since(startTime), err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if !v.logger.IsNoop() {
		v.logger.Debug(ctx, "request created", map[string]interface{}{
			"url":    request.FullUrl(),
			"method": request.Method(),
		})
	}

	if v.config.Adapter != nil {
		result, adapterErr := v.config.Adapter(request)
		v.recordMetrics(ctx, request, result, time.Since(startTime), adapterErr)
		return result, adapterErr
	}

	request, err = v.interceptRequest(ctx, request)
	if err != nil {
		if !v.logger.IsNoop() {
			v.logger.Error(ctx, "request middleware failed", map[string]interface{}{
				"url":   request.FullUrl(),
				"error": err.Error(),
			})
		}
		v.recordMetrics(ctx, request, nil, time.Since(startTime), err)
		return nil, fmt.Errorf("request middleware failed: %w", err)
	}

	retryConfig := v.getRetryConfig(options)

	var breaker *CircuitBreaker
	var cbKey string

	if v.circuitBreakerMgr != nil {
		cbKey = v.requestHandler.getOrSetCircuitBreakerKey(request)
		breaker = v.circuitBreakerMgr.GetOrCreate(cbKey, nil)

		res, err = breaker.Execute(ctx, func() (*Response, error) {
			return v.requestHandler.executeRequest(ctx, request, retryConfig, breaker)
		})

		if err != nil {
			if _, isCbError := err.(*CircuitBreakerError); isCbError {
				return v.requestHandler.handleCircuitBreakerError(ctx, request, cbKey, breaker, startTime, err)
			}
			return v.requestHandler.handleRequestError(ctx, request, method, startTime, err)
		}
	} else {
		res, err = v.requestHandler.executeRequest(ctx, request, retryConfig, nil)
		if err != nil {
			return v.requestHandler.handleRequestError(ctx, request, method, startTime, err)
		}
	}

	duration := time.Since(startTime)

	if !v.logger.IsNoop() {
		v.logger.Info(ctx, "request completed", map[string]interface{}{
			"url":         request.FullUrl(),
			"method":      request.Method(),
			"status_code": res.StatusCode,
		})
	}

	res.success = v.config.ValidateStatus(res)

	if v.circuitBreakerMgr != nil {
		if breaker != nil {
			breaker.RecordResult(res, nil)
		}
	}

	resultRes, err := v.interceptResponse(ctx, res)
	if err != nil {
		if !v.logger.IsNoop() {
			v.logger.Error(ctx, "response middleware failed", map[string]interface{}{
				"url":         request.FullUrl(),
				"status_code": res.StatusCode,
				"error":       err.Error(),
			})
		}
		v.recordMetrics(ctx, request, res, duration, err)
		return nil, fmt.Errorf("response middleware failed: %w", err)
	}

	if v.config.DebugMode {
		v.writeDebugOutput(request, resultRes)
	}

	v.callbackDispatcher.dispatch(ctx, resultRes)

	v.recordMetrics(ctx, request, resultRes, duration, nil)

	return resultRes, nil
}

func (v *Vecto) interceptRequest(ctx context.Context, req *Request) (resultReq *Request, err error) {
	resultReq = req
	for _, mw := range v.middleware.getRequest() {
		resultReq, err = mw(ctx, resultReq)
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
	for _, mw := range v.middleware.getResponse() {
		resultRes, err = mw(ctx, resultRes)
		if err != nil {
			return resultRes, err
		}
	}

	return resultRes, nil
}

// UseRequest adds a middleware function that will be executed before each request is sent.
// Middleware functions are executed in the order they were added.
// If a middleware returns an error, the request chain is stopped and the error is returned.
//
// Example:
//
//	vecto.UseRequest(func(ctx context.Context, req *Request) (*Request, error) {
//	    req.SetHeader("X-Custom", "value")
//	    return req, nil
//	})
func (v *Vecto) UseRequest(mw RequestMiddlewareFunc) {
	v.middleware.addRequest(mw)
}

// UseResponse adds a middleware function that will be executed after each response is received.
// Middleware functions are executed in the order they were added.
// If a middleware returns an error, the response chain is stopped and the error is returned.
//
// Example:
//
//	vecto.UseResponse(func(ctx context.Context, res *Response) (*Response, error) {
//	    // Process response
//	    return res, nil
//	})
func (v *Vecto) UseResponse(mw ResponseMiddlewareFunc) {
	v.middleware.addResponse(mw)
}

func (v *Vecto) newRequest(urlStr string, method string, options *RequestOptions) (*Request, error) {
	reqOptions := RequestOptions{}
	if options != nil {
		reqOptions = *options
	}

	urlStr = sanitizeURL(urlStr)
	if urlStr == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	fullUrlStr := v.config.BaseURL + urlStr

	if err := validateURL(fullUrlStr); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if reqOptions.PathParams != nil {
		fullUrlStr = replacePathParams(fullUrlStr, reqOptions.PathParams)
	}

	transform := ApplicationJsonReqTransformer
	if v.config.RequestTransform != nil {
		transform = v.config.RequestTransform
	}
	if reqOptions.RequestTransform != nil {
		transform = reqOptions.RequestTransform
	}

	data := reqOptions.Data
	headers := make(map[string]string)

	if v.config.Headers != nil {
		maps.Copy(headers, v.config.Headers)
	}
	if reqOptions.Headers != nil {
		maps.Copy(headers, reqOptions.Headers)
	}

	if reqOptions.FormData != nil {
		data = encodeFormData(reqOptions.FormData)
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	builder := newRequestBuilder(fullUrlStr, method).
		SetHeaders(headers).
		SetData(data).
		SetTransform(transform)

	if reqOptions.QueryStruct != nil {
		params, err := structToQueryParams(reqOptions.QueryStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to parse query struct: %w", err)
		}
		for key, value := range params {
			builder.SetParam(key, value)
		}
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

func (v *Vecto) getCircuitBreakerKey(req *Request) string {
	scheme := req.Scheme()
	host := req.Host()

	if host == "" {
		return "default"
	}

	if scheme == "" {
		return host
	}

	return scheme + "://" + host
}

func (v *Vecto) setHTTPClient() (err error) {
	client, err := newDefaultClient(v)
	if err != nil {
		return err
	}

	v.client = client

	return nil
}

func (v *Vecto) getRetryConfig(options *RequestOptions) *RetryConfig {
	if v.config.Retry == nil {
		return nil
	}

	config := *v.config.Retry

	if options != nil && options.MaxRetries != nil {
		config.MaxAttempts = *options.MaxRetries
	}

	return &config
}

func shouldUseRetry(breaker *CircuitBreaker) bool {
	if breaker == nil {
		return true
	}

	state := breaker.GetState()
	return state != StateOpen
}

func (v *Vecto) writeDebugOutput(req *Request, res *Response) {
	if req == nil || res == nil {
		return
	}

	var trace *TraceInfo
	if v.config.EnableTrace && res.TraceInfo != nil {
		trace = res.TraceInfo
	}

	writer := v.logger
	if !writer.IsNoop() {
		debugStr := formatDebugInfo(req, res, trace)
		ctx := context.Background()
		v.logger.Debug(ctx, debugStr, nil)
	}
}

func formatDebugInfo(req *Request, res *Response, trace *TraceInfo) string {
	var b strings.Builder

	b.WriteString("\n=== DEBUG INFO ===\n")
	b.WriteString(fmt.Sprintf("Request: %s %s\n", req.Method(), req.FullUrl()))

	if headers := req.Headers(); len(headers) > 0 {
		b.WriteString("\nRequest Headers:\n")
		for key, value := range headers {
			b.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	if req.Data() != nil {
		b.WriteString(fmt.Sprintf("\nRequest Body:\n  %v\n", req.Data()))
	}

	if res != nil {
		b.WriteString(fmt.Sprintf("\nResponse Status: %d\n", res.StatusCode))

		if headers := res.Headers(); len(headers) > 0 {
			b.WriteString("\nResponse Headers:\n")
			for key, values := range headers {
				for _, value := range values {
					b.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
				}
			}
		}

		if len(res.Data) > 0 {
			bodyPreview := string(res.Data)
			if len(bodyPreview) > 500 {
				bodyPreview = bodyPreview[:500] + "... (truncated)"
			}
			b.WriteString(fmt.Sprintf("\nResponse Body:\n%s\n", bodyPreview))
		}
	}

	if trace != nil {
		b.WriteString(fmt.Sprintf("\n%s", trace.String()))
	}

	b.WriteString(fmt.Sprintf("\nCurl Equivalent:\n%s\n", req.ToCurl()))
	b.WriteString("==================\n")

	return b.String()
}
