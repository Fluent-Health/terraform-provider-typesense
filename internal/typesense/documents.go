package typesense

import (
	"context"
	"net/url"
)

// IndexAction selects the document-write semantics for IndexDocument /
// UpdateDocument. See https://typesense.org/docs/30.2/api/documents.html#index-a-document.
type IndexAction string

const (
	IndexActionCreate  IndexAction = "create"
	IndexActionUpsert  IndexAction = "upsert"
	IndexActionUpdate  IndexAction = "update"
	IndexActionEmplace IndexAction = "emplace"
)

// DirtyValues tells the server what to do when a document value doesn't match
// the declared field type. See https://typesense.org/docs/30.2/api/documents.html#dealing-with-dirty-data.
type DirtyValues string

const (
	DirtyValuesCoerceOrReject DirtyValues = "coerce_or_reject"
	DirtyValuesCoerceOrDrop   DirtyValues = "coerce_or_drop"
	DirtyValuesDrop           DirtyValues = "drop"
	DirtyValuesReject         DirtyValues = "reject"
)

// DocumentParams are optional query-string parameters for document index /
// update endpoints. A nil *DocumentParams means "use server defaults".
type DocumentParams struct {
	Action      *IndexAction
	DirtyValues *DirtyValues
}

func (p *DocumentParams) values() url.Values {
	if p == nil {
		return nil
	}
	v := url.Values{}
	if p.Action != nil {
		v.Set("action", string(*p.Action))
	}
	if p.DirtyValues != nil {
		v.Set("dirty_values", string(*p.DirtyValues))
	}
	if len(v) == 0 {
		return nil
	}
	return v
}

// IndexDocument inserts a document into a collection.
// POST /collections/{name}/documents — https://typesense.org/docs/30.2/api/documents.html#index-a-document
func (c *Client) IndexDocument(ctx context.Context, collection string, doc map[string]any, params *DocumentParams) (map[string]any, error) {
	out := map[string]any{}
	if err := c.do(ctx, "POST", "/collections/"+collection+"/documents", params.values(), doc, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDocument retrieves a document by id.
// GET /collections/{name}/documents/{id} — https://typesense.org/docs/30.2/api/documents.html#retrieve-a-document
func (c *Client) GetDocument(ctx context.Context, collection, id string) (map[string]any, error) {
	out := map[string]any{}
	if err := c.do(ctx, "GET", "/collections/"+collection+"/documents/"+id, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateDocument patches a document by id.
// PATCH /collections/{name}/documents/{id} — https://typesense.org/docs/30.2/api/documents.html#update-a-document
func (c *Client) UpdateDocument(ctx context.Context, collection, id string, doc map[string]any, params *DocumentParams) (map[string]any, error) {
	out := map[string]any{}
	if err := c.do(ctx, "PATCH", "/collections/"+collection+"/documents/"+id, params.values(), doc, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteDocument removes a document by id.
// DELETE /collections/{name}/documents/{id} — https://typesense.org/docs/30.2/api/documents.html#delete-a-document
func (c *Client) DeleteDocument(ctx context.Context, collection, id string) error {
	return c.do(ctx, "DELETE", "/collections/"+collection+"/documents/"+id, nil, nil, nil)
}
