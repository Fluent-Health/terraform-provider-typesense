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
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/typesense/typesense-go/v4/typesense"
	"github.com/typesense/typesense-go/v4/typesense/api"
)

var _ resource.Resource = &StemmingDictionaryResource{}
var _ resource.ResourceWithImportState = &StemmingDictionaryResource{}

func NewStemmingDictionaryResource() resource.Resource {
	return &StemmingDictionaryResource{}
}

type StemmingDictionaryResource struct {
	client *typesense.Client
}

type StemmingDictionaryWordModel struct {
	Word types.String `tfsdk:"word"`
	Root types.String `tfsdk:"root"`
}

type StemmingDictionaryResourceModel struct {
	Id    types.String                  `tfsdk:"id"`
	Name  types.String                  `tfsdk:"name"`
	Words []StemmingDictionaryWordModel `tfsdk:"words"`
}

func (r *StemmingDictionaryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stemming_dictionary"
}

func (r *StemmingDictionaryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A custom stemming dictionary that maps surface word forms to a root form. **Note:** Typesense does not currently expose an HTTP DELETE for stemming dictionaries; `terraform destroy` removes the resource from state and emits a warning, but the dictionary remains on the server until it is overwritten or the server data is wiped.",
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
			"words": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"word": schema.StringAttribute{
							Required:    true,
							Description: "Word form to be stemmed.",
						},
						"root": schema.StringAttribute{
							Required:    true,
							Description: "Root form the word maps to.",
						},
					},
				},
			},
		},
	}
}

func (r *StemmingDictionaryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StemmingDictionaryResource) upsert(ctx context.Context, data *StemmingDictionaryResourceModel) error {
	words := make([]api.StemmingDictionaryWord, 0, len(data.Words))
	for _, w := range data.Words {
		words = append(words, api.StemmingDictionaryWord{
			Word: w.Word.ValueString(),
			Root: w.Root.ValueString(),
		})
	}
	_, err := r.client.Stemming().Dictionaries().Upsert(ctx, data.Name.ValueString(), words)
	return err
}

func (r *StemmingDictionaryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StemmingDictionaryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create stemming dictionary: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StemmingDictionaryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StemmingDictionaryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dict, err := r.client.Stemming().Dictionary(data.Id.ValueString()).Retrieve(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve stemming dictionary: %s", err))
		return
	}
	data.Name = types.StringValue(dict.Id)
	data.Words = nil
	for _, w := range dict.Words {
		data.Words = append(data.Words, StemmingDictionaryWordModel{
			Word: types.StringValue(w.Word),
			Root: types.StringValue(w.Root),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StemmingDictionaryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StemmingDictionaryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update stemming dictionary: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StemmingDictionaryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StemmingDictionaryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Warn(ctx, "Removing stemming dictionary "+data.Id.ValueString()+" from state. Typesense has no DELETE endpoint for stemming dictionaries; the dictionary remains on the server.")
	resp.Diagnostics.AddWarning(
		"Stemming dictionary not deleted on server",
		fmt.Sprintf("Typesense does not expose a DELETE endpoint for stemming dictionaries. The dictionary %q has been removed from Terraform state but still exists on the server.", data.Id.ValueString()),
	)
}

func (r *StemmingDictionaryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
