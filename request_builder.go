package vecto

type requestBuilder struct {
	request *Request
}

func newRequestBuilder(basePath, method string) *requestBuilder {
	return &requestBuilder{
		request: &Request{
			basePath: basePath,
			method:   method,
			headers:  make(map[string]string),
			params:   make(map[string]any),
		},
	}
}

func (b *requestBuilder) SetHeader(key, value string) *requestBuilder {
	b.request.headers[key] = value
	return b
}

func (b *requestBuilder) SetHeaders(headers map[string]string) *requestBuilder {
	for key, value := range headers {
		b.request.headers[key] = value
	}
	return b
}

func (b *requestBuilder) SetParam(key string, value any) *requestBuilder {
	b.request.params[key] = value
	return b
}

func (b *requestBuilder) SetData(data interface{}) *requestBuilder {
	b.request.data = data
	return b
}

func (b *requestBuilder) SetTransform(transform RequestTransformFunc) *requestBuilder {
	b.request.requestTransform = transform
	return b
}

func (b *requestBuilder) Build() (*Request, error) {
	err := b.request.refreshUrl()
	if err != nil {
		return nil, err
	}

	return b.request, nil
}
