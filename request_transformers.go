package vecto

import (
	"encoding/json"
)

func ApplicationJsonReqTransformer(req *Request) (data []byte, err error) {
	reqData := req.Data()
	if reqData == nil || reqData == "" {
		return nil, nil
	}

	return json.Marshal(reqData)
}
