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

var _ resource.Resource = &SynonymSetResource{}
var _ resource.ResourceWithImportState = &SynonymSetResource{}

func NewSynonymSetResource() resource.Resource {
	return &SynonymSetResource{}
}

type SynonymSetResource struct {
	client *typesense.Client
}

type SynonymSetItemModel struct {
	Id             types.String   `tfsdk:"id"`
	Synonyms       []types.String `tfsdk:"synonyms"`
	Root           types.String   `tfsdk:"root"`
	Locale         types.String   `tfsdk:"locale"`
	SymbolsToIndex []types.String `tfsdk:"symbols_to_index"`
}

type SynonymSetResourceModel struct {
	Id    types.String          `tfsdk:"id"`
	Name  types.String          `tfsdk:"name"`
	Items []SynonymSetItemModel `tfsdk:"items"`
}

func (r *SynonymSetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synonym_set"
}

func (r *SynonymSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A global synonym set (Typesense v30+). Collections opt in via the `synonym_sets` collection attribute. Replaces the legacy per-collection `typesense_synonym` resource that was removed in v30.",
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
		},
		Blocks: map[string]schema.Block{
			"items": schema.SetNestedBlock{
				MarkdownDescription: "Synonym items in the set. Each item has its own id and synonym list.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "Unique identifier for the synonym item within this set.",
						},
						"synonyms": schema.SetAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: "Words that should be considered synonyms.",
						},
						"root": schema.StringAttribute{
							Optional:    true,
							Description: "For 1-way synonyms, the root word that the words in `synonyms` map to.",
						},
						"locale": schema.StringAttribute{
							Optional:    true,
							Description: "Locale for the synonym (leave blank to use the standard tokenizer).",
						},
						"symbols_to_index": schema.SetAttribute{
							ElementType: types.StringType,
							Optional:    true,
							Description: "Special characters to keep when indexing synonyms.",
						},
					},
				},
			},
		},
	}
}

func (r *SynonymSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func itemsToAPI(items []SynonymSetItemModel) []typesense.SynonymItem {
	out := make([]typesense.SynonymItem, 0, len(items))
	for _, item := range items {
		apiItem := typesense.SynonymItem{
			Id:       item.Id.ValueString(),
			Synonyms: convertTerraformArrayToStringArray(item.Synonyms),
		}
		if !item.Root.IsNull() && item.Root.ValueString() != "" {
			v := item.Root.ValueString()
			apiItem.Root = &v
		}
		if !item.Locale.IsNull() && item.Locale.ValueString() != "" {
			v := item.Locale.ValueString()
			apiItem.Locale = &v
		}
		if len(item.SymbolsToIndex) > 0 {
			syms := convertTerraformArrayToStringArray(item.SymbolsToIndex)
			apiItem.SymbolsToIndex = &syms
		}
		out = append(out, apiItem)
	}
	return out
}

func itemsFromAPI(items []typesense.SynonymItem) []SynonymSetItemModel {
	out := make([]SynonymSetItemModel, 0, len(items))
	for _, item := range items {
		m := SynonymSetItemModel{
			Id:       types.StringValue(item.Id),
			Synonyms: convertStringArrayToTerraformArray(item.Synonyms),
		}
		if item.Root != nil && *item.Root != "" {
			m.Root = types.StringValue(*item.Root)
		}
		if item.Locale != nil && *item.Locale != "" {
			m.Locale = types.StringValue(*item.Locale)
		}
		if item.SymbolsToIndex != nil {
			m.SymbolsToIndex = convertStringArrayToTerraformArray(*item.SymbolsToIndex)
		}
		out = append(out, m)
	}
	return out
}

func (r *SynonymSetResource) upsert(ctx context.Context, data *SynonymSetResourceModel) error {
	body := &typesense.SynonymSet{
		Items: itemsToAPI(data.Items),
	}
	_, err := r.client.UpsertSynonymSet(ctx, data.Name.ValueString(), body)
	return err
}

func (r *SynonymSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SynonymSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create synonym set: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SynonymSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	set, err := r.client.GetSynonymSet(ctx, data.Id.ValueString())
	if err != nil {
		if typesense.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve synonym set: %s", err))
		return
	}
	data.Name = data.Id
	data.Items = itemsFromAPI(set.Items)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SynonymSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update synonym set: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SynonymSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteSynonymSet(ctx, data.Id.ValueString())
	if err != nil && !typesense.IsNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete synonym set: %s", err))
	}
}

func (r *SynonymSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
