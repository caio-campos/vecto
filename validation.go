package vecto

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

const (
	maxURLLength     = 2048
	maxHeaderNameLen = 256
	maxHeaderValueLen = 8192
)

func validateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if len(urlStr) > maxURLLength {
		return fmt.Errorf("URL exceeds maximum length of %d characters", maxURLLength)
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got: %s", parsed.Scheme)
	}

	return nil
}

func validateHeaderName(name string) error {
	if name == "" {
		return fmt.Errorf("header name cannot be empty")
	}

	if len(name) > maxHeaderNameLen {
		return fmt.Errorf("header name exceeds maximum length of %d characters", maxHeaderNameLen)
	}

	for _, r := range name {
		if !isValidHeaderNameChar(r) {
			return fmt.Errorf("header name contains invalid character: %q", r)
		}
	}

	return nil
}

func validateHeaderValue(value string) error {
	if len(value) > maxHeaderValueLen {
		return fmt.Errorf("header value exceeds maximum length of %d characters", maxHeaderValueLen)
	}

	for _, r := range value {
		if r == '\r' || r == '\n' {
			return fmt.Errorf("header value cannot contain CR or LF characters")
		}
	}

	return nil
}

func isValidHeaderNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
}

func validateHeaders(headers map[string]string) error {
	if headers == nil {
		return nil
	}

	for name, value := range headers {
		if err := validateHeaderName(name); err != nil {
			return fmt.Errorf("invalid header name %q: %w", name, err)
		}

		if err := validateHeaderValue(value); err != nil {
			return fmt.Errorf("invalid header value for %q: %w", name, err)
		}
	}

	return nil
}

func sanitizeURL(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	return urlStr
}

