package vecto

import (
	"fmt"
)

func validateConfig(config Config) error {
	if config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	if config.MaxResponseBodySize < 0 {
		return fmt.Errorf("max response body size cannot be negative")
	}

	if config.MaxResponseBodySize > 0 && config.MaxResponseBodySize > 1024*1024*1024 {
		return fmt.Errorf("max response body size cannot exceed 1GB")
	}

	if config.BaseURL != "" {
		if err := validateURL(config.BaseURL); err != nil {
			return fmt.Errorf("invalid base URL: %w", err)
		}
	}

	if err := validateHeaders(config.Headers); err != nil {
		return fmt.Errorf("invalid headers: %w", err)
	}

	return nil
}

func mergeConfig(provided, defaults Config) Config {
	result := Config{
		BaseURL:                defaults.BaseURL,
		Timeout:                defaults.Timeout,
		Headers:                cloneHeaders(defaults.Headers),
		Certificates:           cloneCertificates(defaults.Certificates),
		HTTPTransport:          defaults.HTTPTransport,
		Adapter:                defaults.Adapter,
		RequestTransform:       defaults.RequestTransform,
		ValidateStatus:         defaults.ValidateStatus,
		InsecureSkipVerify:     defaults.InsecureSkipVerify,
		Logger:              defaults.Logger,
		MetricsCollector:    defaults.MetricsCollector,
		MaxResponseBodySize: defaults.MaxResponseBodySize,
	}

	if provided.BaseURL != "" {
		result.BaseURL = provided.BaseURL
	}

	if provided.Timeout != 0 {
		result.Timeout = provided.Timeout
	}

	if len(provided.Headers) > 0 {
		if result.Headers == nil {
			result.Headers = make(map[string]string, len(provided.Headers))
		}
		for k, v := range provided.Headers {
			result.Headers[k] = v
		}
	}

	if len(provided.Certificates) > 0 {
		result.Certificates = cloneCertificates(provided.Certificates)
	}

	if provided.HTTPTransport != nil {
		result.HTTPTransport = provided.HTTPTransport
	}

	if provided.Adapter != nil {
		result.Adapter = provided.Adapter
	}

	if provided.RequestTransform != nil {
		result.RequestTransform = provided.RequestTransform
	}

	if provided.ValidateStatus != nil {
		result.ValidateStatus = provided.ValidateStatus
	}

	result.InsecureSkipVerify = provided.InsecureSkipVerify

	if provided.Logger != nil {
		result.Logger = provided.Logger
	}

	if provided.MetricsCollector != nil {
		result.MetricsCollector = provided.MetricsCollector
	}

	if provided.MaxResponseBodySize > 0 {
		result.MaxResponseBodySize = provided.MaxResponseBodySize
	}

	if provided.CircuitBreaker != nil {
		result.CircuitBreaker = provided.CircuitBreaker
	}

	if provided.Retry != nil {
		result.Retry = provided.Retry
	}

	return result
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string]string, len(headers))
	for k, v := range headers {
		result[k] = v
	}

	return result
}

func cloneCertificates(certificates []CertificateConfig) []CertificateConfig {
	if len(certificates) == 0 {
		return nil
	}

	result := make([]CertificateConfig, len(certificates))
	copy(result, certificates)

	return result
}

