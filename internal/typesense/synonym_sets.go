package typesense

import "context"

// SynonymItem is a single synonym mapping inside a synonym set.
type SynonymItem struct {
	Id             string    `json:"id"`
	Synonyms       []string  `json:"synonyms"`
	Root           *string   `json:"root,omitempty"`
	Locale         *string   `json:"locale,omitempty"`
	SymbolsToIndex *[]string `json:"symbols_to_index,omitempty"`
}

// SynonymSet is the upsert/retrieve shape for /synonym_sets/{name}.
type SynonymSet struct {
	Items []SynonymItem `json:"items"`
}

// UpsertSynonymSet creates or replaces a synonym set.
func (c *Client) UpsertSynonymSet(ctx context.Context, name string, body *SynonymSet) (*SynonymSet, error) {
	out := &SynonymSet{}
	if err := c.do(ctx, "PUT", "/synonym_sets/"+name, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetSynonymSet retrieves a synonym set by name.
func (c *Client) GetSynonymSet(ctx context.Context, name string) (*SynonymSet, error) {
	out := &SynonymSet{}
	if err := c.do(ctx, "GET", "/synonym_sets/"+name, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteSynonymSet removes a synonym set.
func (c *Client) DeleteSynonymSet(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/synonym_sets/"+name, nil, nil, nil)
}
