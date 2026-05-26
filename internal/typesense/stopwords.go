package typesense

import "context"

// Stopwords is the retrieve shape for GET /stopwords/{name}.
type Stopwords struct {
	Id        string   `json:"id"`
	Locale    *string  `json:"locale,omitempty"`
	Stopwords []string `json:"stopwords"`
}

// stopwordsGetEnvelope is the wire shape returned by GET /stopwords/{name}.
type stopwordsGetEnvelope struct {
	Stopwords Stopwords `json:"stopwords"`
}

// StopwordsUpsertSchema is the body for PUT /stopwords/{name}.
type StopwordsUpsertSchema struct {
	Stopwords []string `json:"stopwords"`
	Locale    *string  `json:"locale,omitempty"`
}

// UpsertStopwords creates or replaces a stopwords set.
// PUT /stopwords/{name} — https://typesense.org/docs/30.2/api/stopwords.html#upsert-a-stopwords-set
func (c *Client) UpsertStopwords(ctx context.Context, name string, body *StopwordsUpsertSchema) (*Stopwords, error) {
	out := &Stopwords{}
	if err := c.do(ctx, "PUT", "/stopwords/"+name, nil, body, out); err != nil {
		return nil, err
	}
	if out.Id == "" {
		out.Id = name
	}
	return out, nil
}

// GetStopwords retrieves a stopwords set by name. The server wraps the set in
// a top-level "stopwords" object; this method unwraps it.
// GET /stopwords/{name} — https://typesense.org/docs/30.2/api/stopwords.html#retrieve-a-stopwords-set
func (c *Client) GetStopwords(ctx context.Context, name string) (*Stopwords, error) {
	env := &stopwordsGetEnvelope{}
	if err := c.do(ctx, "GET", "/stopwords/"+name, nil, nil, env); err != nil {
		return nil, err
	}
	return &env.Stopwords, nil
}

// DeleteStopwords removes a stopwords set.
// DELETE /stopwords/{name} — https://typesense.org/docs/30.2/api/stopwords.html#delete-a-stopwords-set
func (c *Client) DeleteStopwords(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/stopwords/"+name, nil, nil, nil)
}
