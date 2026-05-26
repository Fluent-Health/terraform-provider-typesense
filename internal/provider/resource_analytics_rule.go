package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"ronati-terraform-typesense/internal/typesense"
)

var _ resource.Resource = &AnalyticsRuleResource{}
var _ resource.ResourceWithImportState = &AnalyticsRuleResource{}

func NewAnalyticsRuleResource() resource.Resource {
	return &AnalyticsRuleResource{}
}

type AnalyticsRuleResource struct {
	client *typesense.Client
}

type AnalyticsRuleParamsModel struct {
	DestinationCollection types.String   `tfsdk:"destination_collection"`
	CounterField          types.String   `tfsdk:"counter_field"`
	Limit                 types.Int64    `tfsdk:"limit"`
	ExpandQuery           types.Bool     `tfsdk:"expand_query"`
	Weight                types.Int64    `tfsdk:"weight"`
	CaptureSearchRequests types.Bool     `tfsdk:"capture_search_requests"`
	MetaFields            []types.String `tfsdk:"meta_fields"`
}

type AnalyticsRuleResourceModel struct {
	Id         types.String              `tfsdk:"id"`
	Name       types.String              `tfsdk:"name"`
	Type       types.String              `tfsdk:"type"`
	Collection types.String              `tfsdk:"collection"`
	EventType  types.String              `tfsdk:"event_type"`
	RuleTag    types.String              `tfsdk:"rule_tag"`
	Params     *AnalyticsRuleParamsModel `tfsdk:"params"`
}

func (r *AnalyticsRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analytics_rule"
}

func (r *AnalyticsRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An analytics rule (Typesense v30+ shape). Aggregates events from a source collection into a destination collection.",
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
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Rule type: `popular_queries`, `nohits_queries`, `counter`, or `log`.",
				Validators: []validator.String{
					stringvalidator.OneOf("popular_queries", "nohits_queries", "counter", "log"),
				},
			},
			"collection": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source collection that emits the events to be aggregated.",
			},
			"event_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Event type to track (e.g. `search`, `click`, `conversion`, `visit`, `custom`).",
			},
			"rule_tag": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional free-form tag attached to the rule for grouping.",
			},
			"params": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"destination_collection": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Destination collection where aggregated rows are written.",
					},
					"counter_field": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "For `counter` rules, the field in the destination that stores the running count.",
					},
					"limit": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Maximum number of aggregated rows to keep.",
					},
					"expand_query": schema.BoolAttribute{
						Optional: true,
						Computed: true,
					},
					"weight": schema.Int64Attribute{
						Optional: true,
					},
					"capture_search_requests": schema.BoolAttribute{
						Optional: true,
						Computed: true,
					},
					"meta_fields": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *AnalyticsRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func paramsModelToAPI(p *AnalyticsRuleParamsModel) *typesense.AnalyticsRuleParams {
	if p == nil {
		return nil
	}
	out := &typesense.AnalyticsRuleParams{
		DestinationCollection: p.DestinationCollection.ValueStringPointer(),
		CounterField:          p.CounterField.ValueStringPointer(),
		ExpandQuery:           p.ExpandQuery.ValueBoolPointer(),
		CaptureSearchRequests: p.CaptureSearchRequests.ValueBoolPointer(),
	}
	if !p.Limit.IsNull() && !p.Limit.IsUnknown() {
		v := int(p.Limit.ValueInt64())
		out.Limit = &v
	}
	if !p.Weight.IsNull() && !p.Weight.IsUnknown() {
		v := int(p.Weight.ValueInt64())
		out.Weight = &v
	}
	if len(p.MetaFields) > 0 {
		mfs := convertTerraformArrayToStringArray(p.MetaFields)
		out.MetaFields = &mfs
	}
	return out
}

func paramsAPIToModel(p *typesense.AnalyticsRuleParams) *AnalyticsRuleParamsModel {
	if p == nil {
		return nil
	}
	m := &AnalyticsRuleParamsModel{
		DestinationCollection: types.StringPointerValue(p.DestinationCollection),
		CounterField:          types.StringPointerValue(p.CounterField),
	}
	if p.Limit != nil {
		m.Limit = types.Int64Value(int64(*p.Limit))
	}
	if p.ExpandQuery != nil {
		m.ExpandQuery = types.BoolValue(*p.ExpandQuery)
	}
	if p.Weight != nil {
		m.Weight = types.Int64Value(int64(*p.Weight))
	}
	if p.CaptureSearchRequests != nil {
		m.CaptureSearchRequests = types.BoolValue(*p.CaptureSearchRequests)
	}
	if p.MetaFields != nil {
		m.MetaFields = convertStringArrayToTerraformArray(*p.MetaFields)
	}
	return m
}

func (r *AnalyticsRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := typesense.AnalyticsRuleCreate{
		Name:       data.Name.ValueString(),
		Type:       data.Type.ValueString(),
		Collection: data.Collection.ValueString(),
		EventType:  data.EventType.ValueString(),
		Params:     paramsModelToAPI(data.Params),
	}
	if !data.RuleTag.IsNull() && data.RuleTag.ValueString() != "" {
		v := data.RuleTag.ValueString()
		body.RuleTag = &v
	}

	created, err := r.client.CreateAnalyticsRules(ctx, []typesense.AnalyticsRuleCreate{body})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create analytics rule: %s", err))
		return
	}
	if len(created) == 0 {
		resp.Diagnostics.AddError("Client Error", "Server accepted analytics rule but returned no resources")
		return
	}
	r.applyRuleToModel(&created[0], &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) applyRuleToModel(rule *typesense.AnalyticsRule, data *AnalyticsRuleResourceModel) {
	data.Id = types.StringValue(rule.Name)
	data.Name = types.StringValue(rule.Name)
	data.Type = types.StringValue(rule.Type)
	data.Collection = types.StringValue(rule.Collection)
	data.EventType = types.StringValue(rule.EventType)
	if rule.RuleTag != nil {
		data.RuleTag = types.StringValue(*rule.RuleTag)
	} else {
		data.RuleTag = types.StringValue("")
	}
	data.Params = paramsAPIToModel(rule.Params)
}

func (r *AnalyticsRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.GetAnalyticsRule(ctx, data.Id.ValueString())
	if err != nil {
		if typesense.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve analytics rule: %s", err))
		return
	}
	r.applyRuleToModel(rule, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := &typesense.AnalyticsRuleUpdate{
		Params: paramsModelToAPI(data.Params),
	}
	if !data.RuleTag.IsNull() && data.RuleTag.ValueString() != "" {
		v := data.RuleTag.ValueString()
		body.RuleTag = &v
	}
	rule, err := r.client.UpdateAnalyticsRule(ctx, data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update analytics rule: %s", err))
		return
	}
	r.applyRuleToModel(rule, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteAnalyticsRule(ctx, data.Id.ValueString())
	if err != nil && !typesense.IsNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete analytics rule: %s", err))
	}
}

func (r *AnalyticsRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
