package vecto

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultIdleConnTimeout = 50 * time.Second
)

type httpClientFactory struct {
	config Config
}

func newDefaultClient(vecto *Vecto) (client Client, err error) {
	httpClientFact := newHTTPClientFactory(vecto.config)

	httpClient, err := httpClientFact.make()
	if err != nil {
		return client, err
	}

	client = &DefaultClient{
		client:              httpClient,
		maxResponseBodySize: vecto.config.MaxResponseBodySize,
		enableTrace:         vecto.config.EnableTrace,
	}

	return client, nil
}

func newHTTPClientFactory(config Config) httpClientFactory {
	return httpClientFactory{
		config,
	}
}

func (h *httpClientFactory) make() (client http.Client, err error) {
	transport := h.config.HTTPTransport
	if transport == nil {
		transport, err = h.getTransportConfig()
		if err != nil {
			return client, err
		}
	}

	client = http.Client{
		Transport: transport,
		Timeout:   h.config.Timeout,
	}

	return client, nil
}

func (h *httpClientFactory) getTransportConfig() (transport *http.Transport, err error) {
	if len(h.config.Certificates) == 0 {
		transport = &http.Transport{
			IdleConnTimeout: defaultIdleConnTimeout,
		}

		return transport, err
	}

	certificates := make([]tls.Certificate, 0, len(h.config.Certificates))
	caCertPool := x509.NewCertPool()

	for _, certConfig := range h.config.Certificates {
		ok := caCertPool.AppendCertsFromPEM([]byte(certConfig.Cert))
		if !ok {
			return transport, fmt.Errorf("failed to append certificate to pool")
		}

		cert, err := tls.X509KeyPair([]byte(certConfig.Cert), []byte(certConfig.Key))
		if err != nil {
			return transport, fmt.Errorf("failed to load X509 key pair: %w", err)
		}

		certificates = append(certificates, cert)
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       certificates,
		InsecureSkipVerify: h.config.InsecureSkipVerify,
	}

	transport = &http.Transport{
		TLSClientConfig: tlsConfig,
		IdleConnTimeout: defaultIdleConnTimeout,
	}

	return transport, err
}
