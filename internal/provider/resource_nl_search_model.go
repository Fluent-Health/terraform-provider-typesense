package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"ronati-terraform-typesense/internal/typesense"
)

var _ resource.Resource = &NLSearchModelResource{}
var _ resource.ResourceWithImportState = &NLSearchModelResource{}

func NewNLSearchModelResource() resource.Resource {
	return &NLSearchModelResource{}
}

type NLSearchModelResource struct {
	client *typesense.Client
}

type NLSearchModelResourceModel struct {
	Id              types.String   `tfsdk:"id"`
	ModelName       types.String   `tfsdk:"model_name"`
	ApiKey          types.String   `tfsdk:"api_key"`
	ApiUrl          types.String   `tfsdk:"api_url"`
	ApiVersion      types.String   `tfsdk:"api_version"`
	SystemPrompt    types.String   `tfsdk:"system_prompt"`
	MaxBytes        types.Int64    `tfsdk:"max_bytes"`
	Temperature     types.Float64  `tfsdk:"temperature"`
	TopK            types.Int64    `tfsdk:"top_k"`
	TopP            types.Float64  `tfsdk:"top_p"`
	MaxOutputTokens types.Int64    `tfsdk:"max_output_tokens"`
	StopSequences   []types.String `tfsdk:"stop_sequences"`
	AccountId       types.String   `tfsdk:"account_id"`
	AccessToken     types.String   `tfsdk:"access_token"`
	RefreshToken    types.String   `tfsdk:"refresh_token"`
	ClientId        types.String   `tfsdk:"client_id"`
	ClientSecret    types.String   `tfsdk:"client_secret"`
	ProjectId       types.String   `tfsdk:"project_id"`
	Region          types.String   `tfsdk:"region"`
}

func (r *NLSearchModelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nl_search_model"
}

func (r *NLSearchModelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Natural Language Search model (Typesense v29+). Translates a free-form query string into a structured Typesense search request via an LLM.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"model_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "LLM model name (e.g. `openai/gpt-4o`, `google/gemini-2.5-flash`, `cloudflare/@cf/meta/llama-2-7b-chat-int8`).",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API key for the upstream LLM provider.",
			},
			"api_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom API URL for the LLM service.",
			},
			"api_version": schema.StringAttribute{
				Optional: true,
			},
			"system_prompt": schema.StringAttribute{Optional: true},
			"max_bytes": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"temperature":       schema.Float64Attribute{Optional: true},
			"top_k":             schema.Int64Attribute{Optional: true},
			"top_p":             schema.Float64Attribute{Optional: true},
			"max_output_tokens": schema.Int64Attribute{Optional: true},
			"stop_sequences": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"account_id":    schema.StringAttribute{Optional: true, MarkdownDescription: "Cloudflare account ID."},
			"access_token":  schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Access token (GCP Vertex AI)."},
			"refresh_token": schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Refresh token (GCP Vertex AI)."},
			"client_id":     schema.StringAttribute{Optional: true, MarkdownDescription: "Client ID (GCP Vertex AI)."},
			"client_secret": schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Client secret (GCP Vertex AI)."},
			"project_id":    schema.StringAttribute{Optional: true, MarkdownDescription: "Project ID (GCP Vertex AI)."},
			"region":        schema.StringAttribute{Optional: true, MarkdownDescription: "Region (GCP Vertex AI)."},
		},
	}
}

