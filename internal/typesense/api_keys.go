package typesense

import (
	"context"
	"strconv"
)

// APIKey is a Typesense API key with permissions metadata.
type APIKey struct {
	Id          *int64   `json:"id,omitempty"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Collections []string `json:"collections"`
	ExpiresAt   *int64   `json:"expires_at,omitempty"`
	Value       *string  `json:"value,omitempty"`
	ValuePrefix *string  `json:"value_prefix,omitempty"`
}

// APIKeyCreateSchema is the body for POST /keys.
type APIKeyCreateSchema struct {
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Collections []string `json:"collections"`
	ExpiresAt   *int64   `json:"expires_at,omitempty"`
	Value       *string  `json:"value,omitempty"`
}

// CreateAPIKey provisions a new API key. The full key value is returned only on creation.
func (c *Client) CreateAPIKey(ctx context.Context, body *APIKeyCreateSchema) (*APIKey, error) {
	out := &APIKey{}
	if err := c.do(ctx, "POST", "/keys", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAPIKey retrieves API key metadata (the value itself is never returned on read).
func (c *Client) GetAPIKey(ctx context.Context, id int64) (*APIKey, error) {
	out := &APIKey{}
	if err := c.do(ctx, "GET", "/keys/"+strconv.FormatInt(id, 10), nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteAPIKey revokes an API key by its numeric id.
func (c *Client) DeleteAPIKey(ctx context.Context, id int64) error {
	return c.do(ctx, "DELETE", "/keys/"+strconv.FormatInt(id, 10), nil, nil, nil)
}
