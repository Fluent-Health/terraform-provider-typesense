package typesense

import (
	"context"
	"encoding/json"
)

// Preset is the response shape for GET /presets/{name}.
type Preset struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// PresetUpsertSchema is the body for PUT /presets/{name}. value is a free-form
// JSON object that is either a single search-parameter blob or a `searches` envelope.
type PresetUpsertSchema struct {
	Value json.RawMessage `json:"value"`
}

// UpsertPreset creates or replaces a preset.
// PUT /presets/{name} — https://typesense.org/docs/30.2/api/search.html#presets
func (c *Client) UpsertPreset(ctx context.Context, name string, body *PresetUpsertSchema) (*Preset, error) {
	out := &Preset{}
	if err := c.do(ctx, "PUT", "/presets/"+name, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPreset retrieves a preset by name.
// GET /presets/{name} — https://typesense.org/docs/30.2/api/search.html#presets
func (c *Client) GetPreset(ctx context.Context, name string) (*Preset, error) {
	out := &Preset{}
	if err := c.do(ctx, "GET", "/presets/"+name, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeletePreset removes a preset.
// DELETE /presets/{name} — https://typesense.org/docs/30.2/api/search.html#presets
func (c *Client) DeletePreset(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/presets/"+name, nil, nil, nil)
}
