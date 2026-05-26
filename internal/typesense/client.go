// Package typesense is a small HTTP client for the Typesense REST API,
// scoped to the operations this Terraform provider needs. It replaces
// dependency on github.com/typesense/typesense-go so the provider can
// expose server-supported fields without waiting on SDK regenerations.
package typesense

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

// defaultTimeout caps a single HTTP request. Most Typesense calls return in
// milliseconds, but collection-create with an `embed` field downloads/loads an
// embedding model before responding — that can take several minutes on a cold
// server. Default chosen to cover the slowest realistic case.
const defaultTimeout = 5 * time.Minute

// Client is a thin wrapper around net/http for talking to a Typesense server.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client (mainly for tests).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// NewClient builds a Client for the given Typesense server and API key.
func NewClient(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// APIError is returned for any non-2xx response.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("typesense API error: HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("typesense API error: HTTP %d: %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether err is an APIError with a 404 status code.
func IsNotFound(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == http.StatusNotFound
}

// do performs an HTTP request and decodes the JSON response into out (if non-nil).
//
// body may be nil, json.RawMessage / []byte (passed through verbatim), or any value (JSON-marshaled).
// out may be nil (response discarded), *json.RawMessage (raw bytes captured), or any pointer (JSON-unmarshaled).
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	return c.doWithContentType(ctx, method, path, query, "application/json", body, out)
}

func (c *Client) doWithContentType(ctx context.Context, method, path string, query url.Values, contentType string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		switch b := body.(type) {
		case json.RawMessage:
			reqBody = bytes.NewReader(b)
		case []byte:
			reqBody = bytes.NewReader(b)
		case io.Reader:
			reqBody = b
		default:
			buf, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(buf)
		}
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	if reqBody != nil {
		req.Header.Set("Content-Type", contentType)
	}
	if out != nil {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(data))
		var errResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Message != "" {
			msg = errResp.Message
		}
		if resp.StatusCode == http.StatusNotFound && !strings.Contains(msg, "Not Found") {
			if msg == "" {
				msg = "Not Found"
			} else {
				msg = "Not Found: " + msg
			}
		}
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	if out == nil || len(data) == 0 {
		return nil
	}
	if raw, ok := out.(*json.RawMessage); ok {
		*raw = append((*raw)[:0], data...)
		return nil
	}
	return json.Unmarshal(data, out)
}
