package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"fluent-health-terraform-typesense/internal/typesense"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CollectionResource{}
var _ resource.ResourceWithImportState = &CollectionResource{}
var _ resource.ResourceWithModifyPlan = &CollectionResource{}

func NewCollectionResource() resource.Resource {
	return &CollectionResource{}
}

type CollectionResource struct {
	client *typesense.Client
}

type CollectionResourceModel struct {
	Id                  types.String                   `tfsdk:"id"`
	Name                types.String                   `tfsdk:"name"`
	DefaultSortingField types.String                   `tfsdk:"default_sorting_field"`
	Fields              []CollectionResourceFieldModel `tfsdk:"fields"`
	EnableNestedFields  types.Bool                     `tfsdk:"enable_nested_fields"`
	SymbolsToIndex      []types.String                 `tfsdk:"symbols_to_index"`
	TokenSeparators     []types.String                 `tfsdk:"token_separators"`
	DeletionProtection  types.Bool                     `tfsdk:"deletion_protection"`
}

type CollectionResourceFieldModel struct {
	Name            types.String               `tfsdk:"name"`
	Facet           types.Bool                 `tfsdk:"facet"`
	Index           types.Bool                 `tfsdk:"index"`
	Optional        types.Bool                 `tfsdk:"optional"`
	Sort            types.Bool                 `tfsdk:"sort"`
	Infix           types.Bool                 `tfsdk:"infix"`
	Type            types.String               `tfsdk:"type"`
	Stem            types.Bool                 `tfsdk:"stem"`
	StemDictionary  types.String               `tfsdk:"stem_dictionary"`
	Locale          types.String               `tfsdk:"locale"`
	Store           types.Bool                 `tfsdk:"store"`
	NumDim          types.Int64                `tfsdk:"num_dim"`
	Reference       types.String               `tfsdk:"reference"`
	AsyncReference  types.Bool                 `tfsdk:"async_reference"`
	RangeIndex      types.Bool                 `tfsdk:"range_index"`
	VecDist         types.String               `tfsdk:"vec_dist"`
	SymbolsToIndex  []types.String             `tfsdk:"symbols_to_index"`
	TokenSeparators []types.String             `tfsdk:"token_separators"`
	Embed           *CollectionFieldEmbedModel `tfsdk:"embed"`
}

type CollectionFieldEmbedModel struct {
	From        []types.String                        `tfsdk:"from"`
	ModelConfig *CollectionFieldEmbedModelConfigModel `tfsdk:"model_config"`
}

type CollectionFieldEmbedModelConfigModel struct {
	ModelName      types.String                             `tfsdk:"model_name"`
	Url            types.String                             `tfsdk:"url"`
	AccessToken    types.String                             `tfsdk:"access_token"`
	ApiKey         types.String                             `tfsdk:"api_key"`
	ClientId       types.String                             `tfsdk:"client_id"`
	ClientSecret   types.String                             `tfsdk:"client_secret"`
	IndexingPrefix types.String                             `tfsdk:"indexing_prefix"`
	ProjectId      types.String                             `tfsdk:"project_id"`
	QueryPrefix    types.String                             `tfsdk:"query_prefix"`
	RefreshToken   types.String                             `tfsdk:"refresh_token"`
	Region         types.String                             `tfsdk:"region"`
	ServiceAccount *CollectionFieldEmbedServiceAccountModel `tfsdk:"service_account"`
}

type CollectionFieldEmbedServiceAccountModel struct {
	ClientEmail types.String `tfsdk:"client_email"`
	PrivateKey  types.String `tfsdk:"private_key"`
	TokenURI    types.String `tfsdk:"token_uri"`
}

