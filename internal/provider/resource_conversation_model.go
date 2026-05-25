package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/typesense/typesense-go/v4/typesense"
	"github.com/typesense/typesense-go/v4/typesense/api"
)

var _ resource.Resource = &ConversationModelResource{}
var _ resource.ResourceWithImportState = &ConversationModelResource{}

func NewConversationModelResource() resource.Resource {
	return &ConversationModelResource{}
}

type ConversationModelResource struct {
	client *typesense.Client
}

type ConversationModelResourceModel struct {
	Id                types.String `tfsdk:"id"`
	ModelName         types.String `tfsdk:"model_name"`
	HistoryCollection types.String `tfsdk:"history_collection"`
	ApiKey            types.String `tfsdk:"api_key"`
	SystemPrompt      types.String `tfsdk:"system_prompt"`
	Ttl               types.Int64  `tfsdk:"ttl"`
	MaxBytes          types.Int64  `tfsdk:"max_bytes"`
	AccountId         types.String `tfsdk:"account_id"`
	VllmUrl           types.String `tfsdk:"vllm_url"`
}

func (r *ConversationModelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_conversation_model"
}

func (r *ConversationModelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A conversation model configures Typesense's RAG-style conversational search. References an external LLM (OpenAI, Cloudflare, vLLM) and the Typesense collection that stores chat history.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"model_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "LLM model name (e.g. `openai/gpt-4`, `cloudflare/@cf/meta/llama-2-7b-chat-int8`).",
			},
			"history_collection": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the Typesense collection that stores conversation history.",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API key for the upstream LLM provider.",
			},
			"system_prompt": schema.StringAttribute{Optional: true},
			"ttl": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Seconds after which chat history entries are deleted. Default: 86400 (24h).",
			},
			"max_bytes": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Maximum bytes to include in each LLM call context window.",
			},
			"account_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "LLM provider account ID (Cloudflare only).",
			},
			"vllm_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "URL of a self-hosted vLLM service.",
			},
		},
	}
}

func (r *ConversationModelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*typesense.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *typesense.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *ConversationModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConversationModelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := &api.ConversationModelCreateSchema{
		ModelName:         data.ModelName.ValueString(),
		HistoryCollection: data.HistoryCollection.ValueString(),
		MaxBytes:          int(data.MaxBytes.ValueInt64()),
		ApiKey:            data.ApiKey.ValueStringPointer(),
		SystemPrompt:      data.SystemPrompt.ValueStringPointer(),
		AccountId:         data.AccountId.ValueStringPointer(),
		VllmUrl:           data.VllmUrl.ValueStringPointer(),
	}
	if !data.Ttl.IsNull() && !data.Ttl.IsUnknown() {
		v := int(data.Ttl.ValueInt64())
		body.Ttl = &v
	}

	model, err := r.client.Conversations().Models().Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create conversation model: %s", err))
		return
	}
	data.Id = types.StringValue(model.Id)
	if model.Ttl != nil {
		data.Ttl = types.Int64Value(int64(*model.Ttl))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConversationModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConversationModelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	model, err := r.client.Conversations().Model(data.Id.ValueString()).Retrieve(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve conversation model: %s", err))
		return
	}
	data.ModelName = types.StringValue(model.ModelName)
	data.HistoryCollection = types.StringValue(model.HistoryCollection)
	data.MaxBytes = types.Int64Value(int64(model.MaxBytes))
	if model.SystemPrompt != nil {
		data.SystemPrompt = types.StringValue(*model.SystemPrompt)
	}
	if model.Ttl != nil {
		data.Ttl = types.Int64Value(int64(*model.Ttl))
	}
	if model.AccountId != nil {
		data.AccountId = types.StringValue(*model.AccountId)
	}
	if model.VllmUrl != nil {
		data.VllmUrl = types.StringValue(*model.VllmUrl)
	}
	// api_key is not returned by the server; preserve state value.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConversationModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConversationModelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	maxBytes := int(data.MaxBytes.ValueInt64())
	body := &api.ConversationModelUpdateSchema{
		ModelName:         data.ModelName.ValueStringPointer(),
		HistoryCollection: data.HistoryCollection.ValueStringPointer(),
		ApiKey:            data.ApiKey.ValueStringPointer(),
		SystemPrompt:      data.SystemPrompt.ValueStringPointer(),
		AccountId:         data.AccountId.ValueStringPointer(),
		VllmUrl:           data.VllmUrl.ValueStringPointer(),
		MaxBytes:          &maxBytes,
	}
	if !data.Ttl.IsNull() && !data.Ttl.IsUnknown() {
		v := int(data.Ttl.ValueInt64())
		body.Ttl = &v
	}
	model, err := r.client.Conversations().Model(data.Id.ValueString()).Update(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update conversation model: %s", err))
		return
	}
	if model.Ttl != nil {
		data.Ttl = types.Int64Value(int64(*model.Ttl))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConversationModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConversationModelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Conversations().Model(data.Id.ValueString()).Delete(ctx)
	if err != nil && !strings.Contains(err.Error(), "Not Found") && !strings.Contains(err.Error(), "not found") {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete conversation model: %s", err))
	}
}

func (r *ConversationModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
