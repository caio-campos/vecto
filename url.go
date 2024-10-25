package vecto

import (
	"fmt"
	"net/url"
)

func getUrlInstance(reqUrl string, params map[string]any) (finalURL *url.URL, err error) {
	url, err := url.Parse(reqUrl)
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
