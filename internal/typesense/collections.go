package typesense

import "context"

// Field is a single column in a Typesense collection schema.
type Field struct {
	Name            string      `json:"name"`
	Type            string      `json:"type"`
	Facet           *bool       `json:"facet,omitempty"`
	Index           *bool       `json:"index,omitempty"`
	Optional        *bool       `json:"optional,omitempty"`
	Sort            *bool       `json:"sort,omitempty"`
	Infix           *bool       `json:"infix,omitempty"`
	Stem            *bool       `json:"stem,omitempty"`
	StemDictionary  *string     `json:"stem_dictionary,omitempty"`
	Locale          *string     `json:"locale,omitempty"`
	Store           *bool       `json:"store,omitempty"`
	NumDim          *int        `json:"num_dim,omitempty"`
	Reference       *string     `json:"reference,omitempty"`
	AsyncReference  *bool       `json:"async_reference,omitempty"`
	RangeIndex      *bool       `json:"range_index,omitempty"`
	VecDist         *string     `json:"vec_dist,omitempty"`
	SymbolsToIndex  *[]string   `json:"symbols_to_index,omitempty"`
	TokenSeparators *[]string   `json:"token_separators,omitempty"`
	Embed           *FieldEmbed `json:"embed,omitempty"`
	Drop            *bool       `json:"drop,omitempty"`
}

// FieldEmbed configures an auto-embedded vector field.
type FieldEmbed struct {
	From        []string              `json:"from"`
	ModelConfig FieldEmbedModelConfig `json:"model_config"`
}

// FieldEmbedModelConfig captures the embedding-model parameters.
//
// Region and ServiceAccount enable the GCP-Vertex service-account auth path
// accepted by the Typesense server but missing from the OpenAPI spec and SDK.
type FieldEmbedModelConfig struct {
	ModelName      string                    `json:"model_name"`
	Url            *string                   `json:"url,omitempty"`
	AccessToken    *string                   `json:"access_token,omitempty"`
	ApiKey         *string                   `json:"api_key,omitempty"`
	ClientId       *string                   `json:"client_id,omitempty"`
	ClientSecret   *string                   `json:"client_secret,omitempty"`
	IndexingPrefix *string                   `json:"indexing_prefix,omitempty"`
	ProjectId      *string                   `json:"project_id,omitempty"`
	QueryPrefix    *string                   `json:"query_prefix,omitempty"`
	RefreshToken   *string                   `json:"refresh_token,omitempty"`
	Region         *string                   `json:"region,omitempty"`
	ServiceAccount *FieldEmbedServiceAccount `json:"service_account,omitempty"`
}

// FieldEmbedServiceAccount carries a GCP service-account credential used by the
// embedder. token_uri defaults to Google's OAuth token endpoint if omitted.
type FieldEmbedServiceAccount struct {
	ClientEmail string  `json:"client_email"`
	PrivateKey  string  `json:"private_key"`
	TokenURI    *string `json:"token_uri,omitempty"`
}

// Collection is a Typesense collection (the server's response shape).
type Collection struct {
	Name                string    `json:"name"`
	DefaultSortingField *string   `json:"default_sorting_field,omitempty"`
	EnableNestedFields  *bool     `json:"enable_nested_fields,omitempty"`
	Fields              []Field   `json:"fields"`
	SymbolsToIndex      *[]string `json:"symbols_to_index,omitempty"`
	TokenSeparators     *[]string `json:"token_separators,omitempty"`
	NumDocuments        *int64    `json:"num_documents,omitempty"`
	CreatedAt           *int64    `json:"created_at,omitempty"`
}

// CollectionCreateSchema is the body for POST /collections.
type CollectionCreateSchema struct {
	Name                string    `json:"name"`
	Fields              []Field   `json:"fields"`
	DefaultSortingField *string   `json:"default_sorting_field,omitempty"`
	EnableNestedFields  *bool     `json:"enable_nested_fields,omitempty"`
	SymbolsToIndex      *[]string `json:"symbols_to_index,omitempty"`
	TokenSeparators     *[]string `json:"token_separators,omitempty"`
}

// CollectionUpdateSchema is the body for PATCH /collections/{name}. Field-only.
type CollectionUpdateSchema struct {
	Fields []Field `json:"fields"`
}

// CreateCollection creates a new collection.
func (c *Client) CreateCollection(ctx context.Context, body *CollectionCreateSchema) (*Collection, error) {
	out := &Collection{}
	if err := c.do(ctx, "POST", "/collections", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCollection retrieves a collection by name.
func (c *Client) GetCollection(ctx context.Context, name string) (*Collection, error) {
	out := &Collection{}
	if err := c.do(ctx, "GET", "/collections/"+name, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateCollection patches a collection's field set.
func (c *Client) UpdateCollection(ctx context.Context, name string, body *CollectionUpdateSchema) (*CollectionUpdateSchema, error) {
	out := &CollectionUpdateSchema{}
	if err := c.do(ctx, "PATCH", "/collections/"+name, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteCollection drops a collection.
func (c *Client) DeleteCollection(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/collections/"+name, nil, nil, nil)
}