func (r *NLSearchModelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func nlModelToCreateSchema(d *NLSearchModelResourceModel) *typesense.NLSearchModelUpsertSchema {
	out := &typesense.NLSearchModelUpsertSchema{
		ModelName:    d.ModelName.ValueStringPointer(),
		ApiKey:       d.ApiKey.ValueStringPointer(),
		ApiUrl:       d.ApiUrl.ValueStringPointer(),
		ApiVersion:   d.ApiVersion.ValueStringPointer(),
		SystemPrompt: d.SystemPrompt.ValueStringPointer(),
		AccountId:    d.AccountId.ValueStringPointer(),
		AccessToken:  d.AccessToken.ValueStringPointer(),
		RefreshToken: d.RefreshToken.ValueStringPointer(),
		ClientId:     d.ClientId.ValueStringPointer(),
		ClientSecret: d.ClientSecret.ValueStringPointer(),
		ProjectId:    d.ProjectId.ValueStringPointer(),
		Region:       d.Region.ValueStringPointer(),
	}
	if !d.MaxBytes.IsNull() && !d.MaxBytes.IsUnknown() {
		v := int(d.MaxBytes.ValueInt64())
		out.MaxBytes = &v
	}
	if !d.MaxOutputTokens.IsNull() && !d.MaxOutputTokens.IsUnknown() {
		v := int(d.MaxOutputTokens.ValueInt64())
		out.MaxOutputTokens = &v
	}
	if !d.Temperature.IsNull() && !d.Temperature.IsUnknown() {
		v := float32(d.Temperature.ValueFloat64())
		out.Temperature = &v
	}
	if !d.TopK.IsNull() && !d.TopK.IsUnknown() {
		v := int(d.TopK.ValueInt64())
		out.TopK = &v
	}
	if !d.TopP.IsNull() && !d.TopP.IsUnknown() {
		v := float32(d.TopP.ValueFloat64())
		out.TopP = &v
	}
	if len(d.StopSequences) > 0 {
		ss := convertTerraformArrayToStringArray(d.StopSequences)
		out.StopSequences = &ss
	}
	return out
}

func applyNLSchemaToModel(s *typesense.NLSearchModel, d *NLSearchModelResourceModel) {
	d.Id = types.StringValue(s.Id)
	if s.ModelName != nil {
		d.ModelName = types.StringValue(*s.ModelName)
	}
	if s.ApiUrl != nil {
		d.ApiUrl = types.StringValue(*s.ApiUrl)
	}
	if s.ApiVersion != nil {
		d.ApiVersion = types.StringValue(*s.ApiVersion)
	}
	if s.SystemPrompt != nil {
		d.SystemPrompt = types.StringValue(*s.SystemPrompt)
	}
	if s.MaxBytes != nil {
		d.MaxBytes = types.Int64Value(int64(*s.MaxBytes))
	}
	if s.Temperature != nil {
		d.Temperature = types.Float64Value(float64(*s.Temperature))
	}
	if s.TopK != nil {
		d.TopK = types.Int64Value(int64(*s.TopK))
	}
	if s.TopP != nil {
		d.TopP = types.Float64Value(float64(*s.TopP))
	}
	if s.MaxOutputTokens != nil {
		d.MaxOutputTokens = types.Int64Value(int64(*s.MaxOutputTokens))
	}
	if s.StopSequences != nil {
		d.StopSequences = convertStringArrayToTerraformArray(*s.StopSequences)
	}
	if s.AccountId != nil {
		d.AccountId = types.StringValue(*s.AccountId)
	}
	if s.ProjectId != nil {
		d.ProjectId = types.StringValue(*s.ProjectId)
	}
	if s.Region != nil {
		d.Region = types.StringValue(*s.Region)
	}
	if s.ClientId != nil {
		d.ClientId = types.StringValue(*s.ClientId)
	}
	// api_key, access_token, refresh_token, client_secret are sensitive and not echoed; keep state values.
}

func (r *NLSearchModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NLSearchModelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := nlModelToCreateSchema(&data)
	model, err := r.client.CreateNLSearchModel(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create NL search model: %s", err))
		return
	}
	applyNLSchemaToModel(model, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NLSearchModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NLSearchModelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	model, err := r.client.GetNLSearchModel(ctx, data.Id.ValueString())
	if err != nil {
		if typesense.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve NL search model: %s", err))
		return
	}
	applyNLSchemaToModel(model, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NLSearchModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NLSearchModelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := nlModelToCreateSchema(&data)
	model, err := r.client.UpdateNLSearchModel(ctx, data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update NL search model: %s", err))
		return
	}
	applyNLSchemaToModel(model, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NLSearchModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NLSearchModelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteNLSearchModel(ctx, data.Id.ValueString())
	if err != nil && !typesense.IsNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete NL search model: %s", err))
	}
}

func (r *NLSearchModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
