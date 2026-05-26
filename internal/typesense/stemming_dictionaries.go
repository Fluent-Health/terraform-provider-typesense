package typesense

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
)

// StemmingDictionaryWord is one word→root entry.
type StemmingDictionaryWord struct {
	Word string `json:"word"`
	Root string `json:"root"`
}

// StemmingDictionary is the retrieve shape for GET /stemming/dictionaries/{id}.
type StemmingDictionary struct {
	Id    string                   `json:"id"`
	Words []StemmingDictionaryWord `json:"words"`
}

// UpsertStemmingDictionary uploads the given word→root mappings as JSONL.
// POST /stemming/dictionaries/import?id={name} — https://typesense.org/docs/30.2/api/stemming.html#import-stemming-dictionary
func (c *Client) UpsertStemmingDictionary(ctx context.Context, name string, words []StemmingDictionaryWord) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, w := range words {
		if err := enc.Encode(w); err != nil {
			return err
		}
	}
	q := url.Values{"id": []string{name}}
	return c.doWithContentType(ctx, "POST", "/stemming/dictionaries/import", q, "application/octet-stream", buf.Bytes(), nil)
}

// GetStemmingDictionary retrieves a stemming dictionary by id.
// GET /stemming/dictionaries/{id} — https://typesense.org/docs/30.2/api/stemming.html#retrieve-a-stemming-dictionary
func (c *Client) GetStemmingDictionary(ctx context.Context, id string) (*StemmingDictionary, error) {
	out := &StemmingDictionary{}
	if err := c.do(ctx, "GET", "/stemming/dictionaries/"+id, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
