package vecto

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"
)

type httpClientFactory struct {
	config Config
}

func newDefaultClient(vecto *Vecto) (client Client, err error) {
	httpClientFact := newHttpClientFactory(vecto.config)

	httpClient, err := httpClientFact.make()
	if err != nil {
		return client, err
	}

	client = &DefaultClient{
		client: httpClient,
	}

	return client, nil
}

func newHttpClientFactory(config Config) httpClientFactory {
	return httpClientFactory{
		config,
	}
}

func (h *httpClientFactory) make() (client http.Client, err error) {
	transport, err := h.getTransportConfig()
	if err != nil {
		return client, err
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
			IdleConnTimeout: 50 * time.Second,
		}

		return transport, err
	}

	certificates := make([]tls.Certificate, len(h.config.Certificates))
	caCertPool := x509.NewCertPool()

	for _, certConfig := range h.config.Certificates {
		ok := caCertPool.AppendCertsFromPEM([]byte(certConfig.Cert))
		if !ok {
			return transport, err
		}

		cert, err := tls.X509KeyPair([]byte(certConfig.Cert), []byte(certConfig.Key))
		if err != nil {
			return transport, err
		}

		certificates = append(certificates, cert)
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       certificates,
		InsecureSkipVerify: true,
	}

	transport = &http.Transport{
		TLSClientConfig: tlsConfig,
		IdleConnTimeout: 50 * time.Second,
	}

	return transport, err
}
