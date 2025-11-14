package vecto

import (
	"encoding/json"
	"net/url"
)

func ApplicationJsonReqTransformer(req *Request) (data []byte, err error) {
	reqData := req.Data()
	if reqData == nil || reqData == "" {
		return nil, nil
	}

	return json.Marshal(reqData)
}

func FormEncodedReqTransformer(req *Request) (data []byte, err error) {
	reqData := req.Data()
	if reqData == nil || reqData == "" {
		return nil, nil
	}

	paramsMap := make(map[string]string)
	dataMap, ok := reqData.(map[string]string)
	if ok {
		paramsMap = dataMap
	}

	formData := url.Values{}

	for key, value := range paramsMap {
		formData.Set(key, value)
	}

	return []byte(formData.Encode()), nil
}
