package vecto

import (
	"context"
	"fmt"
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
	config            Config
	client            Client
	logger            Logger
	Interceptors      interceptorCollectionWrapper
	circuitBreakerMgr *CircuitBreakerManager
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

	return &instance, nil
}

func validateConfig(config Config) error {
	if config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	if config.MaxResponseBodySize < 0 {
		return fmt.Errorf("max response body size cannot be negative")
	}

	if config.MaxConcurrentCallbacks < 0 {
		return fmt.Errorf("max concurrent callbacks cannot be negative")
	}

	if config.CallbackTimeout < 0 {
		return fmt.Errorf("callback timeout cannot be negative")
	}

	if config.BaseURL != "" {
		if _, err := http.NewRequest("GET", config.BaseURL, nil); err != nil {
			return fmt.Errorf("invalid base URL: %w", err)
		}
	}

	return nil
}

func mergeConfig(provided, defaults Config) Config {
	result := Config{
		BaseURL:                defaults.BaseURL,
		Timeout:                defaults.Timeout,
		Headers:                cloneHeaders(defaults.Headers),
		Certificates:           cloneCertificates(defaults.Certificates),
		HTTPTransport:          defaults.HTTPTransport,
		Adapter:                defaults.Adapter,
		RequestTransform:       defaults.RequestTransform,
		ValidateStatus:         defaults.ValidateStatus,
		InsecureSkipVerify:     defaults.InsecureSkipVerify,
		Logger:                 defaults.Logger,
		MetricsCollector:       defaults.MetricsCollector,
		MaxResponseBodySize:    defaults.MaxResponseBodySize,
		MaxConcurrentCallbacks: defaults.MaxConcurrentCallbacks,
		CallbackTimeout:        defaults.CallbackTimeout,
	}

	if provided.BaseURL != "" {
		result.BaseURL = provided.BaseURL
	}

	if provided.Timeout != 0 {
		result.Timeout = provided.Timeout
	}

	if len(provided.Headers) > 0 {
		if result.Headers == nil {
			result.Headers = make(map[string]string, len(provided.Headers))
		}
		for k, v := range provided.Headers {
			result.Headers[k] = v
		}
	}

	if len(provided.Certificates) > 0 {
		result.Certificates = cloneCertificates(provided.Certificates)
	}

	if provided.HTTPTransport != nil {
		result.HTTPTransport = provided.HTTPTransport
	}

	if provided.Adapter != nil {
		result.Adapter = provided.Adapter
	}

	if provided.RequestTransform != nil {
		result.RequestTransform = provided.RequestTransform
	}

	if provided.ValidateStatus != nil {
		result.ValidateStatus = provided.ValidateStatus
	}

	result.InsecureSkipVerify = provided.InsecureSkipVerify

	if provided.Logger != nil {
		result.Logger = provided.Logger
	}

	if provided.MetricsCollector != nil {
		result.MetricsCollector = provided.MetricsCollector
	}

	if provided.MaxResponseBodySize > 0 {
		result.MaxResponseBodySize = provided.MaxResponseBodySize
	}

	if provided.MaxConcurrentCallbacks > 0 {
		result.MaxConcurrentCallbacks = provided.MaxConcurrentCallbacks
	}

	if provided.CallbackTimeout > 0 {
		result.CallbackTimeout = provided.CallbackTimeout
	}

	if provided.CircuitBreaker != nil {
		result.CircuitBreaker = provided.CircuitBreaker
	}

	if provided.Retry != nil {
		result.Retry = provided.Retry
	}

	return result
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string]string, len(headers))
	for k, v := range headers {
		result[k] = v
	}

	return result
}