func (r *CollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (r *CollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Group of related documents which are roughly equivalent to a table in a relational database. Terraform will still remove auto-created fields for collections with auto-type, so you need to manually update the collection schema to match generated fields. See the [Typesense API docs](https://typesense.org/docs/30.2/api/collections.html).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Id identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Collection name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_sorting_field": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Default sorting field",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enable_nested_fields": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Enable nested fields, must be enabled to use object/object[] types",
				Default:             booldefault.StaticBool(false),
			},
			"symbols_to_index": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of symbols to index",
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"token_separators": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of token separators",
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"deletion_protection": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether or not to allow Terraform to destroy the collection. Unless this field is set to false in Terraform state, a terraform destroy or terraform apply that would delete the collection will fail.",
				Default:             booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"fields": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"facet": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Facet field. Defaults to false.",
						},
						"index": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Index field. Defaults to true.",
						},
						"optional": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Optional field. Defaults to false.",
						},
						"sort": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Sort field. Defaults to false.",
						},
						"infix": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Infix field. Defaults to false.",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Field type.",
							Validators: []validator.String{
								stringvalidator.OneOf(
									"string",
									"int32",
									"int64",
									"float",
									"bool",
									"geopoint",
									"object",
									"string[]",
									"int32[]",
									"int64[]",
									"float[]",
									"bool[]",
									"geopoint[]",
									"object[]",
									"string*",
									"image",
									"auto",
								),
							},
						},
						"stem": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Enable stemming on field. Defaults to false.",
						},
						"stem_dictionary": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Custom stemming dictionary. Defaults to empty string.",
						},
						"locale": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Locale for language-specific tokenization. Defaults to empty string.",
						},
						"store": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Store field value on disk. Defaults to true.",
						},
						"num_dim": schema.Int64Attribute{
							Optional:    true,
							Computed:    true,
							Description: "Number of dimensions for vector fields (float[] type). Required for manual vector search fields; auto-computed by the server for auto-embedded fields (from the embedder's output dimension).",
							PlanModifiers: []planmodifier.Int64{
								// Auto-embedded fields: HCL leaves num_dim unset and the
								// server computes it from the embedder. Carry the server's
								// value across plans so the set-element hash stays stable
								// and a no-op apply after import stays no-op.
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"reference": schema.StringAttribute{
							Optional:    true,
							Description: "Name of a field in another collection that should be linked to this collection so that it can be joined during query (e.g. \"users.id\").",
						},
						"async_reference": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Allow documents to be indexed successfully even when the referenced document doesn't exist yet. Only meaningful when `reference` is set.",
						},
						"range_index": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Enables an index optimized for range filtering on numerical fields. Defaults to false.",
						},
						"vec_dist": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Distance metric for vector fields. One of `cosine` (default) or `ip` (inner product).",
							Validators: []validator.String{
								stringvalidator.OneOf("cosine", "ip"),
							},
						},
						"symbols_to_index": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
							Description: "Field-level list of symbols/special characters to index (overrides collection-level setting).",
						},
						"token_separators": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
							Description: "Field-level list of token separator characters (overrides collection-level setting).",
						},
					},
					Blocks: map[string]schema.Block{
						"embed": schema.SingleNestedBlock{
							Attributes: map[string]schema.Attribute{
								"from": schema.ListAttribute{
									ElementType: types.StringType,
									Optional:    true,
									Description: "Fields to generate the embedding from",
								},
							},
							Blocks: map[string]schema.Block{
								"model_config": schema.SingleNestedBlock{
									Attributes: map[string]schema.Attribute{
										"model_name": schema.StringAttribute{
											Optional:    true,
											Description: "Model name for embedding generation (e.g. ts/clip-vit-b-p32)",
										},
										"url": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "URL for remote embedding model",
										},
										"access_token": schema.StringAttribute{
											Optional:    true,
											Sensitive:   true,
											Description: "Access token for authentication. Write-only — Typesense never echoes this on Read, so the value is preserved from prior Terraform state.",
										},
										"api_key": schema.StringAttribute{
											Optional:    true,
											Sensitive:   true,
											Description: "API key for authentication. Write-only — Typesense never echoes this on Read, so the value is preserved from prior Terraform state.",
										},
										"client_id": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Client ID for OAuth",
										},
										"client_secret": schema.StringAttribute{
											Optional:    true,
											Sensitive:   true,
											Description: "Client secret for OAuth. Write-only — Typesense never echoes this on Read, so the value is preserved from prior Terraform state.",
										},
										"indexing_prefix": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Prefix added to text during indexing",
										},
										"project_id": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Project ID for cloud providers",
										},
										"query_prefix": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Prefix added to text during querying",
										},
										"refresh_token": schema.StringAttribute{
											Optional:    true,
											Sensitive:   true,
											Description: "Refresh token for OAuth. Write-only — Typesense never echoes this on Read, so the value is preserved from prior Terraform state.",
										},
										"region": schema.StringAttribute{
											Optional:    true,
											Description: "Region for GCP Vertex AI.",
										},
									},
									Blocks: map[string]schema.Block{
										"service_account": schema.SingleNestedBlock{
											Attributes: map[string]schema.Attribute{
												"client_email": schema.StringAttribute{
													Optional:    true,
													Description: "Service-account client_email (from the GCP credentials JSON).",
												},
												"private_key": schema.StringAttribute{
													Optional:    true,
													Sensitive:   true,
													Description: "Service-account private_key PEM (from the GCP credentials JSON).",
												},
												"token_uri": schema.StringAttribute{
													Optional:    true,
													Description: "OAuth token endpoint. Defaults to https://oauth2.googleapis.com/token if omitted.",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *CollectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *CollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CollectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Snapshot the plan's fields before we overwrite data.Fields from the API
	// response. Used to preserve sensitive embed credentials the server never
	// echoes (api_key, access_token, client_secret, refresh_token, and
	// service_account.private_key) — see flattenFieldEmbed.
	planFields := data.Fields

	schema := &typesense.CollectionCreateSchema{}
	schema.Name = data.Name.ValueString()
	schema.DefaultSortingField = data.DefaultSortingField.ValueStringPointer()
	schema.EnableNestedFields = data.EnableNestedFields.ValueBoolPointer()

	symbolsToIndex := []string{}
	for _, symbol := range data.SymbolsToIndex {
		symbolsToIndex = append(symbolsToIndex, symbol.ValueString())
	}
	schema.SymbolsToIndex = &symbolsToIndex

	tokensSeparators := []string{}
	for _, token := range data.TokenSeparators {
		tokensSeparators = append(tokensSeparators, token.ValueString())
	}
	schema.TokenSeparators = &tokensSeparators

	fields := []typesense.Field{}

	for _, field := range data.Fields {
		fields = append(fields, filedModelToApiField(field))
	}

	schema.Fields = fields
	collection, err := r.client.CreateCollection(ctx, schema)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create collection, got error: %s", err))
		return
	}

	data.Id = types.StringValue(collection.Name)
	data.Name = types.StringValue(collection.Name)

	if collection.DefaultSortingField != nil && *collection.DefaultSortingField != "" {
		data.DefaultSortingField = types.StringPointerValue(collection.DefaultSortingField)
	}

	data.EnableNestedFields = types.BoolPointerValue(collection.EnableNestedFields)
	data.Fields = flattenCollectionFields(collection.Fields, planFields)

	data.SymbolsToIndex = []types.String{}
	if collection.SymbolsToIndex != nil {
		for _, symbol := range *collection.SymbolsToIndex {
			data.SymbolsToIndex = append(data.SymbolsToIndex, types.StringValue(symbol))
		}
	}

	data.TokenSeparators = []types.String{}
	if collection.TokenSeparators != nil {
		for _, token := range *collection.TokenSeparators {
			data.TokenSeparators = append(data.TokenSeparators, types.StringValue(token))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CollectionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Snapshot prior state so flattenCollectionFields can preserve sensitive
	// embed credentials the server never echoes (api_key, access_token,
	// client_secret, refresh_token, service_account.private_key).
	priorFields := data.Fields

	id := data.Id.ValueString()

	collection, err := r.client.GetCollection(ctx, id)

	if err != nil {
		if typesense.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve collection, got error: %s", err))
		}

		return
	}

	tflog.Info(ctx, "###Got collection name:"+collection.Name)

	data.Id = types.StringValue(collection.Name)
	data.Name = types.StringValue(collection.Name)

	if collection.DefaultSortingField != nil && *collection.DefaultSortingField != "" {
		data.DefaultSortingField = types.StringPointerValue(collection.DefaultSortingField)
	}

	data.EnableNestedFields = types.BoolPointerValue(collection.EnableNestedFields)
	data.Fields = flattenCollectionFields(collection.Fields, priorFields)

	if collection.SymbolsToIndex != nil {
		data.SymbolsToIndex = []types.String{}
		if collection.SymbolsToIndex != nil {
			for _, symbol := range *collection.SymbolsToIndex {
				data.SymbolsToIndex = append(data.SymbolsToIndex, types.StringValue(symbol))
			}
		}
	}

	if collection.TokenSeparators != nil {
		data.TokenSeparators = []types.String{}
		if collection.TokenSeparators != nil {
			for _, token := range *collection.TokenSeparators {
				data.TokenSeparators = append(data.TokenSeparators, types.StringValue(token))
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func boolPointerValueWithDefault(ptr *bool, defaultVal bool) types.Bool {
	if ptr == nil {
		return types.BoolValue(defaultVal)
	}
	return types.BoolValue(*ptr)
}

func stringPointerValueWithDefault(ptr *string, defaultVal string) types.String {
	if ptr == nil {
		return types.StringValue(defaultVal)
	}
	return types.StringValue(*ptr)
}

func intPointerValue(ptr *int) types.Int64 {
	if ptr == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*ptr))
}

// flattenCollectionFields converts the Typesense API field list to the
// Terraform model. The optional `prior` slice is the previous Terraform model
// for the same collection (state in Read, plan in Create / Update); it is used
// to preserve write-only sensitive embed credentials that the Typesense API
// never echoes back. See flattenFieldEmbed for the specific fields preserved.
func flattenCollectionFields(fields []typesense.Field, prior []CollectionResourceFieldModel) []CollectionResourceFieldModel {
	if fields == nil {
		return make([]CollectionResourceFieldModel, 0)
	}

	priorByName := make(map[string]CollectionResourceFieldModel, len(prior))
	for _, f := range prior {
		priorByName[f.Name.ValueString()] = f
	}

	fis := make([]CollectionResourceFieldModel, len(fields))
	for i, fieldResponse := range fields {
		var field CollectionResourceFieldModel
		field.Name = types.StringValue(fieldResponse.Name)
		field.Facet = boolPointerValueWithDefault(fieldResponse.Facet, false)
		field.Index = boolPointerValueWithDefault(fieldResponse.Index, true)
		field.Optional = boolPointerValueWithDefault(fieldResponse.Optional, false)
		field.Sort = boolPointerValueWithDefault(fieldResponse.Sort, false)
		field.Infix = boolPointerValueWithDefault(fieldResponse.Infix, false)
		field.Type = types.StringValue(fieldResponse.Type)
		field.Stem = boolPointerValueWithDefault(fieldResponse.Stem, false)
		field.StemDictionary = stringPointerValueWithDefault(fieldResponse.StemDictionary, "")
		field.Locale = stringPointerValueWithDefault(fieldResponse.Locale, "")
		field.Store = boolPointerValueWithDefault(fieldResponse.Store, true)
		field.NumDim = intPointerValue(fieldResponse.NumDim)
		if fieldResponse.Reference != nil {
			field.Reference = types.StringValue(*fieldResponse.Reference)
		}
		if fieldResponse.AsyncReference != nil {
			field.AsyncReference = types.BoolValue(*fieldResponse.AsyncReference)
		}
		field.RangeIndex = boolPointerValueWithDefault(fieldResponse.RangeIndex, false)
		field.VecDist = stringPointerValueWithDefault(fieldResponse.VecDist, "cosine")
		if fieldResponse.SymbolsToIndex != nil {
			for _, s := range *fieldResponse.SymbolsToIndex {
				field.SymbolsToIndex = append(field.SymbolsToIndex, types.StringValue(s))
			}
		}
		if fieldResponse.TokenSeparators != nil {
			for _, s := range *fieldResponse.TokenSeparators {
				field.TokenSeparators = append(field.TokenSeparators, types.StringValue(s))
			}
		}
		if fieldResponse.Embed != nil {
			var priorEmbed *CollectionFieldEmbedModel
			if p, ok := priorByName[field.Name.ValueString()]; ok {
				priorEmbed = p.Embed
			}
			field.Embed = flattenFieldEmbed(fieldResponse.Embed, priorEmbed)
		}
		fis[i] = field
	}

	return fis
}

func (r *CollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CollectionResourceModel
	var state CollectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	stateItems := make(map[string]CollectionResourceFieldModel)

	for i := 0; i < len(state.Fields); i += 1 {
		stateItems[state.Fields[i].Name.ValueString()] = state.Fields[i]
	}

	schema := &typesense.CollectionUpdateSchema{}

	var drop = new(bool)
	*drop = true

	for _, field := range plan.Fields {
		// item not exists, need to create
		if _, ok := stateItems[field.Name.ValueString()]; !ok {
			schema.Fields = append(schema.Fields, filedModelToApiField(field))

			tflog.Info(ctx, "###Field will be created: "+field.Name.ValueString())

		} else if !fieldsEqual(stateItems[field.Name.ValueString()], field) {
			// item was changed, need to update

			schema.Fields = append(schema.Fields,
				typesense.Field{
					Drop: drop,
					Name: field.Name.ValueString(),
				},
				filedModelToApiField(field))
			tflog.Info(ctx, "###Field will be updated: "+field.Name.ValueString())

		} else {
			// item was not changed, do nothing
			tflog.Info(ctx, "###Field remaining the same: "+field.Name.ValueString())
		}

		// delete processed field from the state object
		delete(stateItems, field.Name.ValueString())
	}

	for _, field := range stateItems {
		schema.Fields = append(schema.Fields,
			typesense.Field{
				Drop: drop,
				Name: field.Name.ValueString(),
			})
		tflog.Info(ctx, "###Field will be deleted: "+field.Name.ValueString())
	}

	// Only call Typesense API if there are actual field changes
	if len(schema.Fields) > 0 {
		_, err := r.client.UpdateCollection(ctx, state.Id.ValueString(), schema)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update collection, got error: %s", err))
			return
		}
	}

	// Read back the updated collection to get all computed field attributes
	collection, err := r.client.GetCollection(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve updated collection, got error: %s", err))
		return
	}

	plan.Id = types.StringValue(collection.Name)
	plan.Name = types.StringValue(collection.Name)

	if collection.DefaultSortingField != nil && *collection.DefaultSortingField != "" {
		plan.DefaultSortingField = types.StringPointerValue(collection.DefaultSortingField)
	}

	plan.EnableNestedFields = types.BoolPointerValue(collection.EnableNestedFields)
	// Pass the plan's fields (not state's) as the prior reference so the
	// post-Update state reflects the user's just-applied sensitive values.
	plan.Fields = flattenCollectionFields(collection.Fields, plan.Fields)

	plan.SymbolsToIndex = []types.String{}
	if collection.SymbolsToIndex != nil {
		for _, symbol := range *collection.SymbolsToIndex {
			plan.SymbolsToIndex = append(plan.SymbolsToIndex, types.StringValue(symbol))
		}
	}

	plan.TokenSeparators = []types.String{}
	if collection.TokenSeparators != nil {
		for _, token := range *collection.TokenSeparators {
			plan.TokenSeparators = append(plan.TokenSeparators, types.StringValue(token))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CollectionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check deletion protection
	if data.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Cannot destroy collection",
			fmt.Sprintf("Collection %q has deletion_protection set to true. Set it to false before destroying.", data.Name.ValueString()),
		)
		return
	}

	tflog.Warn(ctx, "###Delete collection with id="+data.Id.ValueString())

	err := r.client.DeleteCollection(ctx, data.Id.ValueString())

	if err != nil {
		if typesense.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete collection, got error: %s", err))
		}

		return
	}

	data.Id = types.StringValue("")
}

func (r *CollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CollectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan CollectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	modified := false
	for i := range plan.Fields {
		if plan.Fields[i].Facet.IsUnknown() || plan.Fields[i].Facet.IsNull() {
			plan.Fields[i].Facet = types.BoolValue(false)
			modified = true
		}
		if plan.Fields[i].Index.IsUnknown() || plan.Fields[i].Index.IsNull() {
			plan.Fields[i].Index = types.BoolValue(true)
			modified = true
		}
		if plan.Fields[i].Optional.IsUnknown() || plan.Fields[i].Optional.IsNull() {
			plan.Fields[i].Optional = types.BoolValue(false)
			modified = true
		}
		if plan.Fields[i].Sort.IsUnknown() || plan.Fields[i].Sort.IsNull() {
			plan.Fields[i].Sort = types.BoolValue(false)
			modified = true
		}
		if plan.Fields[i].Infix.IsUnknown() || plan.Fields[i].Infix.IsNull() {
			plan.Fields[i].Infix = types.BoolValue(false)
			modified = true
		}
		if plan.Fields[i].Stem.IsUnknown() || plan.Fields[i].Stem.IsNull() {
			plan.Fields[i].Stem = types.BoolValue(false)
			modified = true
		}
		if plan.Fields[i].StemDictionary.IsUnknown() || plan.Fields[i].StemDictionary.IsNull() {
			plan.Fields[i].StemDictionary = types.StringValue("")
			modified = true
		}
		if plan.Fields[i].Locale.IsUnknown() || plan.Fields[i].Locale.IsNull() {
			plan.Fields[i].Locale = types.StringValue("")
			modified = true
		}
		if plan.Fields[i].Store.IsUnknown() || plan.Fields[i].Store.IsNull() {
			plan.Fields[i].Store = types.BoolValue(true)
			modified = true
		}
		if plan.Fields[i].RangeIndex.IsUnknown() || plan.Fields[i].RangeIndex.IsNull() {
			plan.Fields[i].RangeIndex = types.BoolValue(false)
			modified = true
		}
		if plan.Fields[i].VecDist.IsUnknown() || plan.Fields[i].VecDist.IsNull() {
			plan.Fields[i].VecDist = types.StringValue("cosine")
			modified = true
		}
		// async_reference: server returns false-by-default for fields that have
		// a reference; for non-reference fields the server omits it. Keep plan
		// and state aligned so we don't get "provider produced inconsistent
		// result after apply".
		hasReference := !plan.Fields[i].Reference.IsNull() && !plan.Fields[i].Reference.IsUnknown() && plan.Fields[i].Reference.ValueString() != ""
		if hasReference {
			if plan.Fields[i].AsyncReference.IsUnknown() || plan.Fields[i].AsyncReference.IsNull() {
				plan.Fields[i].AsyncReference = types.BoolValue(false)
				modified = true
			}
		} else if plan.Fields[i].AsyncReference.IsUnknown() {
			plan.Fields[i].AsyncReference = types.BoolNull()
			modified = true
		}
	}

	if modified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func filedModelToApiField(field CollectionResourceFieldModel) typesense.Field {
	apiField := typesense.Field{
		Name:           field.Name.ValueString(),
		Facet:          field.Facet.ValueBoolPointer(),
		Index:          field.Index.ValueBoolPointer(),
		Optional:       field.Optional.ValueBoolPointer(),
		Sort:           field.Sort.ValueBoolPointer(),
		Infix:          field.Infix.ValueBoolPointer(),
		Type:           field.Type.ValueString(),
		Stem:           field.Stem.ValueBoolPointer(),
		StemDictionary: field.StemDictionary.ValueStringPointer(),
		Locale:         field.Locale.ValueStringPointer(),
		Store:          field.Store.ValueBoolPointer(),
		Reference:      field.Reference.ValueStringPointer(),
		AsyncReference: field.AsyncReference.ValueBoolPointer(),
		RangeIndex:     field.RangeIndex.ValueBoolPointer(),
		VecDist:        field.VecDist.ValueStringPointer(),
	}

	if !field.NumDim.IsNull() && !field.NumDim.IsUnknown() {
		numDim := int(field.NumDim.ValueInt64())
		apiField.NumDim = &numDim
	}

	if len(field.SymbolsToIndex) > 0 {
		syms := make([]string, 0, len(field.SymbolsToIndex))
		for _, s := range field.SymbolsToIndex {
			if !s.IsNull() && !s.IsUnknown() {
				syms = append(syms, s.ValueString())
			}
		}
		apiField.SymbolsToIndex = &syms
	}

	if len(field.TokenSeparators) > 0 {
		seps := make([]string, 0, len(field.TokenSeparators))
		for _, s := range field.TokenSeparators {
			if !s.IsNull() && !s.IsUnknown() {
				seps = append(seps, s.ValueString())
			}
		}
		apiField.TokenSeparators = &seps
	}

	apiField.Embed = fieldEmbedModelToAPI(field.Embed)

	return apiField
}

func fieldEmbedModelToAPI(embed *CollectionFieldEmbedModel) *typesense.FieldEmbed {
	if embed == nil {
		return nil
	}

	embedAPI := &typesense.FieldEmbed{}

	if embed.From != nil {
		from := make([]string, 0, len(embed.From))
		for _, f := range embed.From {
			if !f.IsNull() && !f.IsUnknown() {
				from = append(from, f.ValueString())
			}
		}
		embedAPI.From = from
	}

	if embed.ModelConfig != nil {
		embedAPI.ModelConfig.ModelName = embed.ModelConfig.ModelName.ValueString()
		embedAPI.ModelConfig.Url = embed.ModelConfig.Url.ValueStringPointer()
		embedAPI.ModelConfig.AccessToken = embed.ModelConfig.AccessToken.ValueStringPointer()
		embedAPI.ModelConfig.ApiKey = embed.ModelConfig.ApiKey.ValueStringPointer()
		embedAPI.ModelConfig.ClientId = embed.ModelConfig.ClientId.ValueStringPointer()
		embedAPI.ModelConfig.ClientSecret = embed.ModelConfig.ClientSecret.ValueStringPointer()
		embedAPI.ModelConfig.IndexingPrefix = embed.ModelConfig.IndexingPrefix.ValueStringPointer()
		embedAPI.ModelConfig.ProjectId = embed.ModelConfig.ProjectId.ValueStringPointer()
		embedAPI.ModelConfig.QueryPrefix = embed.ModelConfig.QueryPrefix.ValueStringPointer()
		embedAPI.ModelConfig.RefreshToken = embed.ModelConfig.RefreshToken.ValueStringPointer()
		embedAPI.ModelConfig.Region = embed.ModelConfig.Region.ValueStringPointer()
		if sa := embed.ModelConfig.ServiceAccount; sa != nil &&
			(!sa.ClientEmail.IsNull() || !sa.PrivateKey.IsNull() || !sa.TokenURI.IsNull()) {
			embedAPI.ModelConfig.ServiceAccount = &typesense.GCPServiceAccount{
				ClientEmail: sa.ClientEmail.ValueString(),
				PrivateKey:  sa.PrivateKey.ValueString(),
				TokenURI:    sa.TokenURI.ValueStringPointer(),
			}
		}
	}

	return embedAPI
}

// flattenFieldEmbed converts the Typesense API embed shape to the Terraform
// model. `prior` is the previous Terraform value for this same embed block
// (state on Read, plan on Create/Update). It is used to preserve write-only
// sensitive credentials:
//
//   - model_config.{access_token, api_key, client_secret, refresh_token}
//   - model_config.service_account.private_key
//
// The Typesense server never echoes these on a subsequent GET, so without
// this preservation, a Read after Create silently blanks them in Terraform
// state. That creates a phantom diff on the next plan, and because
// Typesense's collection PATCH endpoint only supports drop+add for field
// changes, applying it would re-embed every document in the collection.
//
// This mirrors the comment already in applyNLSchemaToModel
// (resource_nl_search_model.go) for the same reason on /nl_search_models.
func flattenFieldEmbed(embed *typesense.FieldEmbed, prior *CollectionFieldEmbedModel) *CollectionFieldEmbedModel {
	if embed == nil {
		return nil
	}

	res := &CollectionFieldEmbedModel{}

	if embed.From != nil {
		from := make([]types.String, 0, len(embed.From))
		for _, f := range embed.From {
			from = append(from, types.StringValue(f))
		}
		res.From = from
	}

	res.ModelConfig = &CollectionFieldEmbedModelConfigModel{
		ModelName:      types.StringValue(embed.ModelConfig.ModelName),
		Url:            types.StringPointerValue(embed.ModelConfig.Url),
		ClientId:       types.StringPointerValue(embed.ModelConfig.ClientId),
		IndexingPrefix: types.StringPointerValue(embed.ModelConfig.IndexingPrefix),
		ProjectId:      types.StringPointerValue(embed.ModelConfig.ProjectId),
		QueryPrefix:    types.StringPointerValue(embed.ModelConfig.QueryPrefix),
		Region:         types.StringPointerValue(embed.ModelConfig.Region),
		// access_token, api_key, client_secret, refresh_token are sensitive
		// and not echoed by the server — preserved below from `prior`.
	}
	if sa := embed.ModelConfig.ServiceAccount; sa != nil {
		res.ModelConfig.ServiceAccount = &CollectionFieldEmbedServiceAccountModel{
			ClientEmail: types.StringValue(sa.ClientEmail),
			TokenURI:    types.StringPointerValue(sa.TokenURI),
			// private_key is sensitive and not echoed — preserved below from `prior`.
		}
	}

	if prior != nil && prior.ModelConfig != nil {
		res.ModelConfig.AccessToken = prior.ModelConfig.AccessToken
		res.ModelConfig.ApiKey = prior.ModelConfig.ApiKey
		res.ModelConfig.ClientSecret = prior.ModelConfig.ClientSecret
		res.ModelConfig.RefreshToken = prior.ModelConfig.RefreshToken

		// Server may mask credential-like fields on GET (project_id,
		// client_id, service_account.client_email come back as
		// "fh-de*****" / "***********" / "searc***********"). If prior
		// holds a real (non-null, non-unknown, non-masked) value, keep
		// it — otherwise the post-Update state would differ from plan
		// and Terraform aborts with "Provider produced inconsistent
		// result after apply". Mirrors setPreservingPriorReal in
		// resource_nl_search_model.go.
		preserveFromPriorIfServerMasked(&res.ModelConfig.ProjectId, prior.ModelConfig.ProjectId)
		preserveFromPriorIfServerMasked(&res.ModelConfig.ClientId, prior.ModelConfig.ClientId)

		if prior.ModelConfig.ServiceAccount != nil {
			if res.ModelConfig.ServiceAccount == nil {
				// Server omitted the service_account block entirely; restore it
				// from prior so we don't lose the user's credentials in state.
				res.ModelConfig.ServiceAccount = prior.ModelConfig.ServiceAccount
			} else {
				res.ModelConfig.ServiceAccount.PrivateKey = prior.ModelConfig.ServiceAccount.PrivateKey
				preserveFromPriorIfServerMasked(&res.ModelConfig.ServiceAccount.ClientEmail, prior.ModelConfig.ServiceAccount.ClientEmail)
			}
		}
	}

	return res
}

// preserveFromPriorIfServerMasked replaces *d with priorVal when *d (the
// freshly-flattened server value) matches Typesense's masking pattern
// and priorVal carries a real (non-null, non-unknown, non-masked) value.
// The opposite-direction sibling of setPreservingPriorReal in
// resource_nl_search_model.go — both bridge the same server-side
// masking, but flatten populates the destination from the server
// upfront and needs an "override if masked" check, while the NL
// resource fills the destination from the server pointer and needs
// "skip overwrite if masked".
func preserveFromPriorIfServerMasked(d *types.String, priorVal types.String) {
	if isServerRedacted(*d) && !priorVal.IsNull() && !priorVal.IsUnknown() && !isServerRedacted(priorVal) {
		*d = priorVal
	}
}

// fieldsEqual decides whether two Terraform field models are equivalent for
// the purpose of triggering a Typesense PATCH drop+add. A drop+add re-embeds
// every document in the collection, so we want to avoid it unless something
// actually changed on the server side.
//
// Asymmetric absorption normalizes four classes of state-vs-plan mismatch
// that aren't user-driven changes:
//
//  1. Write-only sensitive embed credentials (access_token, api_key,
//     client_secret, refresh_token, service_account.private_key, and the
//     whole service_account block): the Typesense API never echoes these
//     on GET, so post-import state has them as null while plan has the
//     user's real values. Treat as equal — the server already has them.
//
//  2. Server-computed num_dim on auto-embedded float[] fields: Typesense
//     computes num_dim from the embedder's output dimension and returns
//     it on GET. HCL doesn't (and shouldn't) set num_dim on auto-embedded
//     fields, so post-import state has a concrete value while plan stays
//     null. Treat as equal — the server keeps computing it.
//
//  3. Server-redacted credential-like embed fields (model_config.client_id,
//     model_config.project_id, model_config.service_account.client_email):
//     Typesense 30.x masks these on GET as a defensive measure, returning
//     values like "fh-de*****" or "***********" instead of the real
//     string. The provider's flatten faithfully copies the masked value
//     into state; plan carries the user's real value from HCL (when HCL
//     sets it) or Unknown (when HCL leaves the Optional+Computed slot
//     empty). Treat state-masked vs plan-non-null/Unknown as equal — the
//     server already holds the real value. Masking detection keys off
//     five or more consecutive asterisks (Typesense's shortest observed
//     redaction).
//
//  4. Server-echoed-or-omitted Optional+Computed embed.model_config
//     fields (client_id, indexing_prefix, project_id, query_prefix,
//     url): Typesense's GET response for these is inconsistent across
//     instances — some echo "", some echo masked strings (covered by
//     case 3), some omit the key entirely (state ends up null). HCL
//     rarely sets them, so plan resolves to Unknown (Optional+Computed
//     without UseStateForUnknown). Treat any plan-Unknown as equal to
//     whatever state holds — the framework would have resolved Unknown
//     to the server's value after apply anyway, and that's what state
//     already reflects. Doing this in the equivalence layer instead of
//     via UseStateForUnknown at the schema layer avoids writing masked
//     credentials back to the server on a real, unrelated Update.
//
// The asymmetries are deliberately one-directional. For sensitive fields,
// state-null vs plan-non-null is the post-import gap (suppress); but
// state-non-null vs plan-different-non-null is a real rotation (do not
// suppress — let drop+add fire so the new value reaches the server). For
// num_dim, state-non-null vs plan-null is the auto-embed import path
// (suppress); but state-non-null vs plan-different-non-null is a manual
// vector field's dim change (do not suppress). For server-redacted fields,
// state-masked vs plan-non-null/Unknown is the round-trip-through-server
// case (suppress); but state-unmasked vs plan-different-non-null is a
// genuine config change (do not suppress). For server-echoed
// Optional+Computed fields, plan-Unknown is the HCL-leaves-it-to-
// the-server case (suppress regardless of state); but plan-non-null-
// and-different is a user-driven set (do not suppress).
func fieldsEqual(state, plan CollectionResourceFieldModel) bool {
	return reflect.DeepEqual(absorbPostImportSensitive(state, plan), plan)
}

// isServerRedacted reports whether s looks like Typesense's "redacted on
// read" pattern (five or more consecutive asterisks). Used by
// absorbPostImportSensitive to identify state values that the server
// masked on GET so they can be treated as matching the user's real plan
// value.
func isServerRedacted(s types.String) bool {
	if s.IsNull() || s.IsUnknown() {
		return false
	}
	return strings.Contains(s.ValueString(), "*****")
}

// absorbPostImportSensitive returns a copy of `state` with post-import
// asymmetries normalized against `plan`. See fieldsEqual for the list of
// fields covered and the direction of each absorption. Used only to
// neutralize the post-import gap; the actual server-side state of those
// fields stays opaque to Terraform.
func absorbPostImportSensitive(state, plan CollectionResourceFieldModel) CollectionResourceFieldModel {
	if state.Embed == nil || state.Embed.ModelConfig == nil ||
		plan.Embed == nil || plan.Embed.ModelConfig == nil {
		return state
	}

	embedCopy := *state.Embed
	cfgCopy := *state.Embed.ModelConfig
	embedCopy.ModelConfig = &cfgCopy

	fillIfStateNull := func(s *types.String, p types.String) {
		if s.IsNull() && !p.IsNull() {
			*s = p
		}
	}
	fillIfStateNull(&cfgCopy.AccessToken, plan.Embed.ModelConfig.AccessToken)
	fillIfStateNull(&cfgCopy.ApiKey, plan.Embed.ModelConfig.ApiKey)
	fillIfStateNull(&cfgCopy.ClientSecret, plan.Embed.ModelConfig.ClientSecret)
	fillIfStateNull(&cfgCopy.RefreshToken, plan.Embed.ModelConfig.RefreshToken)

	// Server-redacted credential-like fields on the model_config. State
	// has the masked value, plan has the real one — treat as equal.
	fillIfStateRedacted := func(s *types.String, p types.String) {
		if isServerRedacted(*s) && !p.IsNull() {
			*s = p
		}
	}
	fillIfStateRedacted(&cfgCopy.ClientId, plan.Embed.ModelConfig.ClientId)
	fillIfStateRedacted(&cfgCopy.ProjectId, plan.Embed.ModelConfig.ProjectId)

	// Server-echoed-or-omitted Optional+Computed model_config fields.
	// Typesense's GET response for these is inconsistent: some
	// instances echo "" (observed on dev), others omit the key entirely
	// (observed on test). State picks up "" or null respectively. HCL
	// rarely sets these, so the framework resolves plan to Unknown
	// (Optional+Computed without UseStateForUnknown). Without this
	// absorb, both state-"" vs plan-Unknown AND state-null vs
	// plan-Unknown would trip reflect.DeepEqual and fire drop+add even
	// after the masking absorb runs.
	//
	// We treat any plan-Unknown as equal to whatever state holds — the
	// framework would have resolved Unknown to the server's value after
	// apply anyway, and that's what state already reflects. Doing this
	// in the equivalence layer instead of via UseStateForUnknown at the
	// schema layer avoids writing masked credentials back to the
	// server on a real, unrelated Update.
	fillIfPlanUnknown := func(s *types.String, p types.String) {
		if p.IsUnknown() {
			*s = p
		}
	}
	fillIfPlanUnknown(&cfgCopy.ClientId, plan.Embed.ModelConfig.ClientId)
	fillIfPlanUnknown(&cfgCopy.IndexingPrefix, plan.Embed.ModelConfig.IndexingPrefix)
	fillIfPlanUnknown(&cfgCopy.ProjectId, plan.Embed.ModelConfig.ProjectId)
	fillIfPlanUnknown(&cfgCopy.QueryPrefix, plan.Embed.ModelConfig.QueryPrefix)
	fillIfPlanUnknown(&cfgCopy.Url, plan.Embed.ModelConfig.Url)

	switch {
	case cfgCopy.ServiceAccount == nil && plan.Embed.ModelConfig.ServiceAccount != nil:
		// Whole service_account block missing in state (server didn't echo it
		// after import). Treat as if state matched plan; the user's HCL is the
		// only source of truth for these credentials anyway.
		cfgCopy.ServiceAccount = plan.Embed.ModelConfig.ServiceAccount
	case cfgCopy.ServiceAccount != nil && plan.Embed.ModelConfig.ServiceAccount != nil:
		saCopy := *cfgCopy.ServiceAccount
		cfgCopy.ServiceAccount = &saCopy
		fillIfStateNull(&saCopy.PrivateKey, plan.Embed.ModelConfig.ServiceAccount.PrivateKey)
		fillIfStateRedacted(&saCopy.ClientEmail, plan.Embed.ModelConfig.ServiceAccount.ClientEmail)
	}

	result := state
	result.Embed = &embedCopy

	// Server-computed num_dim. We only reach here when both sides have an
	// embed block (early return above), so this is the auto-embed path.
	// State carries the server-computed dim, plan has null — let plan win.
	// If the user actually set num_dim in HCL (manual override on an auto-
	// embedded field), plan is non-null and this branch is skipped, so a
	// real change still propagates through reflect.DeepEqual.
	if !state.NumDim.IsNull() && plan.NumDim.IsNull() {
		result.NumDim = plan.NumDim
	}

	return result
}
