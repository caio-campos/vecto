package vecto

import (
	"context"
	"fmt"
	"net/http"
	"time"
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

	if provided.MaxResponseBodySize > 0 {
		result.MaxResponseBodySize = provided.MaxResponseBodySize
	}

	if provided.MaxConcurrentCallbacks > 0 {
		result.MaxConcurrentCallbacks = provided.MaxConcurrentCallbacks
	}

	if provided.CallbackTimeout > 0 {
		result.CallbackTimeout = provided.CallbackTimeout
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
	// Validação de context nil
	if ctx == nil {
		ctx = context.Background()
	}

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

	// Dispatch com context original preservado
	v.dispatchRequestCompleted(ctx, resultRes)

	return resultRes, nil
}

// dispatchRequestCompleted executa callbacks em goroutines com timeout e rate limiting
func (v *Vecto) dispatchRequestCompleted(ctx context.Context, res *Response) {
	// Deep copy da Response para evitar race conditions
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
	// Usa getAll() thread-safe em vez de acessar slice diretamente
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
	// Usa getAll() thread-safe em vez de acessar slice diretamente
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

	fullUrlStr := fmt.Sprintf("%s%s", v.config.BaseURL, urlStr)

	transform := ApplicationJsonReqTransformer
	if v.config.RequestTransform != nil {
		transform = v.config.RequestTransform
	}
	if reqOptions.RequestTransform != nil {
		transform = reqOptions.RequestTransform
	}

	builder := newRequestBuilder(fullUrlStr, method).
		SetHeaders(v.config.Headers).
		SetHeaders(reqOptions.Headers).
		SetData(reqOptions.Data).
		SetTransform(transform)

	for key, value := range reqOptions.Params {
		builder.SetParam(key, value)
	}

	req, err := builder.Build()
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (v *Vecto) setHTTPClient() (err error) {
	client, err := newDefaultClient(v)
	if err != nil {
		return err
	}

	v.client = client

	return nil
}
