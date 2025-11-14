package vecto

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHttpClientFactoryCustomTransport(t *testing.T) {
	customTransport := &http.Transport{}

	config := Config{
		Timeout:       5 * time.Second,
		HTTPTransport: customTransport,
	}

	factory := newHTTPClientFactory(config)
	client, err := factory.make()

	assert.NoError(t, err)
	assert.Equal(t, customTransport, client.Transport)
	assert.Equal(t, config.Timeout, client.Timeout)
}

func TestHttpClientFactoryInvalidCertificate(t *testing.T) {
	config := Config{
		Timeout: 5 * time.Second,
		Certificates: []CertificateConfig{
			{
				Cert: "invalid-cert",
				Key:  "invalid-key",
			},
		},
	}

	factory := newHTTPClientFactory(config)
	_, err := factory.make()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to")
}

