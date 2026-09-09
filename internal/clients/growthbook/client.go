/*
Copyright 2026 The provider-growthbook Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package growthbook is a thin HTTP client for the GrowthBook REST API.
// Only the endpoints the provider manages are implemented; the request and
// response shapes mirror the GrowthBook OpenAPI spec.
package growthbook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultAPIURL is the GrowthBook Cloud API root. Self-hosted instances
	// use "https://<host>/api".
	DefaultAPIURL = "https://api.growthbook.io/api"

	defaultTimeout = 30 * time.Second
)

// Credentials is the JSON document stored in the ProviderConfig secret.
type Credentials struct {
	// APIKey is a GrowthBook secret key ("secret_..."). Required.
	APIKey string `json:"apiKey"`
	// APIURL is the API root including the "/api" suffix. Optional; defaults
	// to DefaultAPIURL.
	APIURL string `json:"apiUrl,omitempty"`
}

// ParseCredentials decodes the secret payload. A payload that is not JSON is
// treated as a bare API key so a plain-text secret also works.
func ParseCredentials(data []byte) (Credentials, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return Credentials{}, fmt.Errorf("credentials are empty")
	}
	var c Credentials
	if trimmed[0] == '{' {
		if err := json.Unmarshal(trimmed, &c); err != nil {
			return Credentials{}, fmt.Errorf("cannot decode credentials JSON: %w", err)
		}
	} else {
		c.APIKey = string(trimmed)
	}
	if c.APIKey == "" {
		return Credentials{}, fmt.Errorf("credentials do not contain an apiKey")
	}
	if c.APIURL == "" {
		c.APIURL = DefaultAPIURL
	}
	c.APIURL = strings.TrimRight(c.APIURL, "/")
	return c, nil
}

// APIError is returned for any non-2xx response.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("growthbook api: %d %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether err means the resource does not exist.
// GrowthBook is inconsistent here: some endpoints return 404, others return
// 400 with a "Could not find ..." message (observed on GET /v1/projects/{id}
// against a live instance). Both count.
func IsNotFound(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	if ae.StatusCode == http.StatusNotFound {
		return true
	}
	if ae.StatusCode != http.StatusBadRequest {
		return false
	}
	msg := strings.ToLower(ae.Message)
	return strings.Contains(msg, "could not find") || strings.Contains(msg, "not found")
}

// Client talks to one GrowthBook organization.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// Option customises a Client.
type Option func(*Client)

// WithHTTPClient overrides the underlying http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// New builds a Client from parsed credentials.
func New(creds Credentials, opts ...Option) (*Client, error) {
	if creds.APIKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}
	base := creds.APIURL
	if base == "" {
		base = DefaultAPIURL
	}
	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("invalid apiUrl %q: %w", base, err)
	}
	c := &Client{
		baseURL: strings.TrimRight(base, "/"),
		apiKey:  creds.APIKey,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// NewFromSecret parses the secret payload and builds a Client.
func NewFromSecret(data []byte, opts ...Option) (*Client, error) {
	creds, err := ParseCredentials(data)
	if err != nil {
		return nil, err
	}
	return New(creds, opts...)
}

// do sends one authenticated request and decodes a 2xx JSON body into out.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close() //nolint:errcheck // nothing useful to do with a close error on a read body

	return decodeResponse(resp, out)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("cannot encode request body: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return nil, fmt.Errorf("cannot build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func decodeResponse(resp *http.Response, out any) error {
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("cannot read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &APIError{StatusCode: resp.StatusCode, Message: errorMessage(payload)}
	}
	if out == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("cannot decode response body: %w", err)
	}
	return nil
}

// errorMessage extracts GrowthBook's {"message": "..."} error body, falling
// back to the raw payload.
func errorMessage(payload []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(payload, &e); err == nil && e.Message != "" {
		return e.Message
	}
	s := strings.TrimSpace(string(payload))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
