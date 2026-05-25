package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/typesense/typesense-go/v4/typesense"
	"github.com/typesense/typesense-go/v4/typesense/api"
)

var _ resource.Resource = &CurationSetResource{}
var _ resource.ResourceWithImportState = &CurationSetResource{}

func NewCurationSetResource() resource.Resource {
	return &CurationSetResource{}
}

type CurationSetResource struct {
	client *typesense.Client
}

type CurationRuleModel struct {
	Query    types.String   `tfsdk:"query"`
	Match    types.String   `tfsdk:"match"`
	FilterBy types.String   `tfsdk:"filter_by"`
	Tags     []types.String `tfsdk:"tags"`
}

type CurationIncludeModel struct {
	Id       types.String `tfsdk:"id"`
	Position types.Int64  `tfsdk:"position"`
}

type CurationExcludeModel struct {
	Id types.String `tfsdk:"id"`
}

type CurationItemModel struct {
	Id                  types.String           `tfsdk:"id"`
	Rule                *CurationRuleModel     `tfsdk:"rule"`
	Includes            []CurationIncludeModel `tfsdk:"includes"`
	Excludes            []CurationExcludeModel `tfsdk:"excludes"`
	FilterBy            types.String           `tfsdk:"filter_by"`
	SortBy              types.String           `tfsdk:"sort_by"`
	ReplaceQuery        types.String           `tfsdk:"replace_query"`
	RemoveMatchedTokens types.Bool             `tfsdk:"remove_matched_tokens"`
	FilterCuratedHits   types.Bool             `tfsdk:"filter_curated_hits"`
	StopProcessing      types.Bool             `tfsdk:"stop_processing"`
	EffectiveFromTs     types.Int64            `tfsdk:"effective_from_ts"`
	EffectiveToTs       types.Int64            `tfsdk:"effective_to_ts"`
}

type CurationSetResourceModel struct {
	Id          types.String        `tfsdk:"id"`
	Name        types.String        `tfsdk:"name"`
	Description types.String        `tfsdk:"description"`
	Items       []CurationItemModel `tfsdk:"items"`
}

func (r *CurationSetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_curation_set"
}

