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

	factory := newHttpClientFactory(config)
	client, err := factory.make()

	assert.NoError(t, err)
	assert.Equal(t, customTransport, client.Transport)
	assert.Equal(t, config.Timeout, client.Timeout)
}

