package typesense

import "context"

// NLSearchModel is a natural-language search model definition.
//
// ServiceAccount carries a GCP-Vertex service-account credential; it is an
// alternative to the access_token/refresh_token/client_id/client_secret tuple
// and is accepted by the server but missing from the OpenAPI spec and SDK.
type NLSearchModel struct {
	Id              string             `json:"id"`
	ModelName       *string            `json:"model_name,omitempty"`
	ApiKey          *string            `json:"api_key,omitempty"`
	ApiUrl          *string            `json:"api_url,omitempty"`
	ApiVersion      *string            `json:"api_version,omitempty"`
	SystemPrompt    *string            `json:"system_prompt,omitempty"`
	MaxBytes        *int               `json:"max_bytes,omitempty"`
	Temperature     *float32           `json:"temperature,omitempty"`
	TopK            *int               `json:"top_k,omitempty"`
	TopP            *float32           `json:"top_p,omitempty"`
	MaxOutputTokens *int               `json:"max_output_tokens,omitempty"`
	StopSequences   *[]string          `json:"stop_sequences,omitempty"`
	AccountId       *string            `json:"account_id,omitempty"`
	AccessToken     *string            `json:"access_token,omitempty"`
	RefreshToken    *string            `json:"refresh_token,omitempty"`
	ClientId        *string            `json:"client_id,omitempty"`
	ClientSecret    *string            `json:"client_secret,omitempty"`
	ProjectId       *string            `json:"project_id,omitempty"`
	Region          *string            `json:"region,omitempty"`
	ServiceAccount  *GCPServiceAccount `json:"service_account,omitempty"`
}

// NLSearchModelUpsertSchema is shared by create (POST) and update (PUT) bodies.
type NLSearchModelUpsertSchema struct {
	ModelName       *string            `json:"model_name,omitempty"`
	ApiKey          *string            `json:"api_key,omitempty"`
	ApiUrl          *string            `json:"api_url,omitempty"`
	ApiVersion      *string            `json:"api_version,omitempty"`
	SystemPrompt    *string            `json:"system_prompt,omitempty"`
	MaxBytes        *int               `json:"max_bytes,omitempty"`
	Temperature     *float32           `json:"temperature,omitempty"`
	TopK            *int               `json:"top_k,omitempty"`
	TopP            *float32           `json:"top_p,omitempty"`
	MaxOutputTokens *int               `json:"max_output_tokens,omitempty"`
	StopSequences   *[]string          `json:"stop_sequences,omitempty"`
	AccountId       *string            `json:"account_id,omitempty"`
	AccessToken     *string            `json:"access_token,omitempty"`
	RefreshToken    *string            `json:"refresh_token,omitempty"`
	ClientId        *string            `json:"client_id,omitempty"`
	ClientSecret    *string            `json:"client_secret,omitempty"`
	ProjectId       *string            `json:"project_id,omitempty"`
	Region          *string            `json:"region,omitempty"`
	ServiceAccount  *GCPServiceAccount `json:"service_account,omitempty"`
}

// CreateNLSearchModel creates an NL search model.
// POST /nl_search_models — https://typesense.org/docs/30.2/api/natural-language-search.html#create-a-natural-language-search-model
func (c *Client) CreateNLSearchModel(ctx context.Context, body *NLSearchModelUpsertSchema) (*NLSearchModel, error) {
	out := &NLSearchModel{}
	if err := c.do(ctx, "POST", "/nl_search_models", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetNLSearchModel retrieves an NL search model.
// GET /nl_search_models/{id} — https://typesense.org/docs/30.2/api/natural-language-search.html#retrieve-a-natural-language-search-model
func (c *Client) GetNLSearchModel(ctx context.Context, id string) (*NLSearchModel, error) {
	out := &NLSearchModel{}
	if err := c.do(ctx, "GET", "/nl_search_models/"+id, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateNLSearchModel replaces an existing NL search model's settings.
// PUT /nl_search_models/{id} — https://typesense.org/docs/30.2/api/natural-language-search.html#update-a-natural-language-search-model
func (c *Client) UpdateNLSearchModel(ctx context.Context, id string, body *NLSearchModelUpsertSchema) (*NLSearchModel, error) {
	out := &NLSearchModel{}
	if err := c.do(ctx, "PUT", "/nl_search_models/"+id, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteNLSearchModel removes an NL search model.
// DELETE /nl_search_models/{id} — https://typesense.org/docs/30.2/api/natural-language-search.html#delete-a-natural-language-search-model
func (c *Client) DeleteNLSearchModel(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/nl_search_models/"+id, nil, nil, nil)
}
