package typesense

import "context"

// ConversationModel configures Typesense's RAG-style conversational search.
type ConversationModel struct {
	Id                string  `json:"id"`
	ModelName         string  `json:"model_name"`
	HistoryCollection string  `json:"history_collection"`
	MaxBytes          int     `json:"max_bytes"`
	ApiKey            *string `json:"api_key,omitempty"`
	SystemPrompt      *string `json:"system_prompt,omitempty"`
	Ttl               *int    `json:"ttl,omitempty"`
	AccountId         *string `json:"account_id,omitempty"`
	VllmUrl           *string `json:"vllm_url,omitempty"`
}

// ConversationModelCreateSchema is the body for POST /conversations/models.
type ConversationModelCreateSchema struct {
	ModelName         string  `json:"model_name"`
	HistoryCollection string  `json:"history_collection"`
	MaxBytes          int     `json:"max_bytes"`
	ApiKey            *string `json:"api_key,omitempty"`
	SystemPrompt      *string `json:"system_prompt,omitempty"`
	Ttl               *int    `json:"ttl,omitempty"`
	AccountId         *string `json:"account_id,omitempty"`
	VllmUrl           *string `json:"vllm_url,omitempty"`
}

// ConversationModelUpdateSchema is the body for PUT /conversations/models/{id}.
type ConversationModelUpdateSchema struct {
	ModelName         *string `json:"model_name,omitempty"`
	HistoryCollection *string `json:"history_collection,omitempty"`
	MaxBytes          *int    `json:"max_bytes,omitempty"`
	ApiKey            *string `json:"api_key,omitempty"`
	SystemPrompt      *string `json:"system_prompt,omitempty"`
	Ttl               *int    `json:"ttl,omitempty"`
	AccountId         *string `json:"account_id,omitempty"`
	VllmUrl           *string `json:"vllm_url,omitempty"`
}

// CreateConversationModel creates a conversation model.
// POST /conversations/models — https://typesense.org/docs/30.2/api/conversational-search-rag.html#create-a-conversation-model
func (c *Client) CreateConversationModel(ctx context.Context, body *ConversationModelCreateSchema) (*ConversationModel, error) {
	out := &ConversationModel{}
	if err := c.do(ctx, "POST", "/conversations/models", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetConversationModel retrieves a conversation model.
// GET /conversations/models/{id} — https://typesense.org/docs/30.2/api/conversational-search-rag.html#retrieve-a-conversation-model
func (c *Client) GetConversationModel(ctx context.Context, id string) (*ConversationModel, error) {
	out := &ConversationModel{}
	if err := c.do(ctx, "GET", "/conversations/models/"+id, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateConversationModel replaces an existing conversation model's settings.
// PUT /conversations/models/{id} — https://typesense.org/docs/30.2/api/conversational-search-rag.html#update-a-conversation-model
func (c *Client) UpdateConversationModel(ctx context.Context, id string, body *ConversationModelUpdateSchema) (*ConversationModel, error) {
	out := &ConversationModel{}
	if err := c.do(ctx, "PUT", "/conversations/models/"+id, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteConversationModel removes a conversation model.
// DELETE /conversations/models/{id} — https://typesense.org/docs/30.2/api/conversational-search-rag.html#delete-a-conversation-model
func (c *Client) DeleteConversationModel(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/conversations/models/"+id, nil, nil, nil)
}
