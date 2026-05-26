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
// PUT /aliases/{name} — https://typesense.org/docs/30.2/api/collection-alias.html#create-or-update-an-alias
func (c *Client) UpsertAlias(ctx context.Context, name string, body *AliasUpsertSchema) (*CollectionAlias, error) {
	out := &CollectionAlias{}
	if err := c.do(ctx, "PUT", "/aliases/"+name, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAlias retrieves an alias by name.
// GET /aliases/{name} — https://typesense.org/docs/30.2/api/collection-alias.html#retrieve-an-alias
func (c *Client) GetAlias(ctx context.Context, name string) (*CollectionAlias, error) {
	out := &CollectionAlias{}
	if err := c.do(ctx, "GET", "/aliases/"+name, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteAlias removes an alias.
// DELETE /aliases/{name} — https://typesense.org/docs/30.2/api/collection-alias.html#delete-an-alias
func (c *Client) DeleteAlias(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/aliases/"+name, nil, nil, nil)
}
