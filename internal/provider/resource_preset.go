package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/typesense/typesense-go/v4/typesense"
	"github.com/typesense/typesense-go/v4/typesense/api"
)

var _ resource.Resource = &PresetResource{}
var _ resource.ResourceWithImportState = &PresetResource{}

func NewPresetResource() resource.Resource {
	return &PresetResource{}
}

type PresetResource struct {
	client *typesense.Client
}

type PresetResourceModel struct {
	Id    types.String         `tfsdk:"id"`
	Name  types.String         `tfsdk:"name"`
	Value jsontypes.Normalized `tfsdk:"value"`
}

func (r *PresetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_preset"
}

func (r *PresetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A search preset stores a JSON blob of search parameters under a name so clients can reference it via the `preset` search parameter.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Preset name (used as the ID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:            true,
				CustomType:          jsontypes.NormalizedType{},
				MarkdownDescription: "JSON-encoded search parameters (or a `searches` object for multi-search).",
			},
		},
	}
}

func (r *PresetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PresetResource) upsert(ctx context.Context, data *PresetResourceModel) error {
	upsertSchema := &api.PresetUpsertSchema{}
	if err := upsertSchema.Value.UnmarshalJSON([]byte(data.Value.ValueString())); err != nil {
		return fmt.Errorf("invalid preset value JSON: %w", err)
	}
	_, err := r.client.Presets().Upsert(ctx, data.Name.ValueString(), upsertSchema)
	return err
}

func (r *PresetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PresetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create preset: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PresetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PresetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	preset, err := r.client.Preset(data.Id.ValueString()).Retrieve(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve preset: %s", err))
		return
	}
	data.Name = types.StringValue(preset.Name)
	valueBytes, err := preset.Value.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError("JSON error", fmt.Sprintf("Unable to marshal preset value: %s", err))
		return
	}
	data.Value = jsontypes.NewNormalizedValue(string(valueBytes))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PresetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PresetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update preset: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PresetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PresetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Warn(ctx, "Delete preset "+data.Id.ValueString())
	_, err := r.client.Preset(data.Id.ValueString()).Delete(ctx)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete preset: %s", err))
	}
}

func (r *PresetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
