package typesense

import (
	"context"
	"net/url"
)

// DocumentParams are query-string parameters accepted by document index/update endpoints.
type DocumentParams struct {
	Action      *string
	DirtyValues *string
}

func (p *DocumentParams) values() url.Values {
	if p == nil {
		return nil
	}
	v := url.Values{}
	if p.Action != nil {
		v.Set("action", *p.Action)
	}
	if p.DirtyValues != nil {
		v.Set("dirty_values", *p.DirtyValues)
	}
	if len(v) == 0 {
		return nil
	}
	return v
}

// IndexDocument inserts a document into a collection.
func (c *Client) IndexDocument(ctx context.Context, collection string, doc map[string]any, params *DocumentParams) (map[string]any, error) {
	out := map[string]any{}
	if err := c.do(ctx, "POST", "/collections/"+collection+"/documents", params.values(), doc, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDocument retrieves a document by id.
func (c *Client) GetDocument(ctx context.Context, collection, id string) (map[string]any, error) {
	out := map[string]any{}
	if err := c.do(ctx, "GET", "/collections/"+collection+"/documents/"+id, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateDocument patches a document by id.
func (c *Client) UpdateDocument(ctx context.Context, collection, id string, doc map[string]any, params *DocumentParams) (map[string]any, error) {
	out := map[string]any{}
	if err := c.do(ctx, "PATCH", "/collections/"+collection+"/documents/"+id, params.values(), doc, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteDocument removes a document by id.
func (c *Client) DeleteDocument(ctx context.Context, collection, id string) error {
	return c.do(ctx, "DELETE", "/collections/"+collection+"/documents/"+id, nil, nil, nil)
}
