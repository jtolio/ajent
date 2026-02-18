// Package providers provides LLM provider adapters.
package providers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/services/vllm"
)

// NewOpenAICompat creates an OpenAI-compatible provider that supports custom
// base URLs and API key authentication. It reuses the bellman vLLM provider
// (which already implements the OpenAI protocol with configurable URLs) and
// adds Bearer token auth via an HTTP transport wrapper.
//
// baseURL should not include /v1 (e.g. "https://api.fireworks.ai/inference").
// If empty, defaults to "https://api.openai.com".
func NewOpenAICompat(apiKey, baseURL, model string) (gen.Gen, error) {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	wrapped := http.DefaultTransport
	if http.DefaultClient.Transport != nil {
		wrapped = http.DefaultClient.Transport
	}
	http.DefaultClient.Transport = &authTransport{
		apiKey:  apiKey,
		host:    u.Host,
		wrapped: wrapped,
	}

	return vllm.New([]string{baseURL}, []string{model}), nil
}

// authTransport is an http.RoundTripper that adds an Authorization Bearer
// header to requests matching a specific host.
type authTransport struct {
	apiKey  string
	host    string
	wrapped http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == t.host && req.URL.Scheme == "https" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	return t.wrapped.RoundTrip(req)
}
