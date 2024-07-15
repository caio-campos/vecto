package vecto

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"dario.cat/mergo"
)

type Vecto struct {
	config       Config
	client       Client
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
		return nil, err
	}

	if v.config.Adapter != nil {
		return v.config.Adapter(request)
	}

	request, err = v.interceptRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	res, err = v.client.Do(ctx, request)
	if err != nil {
		return nil, err
	}

	res.success = v.config.ValidateStatus(res)
	resultRes, err := v.interceptResponse(ctx, res)
	if err != nil {
		return nil, err
	}

	v.dispatchRequestCompleted(resultRes)

	return resultRes, nil
}

func (v *Vecto) dispatchRequestCompleted(res *Response) {
	event := RequestCompletedEvent{
		response: res,
	}

	for _, cb := range res.request.events.completed {
		go cb(event)
	}
}

func (v *Vecto) interceptRequest(ctx context.Context, req Request) (resultReq Request, err error) {
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

func (v *Vecto) newRequest(urlStr string, method string, options *RequestOptions) (req Request, err error) {
	reqOptions := RequestOptions{}
	if options != nil {
		reqOptions = *options
	}

	headersCopy := make(map[string]string, len(v.config.Headers))
	for key, value := range v.config.Headers {
		headersCopy[key] = value
	}
	for key, value := range reqOptions.Headers {
		headersCopy[key] = value
	}

	fullUrl, err := v.getUrlInstance(urlStr, reqOptions.Params)
	if err != nil {
		return Request{}, err
	}

	requestTransform := ApplicationJsonReqTransformer
	if options != nil && options.RequestTransform != nil {
		requestTransform = options.RequestTransform
	}

	emptyReq, _ := http.NewRequest(method, fullUrl.String(), bytes.NewReader([]byte{}))
	baseUrl := fullUrl.Scheme + "://" + fullUrl.Host + fullUrl.Path

	req = Request{
		FullUrl:          fullUrl.String(),
		BaseUrl:          baseUrl,
		Method:           method,
		Params:           reqOptions.Params,
		Headers:          headersCopy,
		Data:             reqOptions.Data,
		requestTransform: requestTransform,
		rawRequest:       emptyReq,
		Host:             fullUrl.Host,
		Scheme:           fullUrl.Scheme,
		Path:             fullUrl.Path,
		events: requestEvents{
			completed: make([]RequestCompletedCallback, 0),
		},
	}

	return req, nil
}

func (v *Vecto) getUrlInstance(reqUrl string, params map[string]any) (finalURL *url.URL, err error) {
	url, err := url.Parse(v.config.BaseURL + reqUrl)
	if err != nil {
		return finalURL, err
	}

	urlParams := url.Query()
	for key, value := range params {
		urlParams.Add(key, fmt.Sprintf("%v", value))
	}

	url.RawQuery = urlParams.Encode()
	return url, nil
}

func (v *Vecto) setHttpClient() (err error) {
	client, err := newDefaultClient(v)
	if err != nil {
		return err
	}

	v.client = client

	return nil
}
