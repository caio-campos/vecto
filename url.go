package vecto

import (
	"fmt"
	"net/url"
)

func getUrlInstance(reqUrl string, params map[string]any) (finalURL *url.URL, err error) {
	parsedURL, err := url.Parse(reqUrl)
	if err != nil {
		return finalURL, err
	}

	urlParams := parsedURL.Query()
	for key, value := range params {
		urlParams.Add(key, fmt.Sprintf("%v", value))
	}

	parsedURL.RawQuery = urlParams.Encode()
	return parsedURL, nil
}
