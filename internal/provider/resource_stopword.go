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

var _ resource.Resource = &StopwordResource{}
var _ resource.ResourceWithImportState = &StopwordResource{}

func NewStopwordResource() resource.Resource {
	return &StopwordResource{}
}

type StopwordResource struct {
	client *typesense.Client
}

type StopwordResourceModel struct {
	Id        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	Locale    types.String   `tfsdk:"locale"`
	Stopwords []types.String `tfsdk:"stopwords"`
}

func (r *StopwordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stopword"
}

func (r *StopwordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A stopwords set is a named list of common words removed from search queries that reference this set via the `stopwords` search parameter.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locale": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Locale of the stopwords (e.g. `en`, `de`).",
			},
			"stopwords": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
			},
		},
	}
}

func (r *StopwordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StopwordResource) upsert(ctx context.Context, data *StopwordResourceModel) error {
	body := &api.StopwordsSetUpsertSchema{
		Stopwords: convertTerraformArrayToStringArray(data.Stopwords),
	}
	if !data.Locale.IsNull() && data.Locale.ValueString() != "" {
		loc := data.Locale.ValueString()
		body.Locale = &loc
	}
	_, err := r.client.Stopwords().Upsert(ctx, data.Name.ValueString(), body)
	return err
}

func (r *StopwordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StopwordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create stopword set: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StopwordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StopwordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	set, err := r.client.Stopword(data.Id.ValueString()).Retrieve(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve stopword set: %s", err))
		return
	}
	data.Name = types.StringValue(set.Id)
	if set.Locale != nil {
		data.Locale = types.StringValue(*set.Locale)
	}
	data.Stopwords = convertStringArrayToTerraformArray(set.Stopwords)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StopwordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StopwordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update stopword set: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StopwordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StopwordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Stopword(data.Id.ValueString()).Delete(ctx)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete stopword set: %s", err))
	}
}

func (r *StopwordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
