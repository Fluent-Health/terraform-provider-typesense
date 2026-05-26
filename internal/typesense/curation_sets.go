package typesense

import "context"

// CurationRule is the matching rule for a curation item.
type CurationRule struct {
	Query    *string   `json:"query,omitempty"`
	Match    *string   `json:"match,omitempty"`
	FilterBy *string   `json:"filter_by,omitempty"`
	Tags     *[]string `json:"tags,omitempty"`
}

// CurationInclude pins a document at a specific position in results.
type CurationInclude struct {
	Id       string `json:"id"`
	Position int    `json:"position"`
}

// CurationExclude removes a specific document from results.
type CurationExclude struct {
	Id string `json:"id"`
}

// CurationItem is a single rule+action pair inside a curation set.
type CurationItem struct {
	Id                  *string            `json:"id,omitempty"`
	Rule                CurationRule       `json:"rule"`
	Includes            *[]CurationInclude `json:"includes,omitempty"`
	Excludes            *[]CurationExclude `json:"excludes,omitempty"`
	FilterBy            *string            `json:"filter_by,omitempty"`
	SortBy              *string            `json:"sort_by,omitempty"`
	ReplaceQuery        *string            `json:"replace_query,omitempty"`
	RemoveMatchedTokens *bool              `json:"remove_matched_tokens,omitempty"`
	FilterCuratedHits   *bool              `json:"filter_curated_hits,omitempty"`
	StopProcessing      *bool              `json:"stop_processing,omitempty"`
	EffectiveFromTs     *int               `json:"effective_from_ts,omitempty"`
	EffectiveToTs       *int               `json:"effective_to_ts,omitempty"`
}

// CurationSet is the upsert/retrieve shape for /curation_sets/{name}.
type CurationSet struct {
	Description *string        `json:"description,omitempty"`
	Items       []CurationItem `json:"items"`
}

// UpsertCurationSet creates or replaces a curation set.
func (c *Client) UpsertCurationSet(ctx context.Context, name string, body *CurationSet) (*CurationSet, error) {
	out := &CurationSet{}
	if err := c.do(ctx, "PUT", "/curation_sets/"+name, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCurationSet retrieves a curation set by name.
func (c *Client) GetCurationSet(ctx context.Context, name string) (*CurationSet, error) {
	out := &CurationSet{}
	if err := c.do(ctx, "GET", "/curation_sets/"+name, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteCurationSet removes a curation set.
func (c *Client) DeleteCurationSet(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/curation_sets/"+name, nil, nil, nil)
}
