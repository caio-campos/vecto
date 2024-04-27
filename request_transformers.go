package vecto

import (
	"encoding/json"
	"net/url"
)

func ApplicationJsonReqTransformer(req Request) (data []byte, err error) {
	return json.Marshal(req.Data)
}

func FormEncodedReqTransformer(req Request) (data []byte, err error) {
	paramsMap := make(map[string]string)
	if req.Data != nil {
		dataMap, ok := req.Data.(map[string]string)
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
