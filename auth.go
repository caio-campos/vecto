package vecto

import (
	"encoding/base64"
	"fmt"
)

// AuthType represents the type of authentication.
type AuthType int

const (
	// AuthTypeBasic represents HTTP Basic Authentication.
	AuthTypeBasic AuthType = iota
	// AuthTypeBearer represents Bearer Token Authentication.
	AuthTypeBearer
	// AuthTypeDigest represents HTTP Digest Authentication.
	AuthTypeDigest
)

// String returns the string representation of the AuthType.
func (a AuthType) String() string {
	switch a {
	case AuthTypeBasic:
		return "Basic"
	case AuthTypeBearer:
		return "Bearer"
	case AuthTypeDigest:
		return "Digest"
	default:
		return "Unknown"
	}
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Type     AuthType
	Username string
	Password string
	Token    string
}

// SetBasicAuth sets HTTP Basic Authentication for all requests made by this Vecto instance.
// It encodes the username and password in Base64 and adds the Authorization header.
func (v *Vecto) SetBasicAuth(username, password string) {
	if username == "" {
		return
	}

	encodedAuth := encodeBasicAuth(username, password)
	if v.config.Headers == nil {
		v.config.Headers = make(map[string]string, 1)
	}
	v.config.Headers["Authorization"] = encodedAuth
}

// SetBearerToken sets Bearer Token Authentication for all requests made by this Vecto instance.
// It adds the Authorization header with the Bearer token.
func (v *Vecto) SetBearerToken(token string) {
	if token == "" {
		return
	}

	if v.config.Headers == nil {
		v.config.Headers = make(map[string]string, 1)
	}
	v.config.Headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
}

// SetAuthToken is an alias for SetBearerToken.
// It sets Bearer Token Authentication for all requests made by this Vecto instance.
func (v *Vecto) SetAuthToken(token string) {
	v.SetBearerToken(token)
}

// ClearAuth removes all authentication headers from the Vecto instance.
func (v *Vecto) ClearAuth() {
	if v.config.Headers != nil {
		delete(v.config.Headers, "Authorization")
	}
}

// SetBasicAuth sets HTTP Basic Authentication for this specific request.
// It encodes the username and password in Base64 and adds the Authorization header.
func (r *Request) SetBasicAuth(username, password string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	encodedAuth := encodeBasicAuth(username, password)
	if err := r.SetHeader("Authorization", encodedAuth); err != nil {
		return fmt.Errorf("failed to set authorization header: %w", err)
	}
	return nil
}

// SetBearerToken sets Bearer Token Authentication for this specific request.
// It adds the Authorization header with the Bearer token.
func (r *Request) SetBearerToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if err := r.SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)); err != nil {
		return fmt.Errorf("failed to set authorization header: %w", err)
	}
	return nil
}

// SetAuthToken is an alias for SetBearerToken.
// It sets Bearer Token Authentication for this specific request.
func (r *Request) SetAuthToken(token string) error {
	return r.SetBearerToken(token)
}

// ClearAuth removes the Authorization header from this specific request.
func (r *Request) ClearAuth() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.headers != nil {
		delete(r.headers, "Authorization")
	}
}

// GetAuthHeader returns the current Authorization header value.
func (r *Request) GetAuthHeader() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.headers == nil {
		return ""
	}
	return r.headers["Authorization"]
}

// encodeBasicAuth encodes username and password for HTTP Basic Authentication.
func encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// DecodeBasicAuth decodes a Basic Authentication header.
// Returns the username and password, or an error if decoding fails.
func DecodeBasicAuth(authHeader string) (username, password string, err error) {
	const prefix = "Basic "
	if len(authHeader) < len(prefix) {
		return "", "", fmt.Errorf("invalid basic auth header")
	}

	if authHeader[:len(prefix)] != prefix {
		return "", "", fmt.Errorf("not a basic auth header")
	}

	encoded := authHeader[len(prefix):]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode basic auth: %w", err)
	}

	decodedStr := string(decoded)
	for i := 0; i < len(decodedStr); i++ {
		if decodedStr[i] == ':' {
			return decodedStr[:i], decodedStr[i+1:], nil
		}
	}

	return "", "", fmt.Errorf("invalid basic auth format")
}
