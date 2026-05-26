package typesense

import "context"

// CollectionAlias maps a virtual collection name to a real collection.
type CollectionAlias struct {
	Name           *string `json:"name,omitempty"`
	CollectionName string  `json:"collection_name"`
}

// AliasUpsertSchema is the body for PUT /aliases/{name}.
type AliasUpsertSchema struct {
	CollectionName string `json:"collection_name"`
}

// UpsertAlias creates or updates an alias.
func (c *Client) UpsertAlias(ctx context.Context, name string, body *AliasUpsertSchema) (*CollectionAlias, error) {
	out := &CollectionAlias{}
	if err := c.do(ctx, "PUT", "/aliases/"+name, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAlias retrieves an alias by name.
func (c *Client) GetAlias(ctx context.Context, name string) (*CollectionAlias, error) {
	out := &CollectionAlias{}
	if err := c.do(ctx, "GET", "/aliases/"+name, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteAlias removes an alias.
func (c *Client) DeleteAlias(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/aliases/"+name, nil, nil, nil)
}
