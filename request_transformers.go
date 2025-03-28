package vecto

import (
	"encoding/json"
	"net/url"
)

func ApplicationJsonReqTransformer(req Request) (data []byte, err error) {
	if req.data == nil || req.data == "" {
		return nil, nil
	}

	return json.Marshal(req.data)
}

func FormEncodedReqTransformer(req Request) (data []byte, err error) {
	if req.data == nil || req.data == "" {
		return nil, nil
	}

	paramsMap := make(map[string]string)
	if req.data != nil {
		dataMap, ok := req.data.(map[string]string)
		if ok {
			paramsMap = dataMap
		}
	}

	formData := url.Values{}

	for key, value := range paramsMap {
		formData.Set(key, value)
	}

	return []byte(formData.Encode()), nil
}
