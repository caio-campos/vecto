package vecto

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPClientFactory(t *testing.T) {
	t.Run("default transport", func(t *testing.T) {
		factory := newHTTPClientFactory(Config{
			Timeout: 10 * time.Second,
		})

		client, err := factory.make()
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, client.Timeout)
		assert.NotNil(t, client.Transport)
	})

	t.Run("custom transport", func(t *testing.T) {
		customTransport := &http.Transport{
			MaxIdleConns: 100,
		}

		factory := newHTTPClientFactory(Config{
			Timeout:       5 * time.Second,
			HTTPTransport: customTransport,
		})

		client, err := factory.make()
		assert.NoError(t, err)
		assert.Equal(t, customTransport, client.Transport)
	})

	t.Run("with certificates", func(t *testing.T) {
		factory := newHTTPClientFactory(Config{
			Timeout: 10 * time.Second,
			Certificates: []CertificateConfig{
				{Cert: "cert", Key: "key"},
			},
		})

		client, err := factory.make()
		assert.Error(t, err)
		assert.Equal(t, http.Client{}, client)
	})

	t.Run("with TLS config", func(t *testing.T) {
		factory := newHTTPClientFactory(Config{
			Timeout:            10 * time.Second,
			InsecureSkipVerify: true,
			Certificates: []CertificateConfig{
				{Cert: "cert", Key: "key"},
			},
		})

		client, err := factory.make()
		assert.Error(t, err)
		assert.Equal(t, http.Client{}, client)
	})
}

func TestGetTransportConfig(t *testing.T) {
	t.Run("no certificates", func(t *testing.T) {
		factory := newHTTPClientFactory(Config{})
		transport, err := factory.getTransportConfig()
		assert.NoError(t, err)
		assert.NotNil(t, transport)
		assert.Equal(t, defaultIdleConnTimeout, transport.IdleConnTimeout)
	})

	t.Run("invalid certificate", func(t *testing.T) {
		factory := newHTTPClientFactory(Config{
			Certificates: []CertificateConfig{
				{Cert: "invalid", Key: "invalid"},
			},
		})

		transport, err := factory.getTransportConfig()
		assert.Error(t, err)
		assert.Nil(t, transport)
	})

	t.Run("with insecure skip verify", func(t *testing.T) {
		factory := newHTTPClientFactory(Config{
			InsecureSkipVerify: true,
		})

		transport, err := factory.getTransportConfig()
		assert.NoError(t, err)
		assert.NotNil(t, transport)

		if transport.TLSClientConfig != nil {
			assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
		}
	})
}

func TestNewDefaultClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vecto := &Vecto{
			config: Config{
				Timeout:             10 * time.Second,
				MaxResponseBodySize: 1024 * 1024,
				EnableTrace:         false,
			},
		}

		client, err := newDefaultClient(vecto)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		defaultClient, ok := client.(*DefaultClient)
		assert.True(t, ok)
		assert.Equal(t, int64(1024*1024), defaultClient.maxResponseBodySize)
		assert.False(t, defaultClient.enableTrace)
	})

	t.Run("with trace enabled", func(t *testing.T) {
		vecto := &Vecto{
			config: Config{
				Timeout:     10 * time.Second,
				EnableTrace: true,
			},
		}

		client, err := newDefaultClient(vecto)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		defaultClient, ok := client.(*DefaultClient)
		assert.True(t, ok)
		assert.True(t, defaultClient.enableTrace)
	})
}
