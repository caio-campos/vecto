package vecto

type requestBuilder struct {
	request *Request
	err     error
}

func newRequestBuilder(basePath, method string) *requestBuilder {
	return &requestBuilder{
		request: &Request{
			baseURL: basePath,
			method:  method,
			headers: make(map[string]string),
			params:  make(map[string]any),
		},
	}
}

func (b *requestBuilder) SetHeader(key, value string) *requestBuilder {
	if b.request.headers == nil {
		b.request.headers = make(map[string]string)
	}
	b.request.headers[key] = value
	return b
}

func (b *requestBuilder) SetHeaders(headers map[string]string) *requestBuilder {
	for key, value := range headers {
		b.SetHeader(key, value)
	}
	return b
}

func (b *requestBuilder) SetParam(key string, value any) *requestBuilder {
	if b.err != nil {
		return b
	}

	if err := b.request.SetParam(key, value); err != nil {
		b.err = err
	}
	return b
}

func (b *requestBuilder) SetData(data interface{}) *requestBuilder {
	b.request.data = data
	return b
}

func (b *requestBuilder) SetTransform(transform RequestTransformFunc) *requestBuilder {
	b.request.transform = transform
	return b
}

func (b *requestBuilder) Build() (*Request, error) {
	if b.err != nil {
		return nil, b.err
	}

	err := b.request.refreshUrl()
	if err != nil {
		return nil, err
	}

	return b.request, nil
}