func (r *CurationSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A global curation set (Typesense v30+). Collections opt in via the `curation_sets` collection attribute. Replaces the per-collection overrides API removed in v30.",
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
			"description": schema.StringAttribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"items": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":                    schema.StringAttribute{Required: true},
						"filter_by":             schema.StringAttribute{Optional: true},
						"sort_by":               schema.StringAttribute{Optional: true},
						"replace_query":         schema.StringAttribute{Optional: true},
						"remove_matched_tokens": schema.BoolAttribute{Optional: true, Computed: true},
						"filter_curated_hits":   schema.BoolAttribute{Optional: true, Computed: true},
						"stop_processing":       schema.BoolAttribute{Optional: true, Computed: true},
						"effective_from_ts":     schema.Int64Attribute{Optional: true},
						"effective_to_ts":       schema.Int64Attribute{Optional: true},
						"includes": schema.SetNestedAttribute{
							Optional: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id":       schema.StringAttribute{Required: true},
									"position": schema.Int64Attribute{Required: true},
								},
							},
						},
						"excludes": schema.SetNestedAttribute{
							Optional: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{Required: true},
								},
							},
						},
					},
					Blocks: map[string]schema.Block{
						"rule": schema.SingleNestedBlock{
							Attributes: map[string]schema.Attribute{
								"query": schema.StringAttribute{Optional: true},
								"match": schema.StringAttribute{
									Optional: true,
									Validators: []validator.String{
										stringvalidator.OneOf("exact", "contains"),
									},
								},
								"filter_by": schema.StringAttribute{Optional: true},
								"tags": schema.ListAttribute{
									ElementType: types.StringType,
									Optional:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *CurationSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func curationItemsToAPI(items []CurationItemModel) []api.CurationItemCreateSchema {
	out := make([]api.CurationItemCreateSchema, 0, len(items))
	for _, it := range items {
		id := it.Id.ValueString()
		apiItem := api.CurationItemCreateSchema{Id: &id}
		if it.Rule != nil {
			apiItem.Rule.Query = it.Rule.Query.ValueStringPointer()
			apiItem.Rule.FilterBy = it.Rule.FilterBy.ValueStringPointer()
			if !it.Rule.Match.IsNull() && it.Rule.Match.ValueString() != "" {
				m := api.CurationRuleMatch(it.Rule.Match.ValueString())
				apiItem.Rule.Match = &m
			}
			if len(it.Rule.Tags) > 0 {
				tags := convertTerraformArrayToStringArray(it.Rule.Tags)
				apiItem.Rule.Tags = &tags
			}
		}
		if len(it.Includes) > 0 {
			incs := make([]api.CurationInclude, 0, len(it.Includes))
			for _, i := range it.Includes {
				incs = append(incs, api.CurationInclude{
					Id:       i.Id.ValueString(),
					Position: int(i.Position.ValueInt64()),
				})
			}
			apiItem.Includes = &incs
		}
		if len(it.Excludes) > 0 {
			exs := make([]api.CurationExclude, 0, len(it.Excludes))
			for _, e := range it.Excludes {
				exs = append(exs, api.CurationExclude{Id: e.Id.ValueString()})
			}
			apiItem.Excludes = &exs
		}
		apiItem.FilterBy = it.FilterBy.ValueStringPointer()
		apiItem.SortBy = it.SortBy.ValueStringPointer()
		apiItem.ReplaceQuery = it.ReplaceQuery.ValueStringPointer()
		apiItem.RemoveMatchedTokens = it.RemoveMatchedTokens.ValueBoolPointer()
		apiItem.FilterCuratedHits = it.FilterCuratedHits.ValueBoolPointer()
		apiItem.StopProcessing = it.StopProcessing.ValueBoolPointer()
		if !it.EffectiveFromTs.IsNull() {
			v := int(it.EffectiveFromTs.ValueInt64())
			apiItem.EffectiveFromTs = &v
		}
		if !it.EffectiveToTs.IsNull() {
			v := int(it.EffectiveToTs.ValueInt64())
			apiItem.EffectiveToTs = &v
		}
		out = append(out, apiItem)
	}
	return out
}

func curationItemsFromAPI(items []api.CurationItemCreateSchema) []CurationItemModel {
	out := make([]CurationItemModel, 0, len(items))
	for _, it := range items {
		m := CurationItemModel{}
		if it.Id != nil {
			m.Id = types.StringValue(*it.Id)
		}
		m.Rule = &CurationRuleModel{
			Query:    types.StringPointerValue(it.Rule.Query),
			FilterBy: types.StringPointerValue(it.Rule.FilterBy),
		}
		if it.Rule.Match != nil {
			m.Rule.Match = types.StringValue(string(*it.Rule.Match))
		}
		if it.Rule.Tags != nil {
			m.Rule.Tags = convertStringArrayToTerraformArray(*it.Rule.Tags)
		}
		if it.Includes != nil {
			for _, inc := range *it.Includes {
				m.Includes = append(m.Includes, CurationIncludeModel{
					Id:       types.StringValue(inc.Id),
					Position: types.Int64Value(int64(inc.Position)),
				})
			}
		}
		if it.Excludes != nil {
			for _, ex := range *it.Excludes {
				m.Excludes = append(m.Excludes, CurationExcludeModel{
					Id: types.StringValue(ex.Id),
				})
			}
		}
		m.FilterBy = types.StringPointerValue(it.FilterBy)
		m.SortBy = types.StringPointerValue(it.SortBy)
		m.ReplaceQuery = types.StringPointerValue(it.ReplaceQuery)
		if it.RemoveMatchedTokens != nil {
			m.RemoveMatchedTokens = types.BoolValue(*it.RemoveMatchedTokens)
		}
		if it.FilterCuratedHits != nil {
			m.FilterCuratedHits = types.BoolValue(*it.FilterCuratedHits)
		}
		if it.StopProcessing != nil {
			m.StopProcessing = types.BoolValue(*it.StopProcessing)
		}
		if it.EffectiveFromTs != nil {
			m.EffectiveFromTs = types.Int64Value(int64(*it.EffectiveFromTs))
		}
		if it.EffectiveToTs != nil {
			m.EffectiveToTs = types.Int64Value(int64(*it.EffectiveToTs))
		}
		out = append(out, m)
	}
	return out
}

func (r *CurationSetResource) upsert(ctx context.Context, data *CurationSetResourceModel) error {
	body := &api.CurationSetCreateSchema{
		Description: data.Description.ValueStringPointer(),
		Items:       curationItemsToAPI(data.Items),
	}
	result, err := r.client.CurationSet(data.Name.ValueString()).Upsert(ctx, body)
	if err != nil {
		return err
	}
	// Populate computed fields with server defaults (stop_processing, filter_curated_hits, remove_matched_tokens).
	// Don't clobber description if the server omits it from the upsert response.
	if result.Description != nil {
		data.Description = types.StringValue(*result.Description)
	}
	data.Items = curationItemsFromAPI(result.Items)
	return nil
}

func (r *CurationSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CurationSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create curation set: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CurationSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CurationSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	set, err := r.client.CurationSet(data.Id.ValueString()).Retrieve(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve curation set: %s", err))
		return
	}
	data.Name = data.Id
	// Typesense v30.1 does not echo description back; preserve state value.
	if set.Description != nil {
		data.Description = types.StringValue(*set.Description)
	}
	data.Items = curationItemsFromAPI(set.Items)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CurationSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CurationSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update curation set: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CurationSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CurationSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.CurationSet(data.Id.ValueString()).Delete(ctx)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete curation set: %s", err))
	}
}

func (r *CurationSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