func cloneCertificates(certificates []CertificateConfig) []CertificateConfig {
	if len(certificates) == 0 {
		return nil
	}

	result := make([]CertificateConfig, len(certificates))
	copy(result, certificates)

	return result
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
			v.logger.Error(ctx, "request interceptor failed", map[string]interface{}{
				"url":   request.FullUrl(),
				"error": err.Error(),
			})
		}
		v.recordMetrics(ctx, request, nil, time.Since(startTime), err)
		return nil, fmt.Errorf("request interceptor failed: %w", err)
	}

	var cbKey string
	var breaker *CircuitBreaker

	retryConfig := v.getRetryConfig(options)

	if v.circuitBreakerMgr != nil {
		request.mu.RLock()
		if request.cbKeyCached {
			cbKey = request.cbKey
			request.mu.RUnlock()
		} else {
			request.mu.RUnlock()
			request.mu.Lock()
			if !request.cbKeyCached {
				request.cbKey = v.getCircuitBreakerKey(request)
				request.cbKeyCached = true
			}
			cbKey = request.cbKey
			request.mu.Unlock()
		}

		breaker = v.circuitBreakerMgr.GetOrCreate(cbKey, nil)

		res, err = breaker.Execute(ctx, func() (*Response, error) {
			if retryConfig != nil && shouldUseRetry(breaker) {
				return v.executeWithRetry(ctx, request, retryConfig)
			}
			return v.client.Do(ctx, request)
		})

		if err != nil {
			duration := time.Since(startTime)
			if _, isCbError := err.(*CircuitBreakerError); isCbError {
				if !v.logger.IsNoop() {
					v.logger.Warn(ctx, "request blocked by circuit breaker", map[string]interface{}{
						"url":    request.FullUrl(),
						"method": request.Method(),
						"key":    cbKey,
						"state":  breaker.GetState().String(),
					})
				}
				v.recordMetrics(ctx, request, nil, duration, err)
				return nil, err
			}

			if !v.logger.IsNoop() {
				v.logger.Error(ctx, "http request failed", map[string]interface{}{
					"url":    request.FullUrl(),
					"method": request.Method(),
					"error":  err.Error(),
				})
			}
			v.recordMetrics(ctx, request, nil, duration, err)
			return nil, fmt.Errorf("http request failed: %w", err)
		}
	} else {
		if retryConfig != nil {
			res, err = v.executeWithRetry(ctx, request, retryConfig)
		} else {
			res, err = v.client.Do(ctx, request)
		}

		if err != nil {
			duration := time.Since(startTime)
			if !v.logger.IsNoop() {
				v.logger.Error(ctx, "http request failed", map[string]interface{}{
					"url":    request.FullUrl(),
					"method": request.Method(),
					"error":  err.Error(),
				})
			}
			v.recordMetrics(ctx, request, nil, duration, err)
			return nil, fmt.Errorf("http request failed: %w", err)
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
			v.logger.Error(ctx, "response interceptor failed", map[string]interface{}{
				"url":         request.FullUrl(),
				"status_code": res.StatusCode,
				"error":       err.Error(),
			})
		}
		v.recordMetrics(ctx, request, res, duration, err)
		return nil, fmt.Errorf("response interceptor failed: %w", err)
	}

	if v.config.DebugMode {
		v.writeDebugOutput(request, resultRes)
	}

	v.dispatchRequestCompleted(ctx, resultRes)

	v.recordMetrics(ctx, request, resultRes, duration, nil)

	return resultRes, nil
}

func (v *Vecto) dispatchRequestCompleted(ctx context.Context, res *Response) {
	responseCopy := res.deepCopy()

	event := RequestCompletedEvent{
		response: responseCopy,
	}

	requestUrl := responseCopy.request.FullUrl()

	res.request.mu.RLock()
	callbacks := make([]RequestCompletedCallback, len(res.request.events.completed))
	copy(callbacks, res.request.events.completed)
	res.request.mu.RUnlock()

	maxConcurrent := v.config.MaxConcurrentCallbacks
	if maxConcurrent <= 0 {
		maxConcurrent = 100
	}

	callbackTimeout := v.config.CallbackTimeout
	if callbackTimeout <= 0 {
		callbackTimeout = 30 * time.Second
	}

	sem := make(chan struct{}, maxConcurrent)

	for _, cb := range callbacks {
		select {
		case <-ctx.Done():
			v.logger.Warn(ctx, "context cancelled, skipping remaining callbacks", map[string]interface{}{
				"url": requestUrl,
			})
			return
		case sem <- struct{}{}:
		}

		go func(callback RequestCompletedCallback, url string) {
			defer func() {
				<-sem
				if r := recover(); r != nil {
					v.logger.Error(ctx, "panic in request completed callback", map[string]interface{}{
						"panic": fmt.Sprintf("%v", r),
						"url":   url,
					})
				}
			}()

			callbackCtx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()

			done := make(chan struct{})
			go func() {
				defer close(done)
				select {
				case <-callbackCtx.Done():
					return
				default:
					callback(event)
				}
			}()

			select {
			case <-done:
			case <-callbackCtx.Done():
				v.logger.Error(ctx, "callback timeout exceeded", map[string]interface{}{
					"url":     url,
					"timeout": callbackTimeout.String(),
				})
			}
		}(cb, requestUrl)
	}
}

func (v *Vecto) interceptRequest(ctx context.Context, req *Request) (resultReq *Request, err error) {
	resultReq = req
	for _, interceptor := range v.Interceptors.Request.getAll() {
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
	for _, interceptor := range v.Interceptors.Response.getAll() {
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

	fullUrlStr := v.config.BaseURL + urlStr

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

	for k, v := range v.config.Headers {
		headers[k] = v
	}
	for k, v := range reqOptions.Headers {
		headers[k] = v
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

func (v *Vecto) recordMetrics(ctx context.Context, req *Request, res *Response, duration time.Duration, err error) {
	if v.config.MetricsCollector == nil {
		return
	}

	var normalizedURL, fullURL string
	var method string
	var requestSize int64
	var statusCode int
	var responseSize int64
	var success bool

	if req != nil {
		method = req.Method()
		fullURL = req.FullUrl()
		normalizedURL = v.normalizeURL(req)

		if req.RawRequest() != nil && req.RawRequest().Body != nil {
			if req.RawRequest().ContentLength > 0 {
				requestSize = req.RawRequest().ContentLength
			}
		}
	}

	if res != nil {
		statusCode = res.StatusCode
		responseSize = int64(len(res.Data))
		success = res.success
	}

	metrics := RequestMetrics{
		Method:       method,
		URL:          normalizedURL,
		FullURL:      fullURL,
		Duration:     duration,
		StatusCode:   statusCode,
		Error:        err,
		RequestSize:  requestSize,
		ResponseSize: responseSize,
		Success:      success,
	}

	v.config.MetricsCollector.RecordRequest(ctx, metrics)
}

func (v *Vecto) recordMetricsWithFallback(ctx context.Context, method, url string, req *Request, res *Response, duration time.Duration, err error) {
	if v.config.MetricsCollector == nil {
		return
	}

	var normalizedURL, fullURL string
	var requestSize int64
	var statusCode int
	var responseSize int64
	var success bool

	if req != nil {
		fullURL = req.FullUrl()
		normalizedURL = v.normalizeURL(req)
		if req.RawRequest() != nil && req.RawRequest().Body != nil {
			if req.RawRequest().ContentLength > 0 {
				requestSize = req.RawRequest().ContentLength
			}
		}
	} else {
		fullURL = url
		normalizedURL = url
	}

	if res != nil {
		statusCode = res.StatusCode
		responseSize = int64(len(res.Data))
		success = res.success
	}

	metrics := RequestMetrics{
		Method:       method,
		URL:          normalizedURL,
		FullURL:      fullURL,
		Duration:     duration,
		StatusCode:   statusCode,
		Error:        err,
		RequestSize:  requestSize,
		ResponseSize: responseSize,
		Success:      success,
	}

	v.config.MetricsCollector.RecordRequest(ctx, metrics)
}

func (v *Vecto) normalizeURL(req *Request) string {
	if req == nil {
		return ""
	}

	scheme := req.Scheme()
	host := req.Host()
	path := req.Path()

	if scheme == "" && host == "" {
		return path
	}

	if host == "" {
		return path
	}

	return scheme + "://" + host + path
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
