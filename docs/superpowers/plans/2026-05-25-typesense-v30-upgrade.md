# Typesense v30.1 Provider Upgrade — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring `terraform-provider-typesense` up to date with Typesense server v30.1 by exposing 6 missing resources and 5 missing collection field attributes that are already supported by the stable `typesense-go` v3.2.0 SDK, and verify each against a real Typesense container in CI.

**Architecture:** Each new Terraform resource lives in its own file `internal/provider/resource_<name>.go` and follows the existing pattern (`terraform-plugin-framework`, embedded `*typesense.Client`, framework Create/Read/Update/Delete/ImportState, `"Not Found"` → state removal). Acceptance tests run against a real Typesense Docker container — CI already does this; we bump the pin from 29.0 to 30.1.

**Tech Stack:**
- Provider: `terraform-plugin-framework` v1.6.0
- SDK: `github.com/typesense/typesense-go/v3` v3.2.0
- Server: `typesense/typesense:30.1` in CI
- Tests: `github.com/hashicorp/terraform-plugin-testing` v1.6.0
- Docs: `terraform-plugin-docs` v0.19.3 (via `make doc` / `go generate ./...`)

**SDK reference (entry points all on `*typesense.Client`):**
- `client.Presets()` / `client.Preset(name)` — search presets
- `client.Stopwords()` / `client.Stopword(id)` — stopword sets
- `client.Collection(c).Overrides()` / `client.Collection(c).Override(id)` — per-collection overrides
- `client.Stemming().Dictionaries()` / `client.Stemming().Dictionary(id)` — stemming dictionaries (no DELETE on server)
- `client.Analytics().Rules()` / `client.Analytics().Rule(name)` — analytics rules
- `client.Conversations().Models()` / `client.Conversations().Model(id)` — conversation models
- Pointer helpers: `github.com/typesense/typesense-go/v3/typesense/api/pointer`

**Important API constraints discovered during research:**
- Stemming dictionaries: server has no DELETE endpoint. `Delete` is a state-only no-op with a warning diagnostic.
- Analytics rule types in v3.2.0 SDK: only `popular_queries`, `nohits_queries`, `counter`. The spec mentioned `log` — it is not in v3.2.0; drop it.
- Conversation models: ID is server-assigned (Create takes `*string` Id, optional). Use Update endpoint for in-place changes.
- Overrides: `Match` enum constants live at `api.Exact` / `api.Contains` of type `api.SearchOverrideRuleMatch`.

---

## Task 0: Bump test infrastructure to Typesense 30.1

**Files:**
- Modify: `.github/workflows/build-and-test.yml`
- Modify: `.github/workflows/release.yml`
- Modify: `README.md`

This validates that the existing test suite still passes against v30.1 before we add new code.

- [ ] **Step 0.1: Bump CI service image in build-and-test.yml**

In `.github/workflows/build-and-test.yml`, change `typesense/typesense:29.0` to `typesense/typesense:30.1`.

```yaml
    services:
      typesense:
        image: typesense/typesense:30.1
        env:
          TYPESENSE_DATA_DIR: /tmp
          TYPESENSE_API_KEY: test-api-key
        ports:
          - 8108:8108
```

- [ ] **Step 0.2: Bump CI service image in release.yml**

Same change in `.github/workflows/release.yml`.

- [ ] **Step 0.3: Update README support claim and Docker snippet**

In `README.md`:
- Change `Supports v28.0+ version of Typesense.` to `Supports Typesense v30.1+`.
- Change `typesense/typesense:29.0` in the `docker run` snippet to `typesense/typesense:30.1`.

- [ ] **Step 0.4: Run existing acceptance suite locally against 30.1**

```bash
docker rm -f typesense-test 2>/dev/null
docker run -d --name typesense-test -p 8108:8108 -e TYPESENSE_DATA_DIR=/tmp -e TYPESENSE_API_KEY=test-api-key typesense/typesense:30.1
sleep 5
curl -f http://localhost:8108/health
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 go test ./internal/provider/... -v -timeout 30m
```

Expected: all existing tests PASS. Leave the container running for subsequent tasks.

- [ ] **Step 0.5: Commit**

```bash
git add .github/workflows/build-and-test.yml .github/workflows/release.yml README.md
git commit -m "$(cat <<'EOF'
chore(ci): bump Typesense test image and support claim to v30.1

Existing acceptance suite passes unchanged against typesense:30.1.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 1: Enhance `typesense_collection` field schema

**Files:**
- Modify: `internal/provider/resource_collection.go`
- Modify: `internal/provider/resource_collection_test.go`
- Modify: `examples/resources/typesense_collection/resource.tf`

Adds 5 per-field attributes that v3.2.0 SDK already supports: `reference`, `range_index`, `vec_dist`, field-level `symbols_to_index`, field-level `token_separators`.

- [ ] **Step 1.1: Add the new attributes to `CollectionResourceFieldModel`**

In `internal/provider/resource_collection.go`, locate `type CollectionResourceFieldModel struct` (around line 51). Add the new fields at the end of the struct (before the closing brace):

```go
type CollectionResourceFieldModel struct {
	Name           types.String               `tfsdk:"name"`
	Facet          types.Bool                 `tfsdk:"facet"`
	Index          types.Bool                 `tfsdk:"index"`
	Optional       types.Bool                 `tfsdk:"optional"`
	Sort           types.Bool                 `tfsdk:"sort"`
	Infix          types.Bool                 `tfsdk:"infix"`
	Type           types.String               `tfsdk:"type"`
	Stem           types.Bool                 `tfsdk:"stem"`
	StemDictionary types.String               `tfsdk:"stem_dictionary"`
	Locale         types.String               `tfsdk:"locale"`
	Store          types.Bool                 `tfsdk:"store"`
	NumDim         types.Int64                `tfsdk:"num_dim"`
	Embed          *CollectionFieldEmbedModel `tfsdk:"embed"`
	Reference      types.String               `tfsdk:"reference"`
	RangeIndex     types.Bool                 `tfsdk:"range_index"`
	VecDist        types.String               `tfsdk:"vec_dist"`
	SymbolsToIndex []types.String             `tfsdk:"symbols_to_index"`
	TokenSeparators []types.String            `tfsdk:"token_separators"`
}
```

- [ ] **Step 1.2: Add schema attributes for the new fields**

In `(r *CollectionResource) Schema(...)`, inside `"fields": schema.SetNestedBlock { NestedObject: schema.NestedBlockObject { Attributes: map[string]schema.Attribute{ ... }}}`, add these entries alongside `num_dim`:

```go
"reference": schema.StringAttribute{
    Optional:    true,
    Description: "Name of a field in another collection that should be linked to this collection so that it can be joined during query (e.g. \"users.id\").",
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
```

- [ ] **Step 1.3: Marshal new attributes in `filedModelToApiField`**

Locate `func filedModelToApiField(...)` (around line 718). Replace the body so it sets the new attributes too:

```go
func filedModelToApiField(field CollectionResourceFieldModel) api.Field {
	apiField := api.Field{
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
```

- [ ] **Step 1.4: Flatten new attributes in `flattenCollectionFields`**

Locate `func flattenCollectionFields(...)` (around line 489). Inside the loop, after the existing assignments, add:

```go
field.Reference = types.StringPointerValue(fieldResponse.Reference)
field.RangeIndex = boolPointerValueWithDefault(fieldResponse.RangeIndex, false)
field.VecDist = stringPointerValueWithDefault(fieldResponse.VecDist, "cosine")
field.SymbolsToIndex = nil
if fieldResponse.SymbolsToIndex != nil {
	for _, s := range *fieldResponse.SymbolsToIndex {
		field.SymbolsToIndex = append(field.SymbolsToIndex, types.StringValue(s))
	}
}
field.TokenSeparators = nil
if fieldResponse.TokenSeparators != nil {
	for _, s := range *fieldResponse.TokenSeparators {
		field.TokenSeparators = append(field.TokenSeparators, types.StringValue(s))
	}
}
```

- [ ] **Step 1.5: Populate ModifyPlan defaults for new bool fields**

In `(r *CollectionResource) ModifyPlan(...)` (around line 662), inside the `for i := range plan.Fields` loop, add:

```go
if plan.Fields[i].RangeIndex.IsUnknown() || plan.Fields[i].RangeIndex.IsNull() {
    plan.Fields[i].RangeIndex = types.BoolValue(false)
    modified = true
}
if plan.Fields[i].VecDist.IsUnknown() || plan.Fields[i].VecDist.IsNull() {
    plan.Fields[i].VecDist = types.StringValue("cosine")
    modified = true
}
```

- [ ] **Step 1.6: Add acceptance test for `reference` (JOIN)**

In `internal/provider/resource_collection_test.go`, add this test function and config helper:

```go
func TestAccCollectionResource_WithReference(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionResourceConfigReference("ref_users", "ref_orders"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.users", "name", "ref_users"),
					resource.TestCheckResourceAttr("typesense_collection.orders", "name", "ref_orders"),
					resource.TestCheckResourceAttr("typesense_collection.orders", "fields.#", "2"),
				),
			},
		},
	})
}

func testAccCollectionResourceConfigReference(usersName, ordersName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "users" {
  name = %[1]q

  fields {
    name = "id"
    type = "string"
  }

  fields {
    name = "email"
    type = "string"
  }
}

resource "typesense_collection" "orders" {
  name = %[2]q

  fields {
    name = "id"
    type = "string"
  }

  fields {
    name      = "user_id"
    type      = "string"
    reference = "${typesense_collection.users.name}.id"
  }
}
`, usersName, ordersName)
}
```

- [ ] **Step 1.7: Add acceptance test for `range_index` and `vec_dist`**

Append to `resource_collection_test.go`:

```go
func TestAccCollectionResource_WithVectorAndRange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionResourceConfigVectorRange("vec_range_coll"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", "vec_range_coll"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.#", "3"),
				),
			},
		},
	})
}

func testAccCollectionResourceConfigVectorRange(name string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  fields {
    name = "title"
    type = "string"
  }

  fields {
    name        = "rating"
    type        = "float"
    range_index = true
    sort        = true
  }

  fields {
    name     = "embedding"
    type     = "float[]"
    num_dim  = 4
    vec_dist = "ip"
  }
}
`, name)
}
```

- [ ] **Step 1.8: Add acceptance test for field-level symbols_to_index and token_separators**

Append to `resource_collection_test.go`:

```go
func TestAccCollectionResource_FieldLevelSymbolsAndTokens(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionResourceConfigFieldSymbols("field_symbols_coll"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", "field_symbols_coll"),
					resource.TestCheckResourceAttr("typesense_collection.test", "fields.#", "1"),
				),
			},
		},
	})
}

func testAccCollectionResourceConfigFieldSymbols(name string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  fields {
    name             = "title"
    type             = "string"
    symbols_to_index = ["+", "-"]
    token_separators = ["/"]
  }
}
`, name)
}
```

- [ ] **Step 1.9: Run tests**

```bash
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -run 'TestAccCollectionResource' -v -timeout 20m
```

Expected: all `TestAccCollectionResource*` tests PASS.

- [ ] **Step 1.10: Update collection example with reference**

In `examples/resources/typesense_collection/resource.tf`, append (or include in an existing example) a block showing a JOIN reference. Keep the existing top-level example intact and add a second collection that references it. Inspect the current file first; only append.

- [ ] **Step 1.11: Commit**

```bash
git add internal/provider/resource_collection.go internal/provider/resource_collection_test.go examples/resources/typesense_collection/resource.tf
git commit -m "$(cat <<'EOF'
feat(collection): expose reference, range_index, vec_dist, field-level symbols_to_index and token_separators

These attributes are already supported by typesense-go v3.2.0 against
Typesense v28+ servers but were previously not surfaced by the provider.
Acceptance tests cover JOIN references, vector field distance metrics,
range index on numeric fields, and field-level symbol/token overrides.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add `typesense_preset` resource

**Files:**
- Create: `internal/provider/resource_preset.go`
- Create: `internal/provider/resource_preset_test.go`
- Create: `examples/resources/typesense_preset/resource.tf`
- Create: `examples/resources/typesense_preset/import.sh`
- Modify: `internal/provider/provider.go` (register the resource)

Search presets save a search-parameter JSON blob under a name so clients can reference it later.

- [ ] **Step 2.1: Write the acceptance test first**

Create `internal/provider/resource_preset_test.go`:

```go
package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPresetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPresetResourceConfig("test_preset_a", `{"per_page":12}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_preset.test", "name", "test_preset_a"),
					resource.TestCheckResourceAttrSet("typesense_preset.test", "id"),
					resource.TestCheckResourceAttr("typesense_preset.test", "value", `{"per_page":12}`),
				),
			},
			{
				ResourceName:      "typesense_preset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPresetResourceConfig("test_preset_a", `{"per_page":50}`),
				Check: resource.TestCheckResourceAttr("typesense_preset.test", "value", `{"per_page":50}`),
			},
		},
	})
}

func testAccPresetResourceConfig(name, value string) string {
	return fmt.Sprintf(`
resource "typesense_preset" "test" {
  name  = %[1]q
  value = %[2]q
}
`, name, value)
}
```

- [ ] **Step 2.2: Run the test to verify it fails**

```bash
go test ./internal/provider/... -run TestAccPresetResource -v
```

Expected: FAIL with "undeclared name: NewPresetResource" or compile error referencing missing `typesense_preset` resource type.

- [ ] **Step 2.3: Implement the resource**

Create `internal/provider/resource_preset.go`:

```go
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

	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
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
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *typesense.Client, got: %T.", req.ProviderData),
		)
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
```

- [ ] **Step 2.4: Register the resource**

In `internal/provider/provider.go`, edit `(p *TypesenseProvider) Resources(...)`:

```go
func (p *TypesenseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCollectionResource,
		NewSynonymResource,
		NewDocumentResource,
		NewAliasResource,
		NewApiKeyResource,
		NewPresetResource,
	}
}
```

- [ ] **Step 2.5: Build and run tests**

```bash
go build ./...
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -run TestAccPresetResource -v -timeout 5m
```

Expected: PASS.

- [ ] **Step 2.6: Add example files**

Create `examples/resources/typesense_preset/resource.tf`:

```hcl
resource "typesense_preset" "high_per_page" {
  name  = "high_per_page"
  value = jsonencode({ per_page = 50 })
}
```

Create `examples/resources/typesense_preset/import.sh`:

```bash
terraform import typesense_preset.high_per_page high_per_page
```

- [ ] **Step 2.7: Commit**

```bash
git add internal/provider/resource_preset.go internal/provider/resource_preset_test.go internal/provider/provider.go examples/resources/typesense_preset
git commit -m "$(cat <<'EOF'
feat(preset): add typesense_preset resource

Wraps the Typesense Presets API. Stores a JSON blob of search
parameters under a name so clients can reference it via the
`preset` search parameter.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add `typesense_stopword` resource

**Files:**
- Create: `internal/provider/resource_stopword.go`
- Create: `internal/provider/resource_stopword_test.go`
- Create: `examples/resources/typesense_stopword/resource.tf`
- Create: `examples/resources/typesense_stopword/import.sh`
- Modify: `internal/provider/provider.go`

- [ ] **Step 3.1: Write the acceptance test**

Create `internal/provider/resource_stopword_test.go`:

```go
package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStopwordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStopwordResourceConfig("test_sw_set", "en", "the", "a", "an"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stopword.test", "name", "test_sw_set"),
					resource.TestCheckResourceAttr("typesense_stopword.test", "locale", "en"),
					resource.TestCheckResourceAttr("typesense_stopword.test", "stopwords.#", "3"),
					resource.TestCheckResourceAttrSet("typesense_stopword.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_stopword.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccStopwordResourceConfig("test_sw_set", "en", "the", "a", "an", "of"),
				Check:  resource.TestCheckResourceAttr("typesense_stopword.test", "stopwords.#", "4"),
			},
		},
	})
}

func testAccStopwordResourceConfig(name, locale string, words ...string) string {
	wordList := ""
	for i, w := range words {
		if i > 0 {
			wordList += ", "
		}
		wordList += fmt.Sprintf("%q", w)
	}
	return fmt.Sprintf(`
resource "typesense_stopword" "test" {
  name      = %[1]q
  locale    = %[2]q
  stopwords = [%[3]s]
}
`, name, locale, wordList)
}
```

- [ ] **Step 3.2: Implement the resource**

Create `internal/provider/resource_stopword.go`:

```go
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

	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
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
		MarkdownDescription: "A stopwords set is a list of common words removed from search queries that reference this set via the `stopwords` search parameter.",
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
			"stopwords": schema.ListAttribute{
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
```

- [ ] **Step 3.3: Register**

Add `NewStopwordResource` to the `Resources` slice in `internal/provider/provider.go`.

- [ ] **Step 3.4: Run tests**

```bash
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -run TestAccStopwordResource -v -timeout 5m
```

Expected: PASS.

- [ ] **Step 3.5: Add example files**

`examples/resources/typesense_stopword/resource.tf`:

```hcl
resource "typesense_stopword" "common_en" {
  name      = "common_en"
  locale    = "en"
  stopwords = ["the", "a", "an", "of", "and"]
}
```

`examples/resources/typesense_stopword/import.sh`:

```bash
terraform import typesense_stopword.common_en common_en
```

- [ ] **Step 3.6: Commit**

```bash
git add internal/provider/resource_stopword.go internal/provider/resource_stopword_test.go internal/provider/provider.go examples/resources/typesense_stopword
git commit -m "$(cat <<'EOF'
feat(stopword): add typesense_stopword resource

Wraps the Typesense Stopwords Sets API for managing named lists of
stopwords applied to searches via the `stopwords` search parameter.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add `typesense_stemming_dictionary` resource

**Files:**
- Create: `internal/provider/resource_stemming_dictionary.go`
- Create: `internal/provider/resource_stemming_dictionary_test.go`
- Create: `examples/resources/typesense_stemming_dictionary/resource.tf`
- Create: `examples/resources/typesense_stemming_dictionary/import.sh`
- Modify: `internal/provider/provider.go`

**Special handling:** Typesense has no DELETE endpoint for stemming dictionaries. `Delete` is a state-only operation that emits a warning diagnostic so users know the dictionary persists on the server.

- [ ] **Step 4.1: Write the acceptance test**

Create `internal/provider/resource_stemming_dictionary_test.go`:

```go
package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStemmingDictionaryResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStemmingDictionaryConfig("test_irregulars_a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "name", "test_irregulars_a"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.#", "2"),
					resource.TestCheckResourceAttrSet("typesense_stemming_dictionary.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_stemming_dictionary.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccStemmingDictionaryConfig(name string) string {
	return fmt.Sprintf(`
resource "typesense_stemming_dictionary" "test" {
  name = %[1]q

  words {
    word = "people"
    root = "person"
  }

  words {
    word = "children"
    root = "child"
  }
}
`, name)
}
```

- [ ] **Step 4.2: Implement the resource**

Create `internal/provider/resource_stemming_dictionary.go`:

```go
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

	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
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
		MarkdownDescription: "A custom stemming dictionary that maps surface word forms to a root form. **Note:** Typesense does not currently expose an HTTP DELETE for stemming dictionaries; `terraform destroy` removes the resource from state and emits a warning, but the dictionary remains on the server until it is overwritten or the server is wiped.",
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
			"words": schema.ListNestedBlock{
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
		fmt.Sprintf("Typesense does not currently expose a DELETE endpoint for stemming dictionaries. The dictionary %q has been removed from Terraform state but still exists on the server.", data.Id.ValueString()),
	)
}

func (r *StemmingDictionaryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
```

- [ ] **Step 4.3: Register**

Add `NewStemmingDictionaryResource` to `Resources` in `provider.go`.

- [ ] **Step 4.4: Run tests**

```bash
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -run TestAccStemmingDictionaryResource -v -timeout 5m
```

Expected: PASS (warning at destroy is expected; test framework does not fail on warnings).

- [ ] **Step 4.5: Add example files**

`examples/resources/typesense_stemming_dictionary/resource.tf`:

```hcl
resource "typesense_stemming_dictionary" "irregulars" {
  name = "irregulars_en"

  words {
    word = "people"
    root = "person"
  }

  words {
    word = "children"
    root = "child"
  }
}
```

`examples/resources/typesense_stemming_dictionary/import.sh`:

```bash
terraform import typesense_stemming_dictionary.irregulars irregulars_en
```

- [ ] **Step 4.6: Commit**

```bash
git add internal/provider/resource_stemming_dictionary.go internal/provider/resource_stemming_dictionary_test.go internal/provider/provider.go examples/resources/typesense_stemming_dictionary
git commit -m "$(cat <<'EOF'
feat(stemming): add typesense_stemming_dictionary resource

Wraps the Typesense Stemming Dictionaries API. Destroy is a state-only
operation because the server has no DELETE endpoint; the resource emits
a warning diagnostic so users are aware.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Add `typesense_analytics_rule` resource

**Files:**
- Create: `internal/provider/resource_analytics_rule.go`
- Create: `internal/provider/resource_analytics_rule_test.go`
- Create: `examples/resources/typesense_analytics_rule/resource.tf`
- Create: `examples/resources/typesense_analytics_rule/import.sh`
- Modify: `internal/provider/provider.go`

Analytics rules require Typesense to be started with the `--enable-search-analytics` flag, which is the default in the official Docker image. Rule types in v3.2.0 SDK are `popular_queries`, `nohits_queries`, `counter`.

- [ ] **Step 5.1: Write the acceptance test**

Create `internal/provider/resource_analytics_rule_test.go`:

```go
package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAnalyticsRuleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalyticsRuleConfig("test_analytics_src", "test_analytics_dst", "test_popular_rule"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "name", "test_popular_rule"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "type", "popular_queries"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "params.source.collections.#", "1"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "params.destination.collection", "test_analytics_dst"),
					resource.TestCheckResourceAttrSet("typesense_analytics_rule.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_analytics_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAnalyticsRuleConfig(srcCollection, dstCollection, ruleName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "src" {
  name = %[1]q

  fields {
    name = "title"
    type = "string"
  }
}

resource "typesense_collection" "dst" {
  name = %[2]q

  fields {
    name  = "q"
    type  = "string"
  }

  fields {
    name = "count"
    type = "int32"
    sort = true
  }

  default_sorting_field = "count"
}

resource "typesense_analytics_rule" "test" {
  name = %[3]q
  type = "popular_queries"

  params = {
    limit = 100
    source = {
      collections = [typesense_collection.src.name]
    }
    destination = {
      collection = typesense_collection.dst.name
    }
  }
}
`, srcCollection, dstCollection, ruleName)
}
```

- [ ] **Step 5.2: Implement the resource**

Create `internal/provider/resource_analytics_rule.go`:

```go
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

	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

var _ resource.Resource = &AnalyticsRuleResource{}
var _ resource.ResourceWithImportState = &AnalyticsRuleResource{}

func NewAnalyticsRuleResource() resource.Resource {
	return &AnalyticsRuleResource{}
}

type AnalyticsRuleResource struct {
	client *typesense.Client
}

type AnalyticsRuleSourceModel struct {
	Collections []types.String `tfsdk:"collections"`
}

type AnalyticsRuleDestinationModel struct {
	Collection   types.String `tfsdk:"collection"`
	CounterField types.String `tfsdk:"counter_field"`
}

type AnalyticsRuleParamsModel struct {
	Source      AnalyticsRuleSourceModel      `tfsdk:"source"`
	Destination AnalyticsRuleDestinationModel `tfsdk:"destination"`
	Limit       types.Int64                   `tfsdk:"limit"`
	ExpandQuery types.Bool                    `tfsdk:"expand_query"`
}

type AnalyticsRuleResourceModel struct {
	Id     types.String             `tfsdk:"id"`
	Name   types.String             `tfsdk:"name"`
	Type   types.String             `tfsdk:"type"`
	Params AnalyticsRuleParamsModel `tfsdk:"params"`
}

func (r *AnalyticsRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analytics_rule"
}

func (r *AnalyticsRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An analytics rule that aggregates search queries (popular_queries, nohits_queries) or generic counters from a source collection into a destination collection.",
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
				MarkdownDescription: "Rule type: `popular_queries`, `nohits_queries`, or `counter`.",
				Validators: []validator.String{
					stringvalidator.OneOf("popular_queries", "nohits_queries", "counter"),
				},
			},
			"params": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"limit": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Maximum number of aggregated entries to keep (for popular_queries / nohits_queries).",
					},
					"expand_query": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether to expand query into related queries when aggregating.",
					},
					"source": schema.SingleNestedAttribute{
						Required: true,
						Attributes: map[string]schema.Attribute{
							"collections": schema.ListAttribute{
								ElementType: types.StringType,
								Required:    true,
							},
						},
					},
					"destination": schema.SingleNestedAttribute{
						Required: true,
						Attributes: map[string]schema.Attribute{
							"collection": schema.StringAttribute{
								Required: true,
							},
							"counter_field": schema.StringAttribute{
								Optional:            true,
								MarkdownDescription: "Field in the destination collection that stores the counter (required for `counter` type).",
							},
						},
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

func (r *AnalyticsRuleResource) upsert(ctx context.Context, data *AnalyticsRuleResourceModel) error {
	params := api.AnalyticsRuleParameters{
		Source: api.AnalyticsRuleParametersSource{
			Collections: convertTerraformArrayToStringArray(data.Params.Source.Collections),
		},
		Destination: api.AnalyticsRuleParametersDestination{
			Collection: data.Params.Destination.Collection.ValueString(),
		},
	}
	if !data.Params.Destination.CounterField.IsNull() && data.Params.Destination.CounterField.ValueString() != "" {
		v := data.Params.Destination.CounterField.ValueString()
		params.Destination.CounterField = &v
	}
	if !data.Params.Limit.IsNull() {
		v := int(data.Params.Limit.ValueInt64())
		params.Limit = &v
	}
	if !data.Params.ExpandQuery.IsNull() {
		v := data.Params.ExpandQuery.ValueBool()
		params.ExpandQuery = &v
	}

	body := &api.AnalyticsRuleUpsertSchema{
		Type:   api.AnalyticsRuleUpsertSchemaType(data.Type.ValueString()),
		Params: params,
	}
	_, err := r.client.Analytics().Rules().Upsert(ctx, data.Name.ValueString(), body)
	return err
}

func (r *AnalyticsRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create analytics rule: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.Analytics().Rule(data.Id.ValueString()).Retrieve(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve analytics rule: %s", err))
		return
	}
	data.Name = types.StringValue(rule.Name)
	data.Type = types.StringValue(string(rule.Type))

	data.Params = AnalyticsRuleParamsModel{
		Source: AnalyticsRuleSourceModel{
			Collections: convertStringArrayToTerraformArray(rule.Params.Source.Collections),
		},
		Destination: AnalyticsRuleDestinationModel{
			Collection: types.StringValue(rule.Params.Destination.Collection),
		},
	}
	if rule.Params.Destination.CounterField != nil {
		data.Params.Destination.CounterField = types.StringValue(*rule.Params.Destination.CounterField)
	}
	if rule.Params.Limit != nil {
		data.Params.Limit = types.Int64Value(int64(*rule.Params.Limit))
	}
	if rule.Params.ExpandQuery != nil {
		data.Params.ExpandQuery = types.BoolValue(*rule.Params.ExpandQuery)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update analytics rule: %s", err))
		return
	}
	data.Id = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AnalyticsRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Analytics().Rule(data.Id.ValueString()).Delete(ctx)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete analytics rule: %s", err))
	}
}

func (r *AnalyticsRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
```

- [ ] **Step 5.3: Register**

Add `NewAnalyticsRuleResource` to `Resources` in `provider.go`.

- [ ] **Step 5.4: Run tests**

```bash
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -run TestAccAnalyticsRuleResource -v -timeout 5m
```

Expected: PASS. If Typesense returns an error mentioning "Analytics is not enabled", the test container must be started with `--enable-search-analytics=true`. Re-launch the test container with:

```bash
docker rm -f typesense-test
docker run -d --name typesense-test -p 8108:8108 \
  -e TYPESENSE_DATA_DIR=/tmp -e TYPESENSE_API_KEY=test-api-key \
  -e TYPESENSE_ENABLE_SEARCH_ANALYTICS=true \
  typesense/typesense:30.1
```

And update `.github/workflows/build-and-test.yml` and `release.yml` to add `TYPESENSE_ENABLE_SEARCH_ANALYTICS: true` under the `services.typesense.env` map. Commit that workflow change along with this task.

- [ ] **Step 5.5: Add example files**

`examples/resources/typesense_analytics_rule/resource.tf`:

```hcl
resource "typesense_analytics_rule" "popular" {
  name = "popular_queries_rule"
  type = "popular_queries"

  params = {
    limit = 100
    source = {
      collections = [typesense_collection.products.name]
    }
    destination = {
      collection = typesense_collection.popular_queries.name
    }
  }
}
```

`examples/resources/typesense_analytics_rule/import.sh`:

```bash
terraform import typesense_analytics_rule.popular popular_queries_rule
```

- [ ] **Step 5.6: Commit**

```bash
git add internal/provider/resource_analytics_rule.go internal/provider/resource_analytics_rule_test.go internal/provider/provider.go examples/resources/typesense_analytics_rule .github/workflows/build-and-test.yml .github/workflows/release.yml
git commit -m "$(cat <<'EOF'
feat(analytics): add typesense_analytics_rule resource

Wraps the Typesense Analytics Rules API for popular_queries,
nohits_queries, and counter rule types. CI service container now sets
TYPESENSE_ENABLE_SEARCH_ANALYTICS=true so acceptance tests can exercise
the API.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Add `typesense_override` resource

**Files:**
- Create: `internal/provider/resource_override.go`
- Create: `internal/provider/resource_override_test.go`
- Create: `examples/resources/typesense_override/resource.tf`
- Create: `examples/resources/typesense_override/import.sh`
- Modify: `internal/provider/provider.go`

Per-collection search overrides (a.k.a. curation rules). Uses compound ID `<collection>.<override_id>` via existing util helpers.

- [ ] **Step 6.1: Write the acceptance test**

Create `internal/provider/resource_override_test.go`:

```go
package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOverrideResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOverrideConfig("test_coll_override", "promote_apple", "apple"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_override.test", "collection_name", "test_coll_override"),
					resource.TestCheckResourceAttr("typesense_override.test", "name", "promote_apple"),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.query", "apple"),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.match", "exact"),
					resource.TestCheckResourceAttrSet("typesense_override.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_override.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "test_coll_override.promote_apple",
			},
		},
	})
}

func testAccOverrideConfig(collectionName, overrideName, query string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  fields {
    name = "name"
    type = "string"
  }
}

resource "typesense_override" "test" {
  collection_name = typesense_collection.test.name
  name            = %[2]q

  rule = {
    query = %[3]q
    match = "exact"
  }

  includes = [
    { id = "doc1", position = 1 }
  ]
}
`, collectionName, overrideName, query)
}
```

- [ ] **Step 6.2: Implement the resource**

Create `internal/provider/resource_override.go`:

```go
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

	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

var _ resource.Resource = &OverrideResource{}
var _ resource.ResourceWithImportState = &OverrideResource{}

func NewOverrideResource() resource.Resource {
	return &OverrideResource{}
}

type OverrideResource struct {
	client *typesense.Client
}

type OverrideRuleModel struct {
	Query    types.String   `tfsdk:"query"`
	Match    types.String   `tfsdk:"match"`
	FilterBy types.String   `tfsdk:"filter_by"`
	Tags     []types.String `tfsdk:"tags"`
}

type OverrideIncludeModel struct {
	Id       types.String `tfsdk:"id"`
	Position types.Int64  `tfsdk:"position"`
}

type OverrideExcludeModel struct {
	Id types.String `tfsdk:"id"`
}

type OverrideResourceModel struct {
	Id                  types.String           `tfsdk:"id"`
	Name                types.String           `tfsdk:"name"`
	CollectionName      types.String           `tfsdk:"collection_name"`
	Rule                OverrideRuleModel      `tfsdk:"rule"`
	Includes            []OverrideIncludeModel `tfsdk:"includes"`
	Excludes            []OverrideExcludeModel `tfsdk:"excludes"`
	FilterBy            types.String           `tfsdk:"filter_by"`
	SortBy              types.String           `tfsdk:"sort_by"`
	ReplaceQuery        types.String           `tfsdk:"replace_query"`
	RemoveMatchedTokens types.Bool             `tfsdk:"remove_matched_tokens"`
	FilterCuratedHits   types.Bool             `tfsdk:"filter_curated_hits"`
	StopProcessing      types.Bool             `tfsdk:"stop_processing"`
	EffectiveFromTs     types.Int64            `tfsdk:"effective_from_ts"`
	EffectiveToTs       types.Int64            `tfsdk:"effective_to_ts"`
}

func (r *OverrideResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_override"
}

func (r *OverrideResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Per-collection search override (curation rule): forces specific documents to appear at specific positions, exclude documents, or rewrite queries when a search matches a rule.",
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
			"collection_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rule": schema.SingleNestedAttribute{
				Required: true,
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
			"includes": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Required: true},
						"position": schema.Int64Attribute{Required: true},
					},
				},
			},
			"excludes": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{Required: true},
					},
				},
			},
			"filter_by":             schema.StringAttribute{Optional: true},
			"sort_by":               schema.StringAttribute{Optional: true},
			"replace_query":         schema.StringAttribute{Optional: true},
			"remove_matched_tokens": schema.BoolAttribute{Optional: true},
			"filter_curated_hits":   schema.BoolAttribute{Optional: true},
			"stop_processing":       schema.BoolAttribute{Optional: true},
			"effective_from_ts":     schema.Int64Attribute{Optional: true},
			"effective_to_ts":       schema.Int64Attribute{Optional: true},
		},
	}
}

func (r *OverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func overrideModelToSchema(d *OverrideResourceModel) *api.SearchOverrideSchema {
	out := &api.SearchOverrideSchema{
		Rule: api.SearchOverrideRule{
			Query:    d.Rule.Query.ValueStringPointer(),
			FilterBy: d.Rule.FilterBy.ValueStringPointer(),
		},
	}
	if !d.Rule.Match.IsNull() && d.Rule.Match.ValueString() != "" {
		m := api.SearchOverrideRuleMatch(d.Rule.Match.ValueString())
		out.Rule.Match = &m
	}
	if len(d.Rule.Tags) > 0 {
		tags := convertTerraformArrayToStringArray(d.Rule.Tags)
		out.Rule.Tags = &tags
	}
	if len(d.Includes) > 0 {
		incs := make([]api.SearchOverrideInclude, 0, len(d.Includes))
		for _, i := range d.Includes {
			incs = append(incs, api.SearchOverrideInclude{
				Id:       i.Id.ValueString(),
				Position: int(i.Position.ValueInt64()),
			})
		}
		out.Includes = &incs
	}
	if len(d.Excludes) > 0 {
		exs := make([]api.SearchOverrideExclude, 0, len(d.Excludes))
		for _, e := range d.Excludes {
			exs = append(exs, api.SearchOverrideExclude{Id: e.Id.ValueString()})
		}
		out.Excludes = &exs
	}
	out.FilterBy = d.FilterBy.ValueStringPointer()
	out.SortBy = d.SortBy.ValueStringPointer()
	out.ReplaceQuery = d.ReplaceQuery.ValueStringPointer()
	out.RemoveMatchedTokens = d.RemoveMatchedTokens.ValueBoolPointer()
	out.FilterCuratedHits = d.FilterCuratedHits.ValueBoolPointer()
	out.StopProcessing = d.StopProcessing.ValueBoolPointer()
	if !d.EffectiveFromTs.IsNull() {
		v := int(d.EffectiveFromTs.ValueInt64())
		out.EffectiveFromTs = &v
	}
	if !d.EffectiveToTs.IsNull() {
		v := int(d.EffectiveToTs.ValueInt64())
		out.EffectiveToTs = &v
	}
	return out
}

func flattenOverride(resp *api.SearchOverride, data *OverrideResourceModel) {
	data.Rule = OverrideRuleModel{
		Query:    types.StringPointerValue(resp.Rule.Query),
		FilterBy: types.StringPointerValue(resp.Rule.FilterBy),
	}
	if resp.Rule.Match != nil {
		data.Rule.Match = types.StringValue(string(*resp.Rule.Match))
	}
	if resp.Rule.Tags != nil {
		data.Rule.Tags = convertStringArrayToTerraformArray(*resp.Rule.Tags)
	}
	data.Includes = nil
	if resp.Includes != nil {
		for _, i := range *resp.Includes {
			data.Includes = append(data.Includes, OverrideIncludeModel{
				Id:       types.StringValue(i.Id),
				Position: types.Int64Value(int64(i.Position)),
			})
		}
	}
	data.Excludes = nil
	if resp.Excludes != nil {
		for _, e := range *resp.Excludes {
			data.Excludes = append(data.Excludes, OverrideExcludeModel{
				Id: types.StringValue(e.Id),
			})
		}
	}
	data.FilterBy = types.StringPointerValue(resp.FilterBy)
	data.SortBy = types.StringPointerValue(resp.SortBy)
	data.ReplaceQuery = types.StringPointerValue(resp.ReplaceQuery)
	if resp.RemoveMatchedTokens != nil {
		data.RemoveMatchedTokens = types.BoolValue(*resp.RemoveMatchedTokens)
	}
	if resp.FilterCuratedHits != nil {
		data.FilterCuratedHits = types.BoolValue(*resp.FilterCuratedHits)
	}
	if resp.StopProcessing != nil {
		data.StopProcessing = types.BoolValue(*resp.StopProcessing)
	}
	if resp.EffectiveFromTs != nil {
		data.EffectiveFromTs = types.Int64Value(int64(*resp.EffectiveFromTs))
	}
	if resp.EffectiveToTs != nil {
		data.EffectiveToTs = types.Int64Value(int64(*resp.EffectiveToTs))
	}
}

func (r *OverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OverrideResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Collection(data.CollectionName.ValueString()).
		Overrides().Upsert(ctx, data.Name.ValueString(), overrideModelToSchema(&data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create override: %s", err))
		return
	}
	data.Id = types.StringValue(createId(data.CollectionName.ValueString(), data.Name.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OverrideResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	collectionName, name, err := splitCollectionRelatedId(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}
	override, err := r.client.Collection(collectionName).Override(name).Retrieve(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve override: %s", err))
		return
	}
	data.Name = types.StringValue(name)
	data.CollectionName = types.StringValue(collectionName)
	flattenOverride(override, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OverrideResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	collectionName, name, err := splitCollectionRelatedId(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}
	_, err = r.client.Collection(collectionName).Overrides().Upsert(ctx, name, overrideModelToSchema(&data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update override: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OverrideResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	collectionName, name, err := splitCollectionRelatedId(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}
	_, err = r.client.Collection(collectionName).Override(name).Delete(ctx)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete override: %s", err))
	}
}

func (r *OverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	collectionName, name, err := splitCollectionRelatedId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'collection_name.override_name', got: %s", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection_name"), collectionName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}
```

- [ ] **Step 6.3: Register**

Add `NewOverrideResource` to `Resources` in `provider.go`.

- [ ] **Step 6.4: Run tests**

```bash
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -run TestAccOverrideResource -v -timeout 5m
```

Expected: PASS.

- [ ] **Step 6.5: Add example files**

`examples/resources/typesense_override/resource.tf`:

```hcl
resource "typesense_override" "promote_apple" {
  collection_name = typesense_collection.products.name
  name            = "promote_apple"

  rule = {
    query = "apple"
    match = "exact"
  }

  includes = [
    { id = "iphone-15", position = 1 }
  ]
}
```

`examples/resources/typesense_override/import.sh`:

```bash
terraform import typesense_override.promote_apple products.promote_apple
```

- [ ] **Step 6.6: Commit**

```bash
git add internal/provider/resource_override.go internal/provider/resource_override_test.go internal/provider/provider.go examples/resources/typesense_override
git commit -m "$(cat <<'EOF'
feat(override): add typesense_override resource

Per-collection search overrides (curation rules). Supports query and
filter-based rules, document includes/excludes with positions, replace
query, sort_by, time-bounded effectiveness, and stop_processing.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Add `typesense_conversation_model` resource

**Files:**
- Create: `internal/provider/resource_conversation_model.go`
- Create: `internal/provider/resource_conversation_model_test.go`
- Create: `examples/resources/typesense_conversation_model/resource.tf`
- Create: `examples/resources/typesense_conversation_model/import.sh`
- Modify: `internal/provider/provider.go`

The conversation model API uses Create (not Upsert), Update, Delete. ID is server-generated when not provided in Create. The test cannot validate against a real LLM, so the test sets `model_name = "openai/gpt-3.5-turbo"` and a dummy `api_key` — Typesense will store the model but real searches against it would fail. The acceptance test only validates CRUD round-trips.

- [ ] **Step 7.1: Write the acceptance test**

Create `internal/provider/resource_conversation_model_test.go`:

```go
package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConversationModelResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConversationModelConfig("conv_history", 16384, "You are a helpful assistant."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_conversation_model.test", "id"),
					resource.TestCheckResourceAttr("typesense_conversation_model.test", "model_name", "openai/gpt-3.5-turbo"),
					resource.TestCheckResourceAttr("typesense_conversation_model.test", "history_collection", "conv_history"),
					resource.TestCheckResourceAttr("typesense_conversation_model.test", "max_bytes", "16384"),
				),
			},
			{
				ResourceName:            "typesense_conversation_model.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
			{
				Config: testAccConversationModelConfig("conv_history", 32768, "Updated prompt."),
				Check:  resource.TestCheckResourceAttr("typesense_conversation_model.test", "max_bytes", "32768"),
			},
		},
	})
}

func testAccConversationModelConfig(historyCollection string, maxBytes int, systemPrompt string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "history" {
  name = %[1]q

  fields {
    name = "conversation_id"
    type = "string"
  }

  fields {
    name = "model_id"
    type = "string"
  }

  fields {
    name = "timestamp"
    type = "int64"
    sort = true
  }

  fields {
    name = "role"
    type = "string"
  }

  fields {
    name = "message"
    type = "string"
  }

  default_sorting_field = "timestamp"
}

resource "typesense_conversation_model" "test" {
  model_name         = "openai/gpt-3.5-turbo"
  history_collection = typesense_collection.history.name
  api_key            = "sk-test-not-real"
  max_bytes          = %[2]d
  system_prompt      = %[3]q
}
`, historyCollection, maxBytes, systemPrompt)
}
```

- [ ] **Step 7.2: Implement the resource**

Create `internal/provider/resource_conversation_model.go`:

```go
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

	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
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
	if !data.Ttl.IsNull() {
		v := int(data.Ttl.ValueInt64())
		body.Ttl = &v
	}

	model, err := r.client.Conversations().Models().Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create conversation model: %s", err))
		return
	}
	data.Id = types.StringValue(model.Id)
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
		if strings.Contains(err.Error(), "Not Found") {
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
	// Note: server does not return api_key on read; preserve state value.
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
	if !data.Ttl.IsNull() {
		v := int(data.Ttl.ValueInt64())
		body.Ttl = &v
	}
	_, err := r.client.Conversations().Model(data.Id.ValueString()).Update(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update conversation model: %s", err))
		return
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
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete conversation model: %s", err))
	}
}

func (r *ConversationModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
```

- [ ] **Step 7.3: Register**

Add `NewConversationModelResource` to `Resources` in `provider.go`.

- [ ] **Step 7.4: Run tests**

```bash
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -run TestAccConversationModelResource -v -timeout 5m
```

Expected: PASS. The conversation model is created server-side without actually calling the LLM (Typesense only contacts the LLM at search time).

- [ ] **Step 7.5: Add example files**

`examples/resources/typesense_conversation_model/resource.tf`:

```hcl
resource "typesense_conversation_model" "rag" {
  model_name         = "openai/gpt-4"
  history_collection = typesense_collection.conversations.name
  api_key            = var.openai_api_key
  max_bytes          = 16384
  system_prompt      = "Answer based on the provided documents."
}
```

`examples/resources/typesense_conversation_model/import.sh`:

```bash
terraform import typesense_conversation_model.rag <conversation-model-id>
```

- [ ] **Step 7.6: Commit**

```bash
git add internal/provider/resource_conversation_model.go internal/provider/resource_conversation_model_test.go internal/provider/provider.go examples/resources/typesense_conversation_model
git commit -m "$(cat <<'EOF'
feat(conversation): add typesense_conversation_model resource

Wraps the Typesense Conversation Models API. Supports OpenAI,
Cloudflare, and self-hosted vLLM models. api_key is sensitive and
preserved from state on Read (server does not echo it back).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Regenerate docs

**Files:**
- Generated: `docs/resources/preset.md`, `stopword.md`, `override.md`, `stemming_dictionary.md`, `analytics_rule.md`, `conversation_model.md`
- Modified: `docs/resources/collection.md` (re-generated to pick up new field attrs)

- [ ] **Step 8.1: Run doc generation**

```bash
make doc
```

This runs `go generate ./...` which formats examples and regenerates `docs/resources/*.md` from in-code descriptions plus the example/import files.

- [ ] **Step 8.2: Inspect changes**

```bash
git status docs/
git diff docs/resources/collection.md
```

Verify:
- 6 new files under `docs/resources/` (one per new resource).
- `collection.md` now lists `reference`, `range_index`, `vec_dist`, `symbols_to_index`, `token_separators` under the `fields` block.

- [ ] **Step 8.3: Run full acceptance suite one last time**

```bash
TF_ACC=1 TYPESENSE_API_KEY=test-api-key TYPESENSE_API_ADDRESS=http://localhost:8108 \
  go test ./internal/provider/... -v -timeout 60m
```

Expected: ALL tests PASS (existing 5 resources + 6 new resources + collection enhancements).

- [ ] **Step 8.4: Commit**

```bash
git add docs/
git commit -m "$(cat <<'EOF'
docs: regenerate provider docs for v30.1 upgrade

Adds docs for preset, stopword, override, stemming_dictionary,
analytics_rule, and conversation_model resources, and updates the
collection docs to reflect the new field attributes (reference,
range_index, vec_dist, field-level symbols_to_index and
token_separators).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 8.5: Tear down the test container**

```bash
docker rm -f typesense-test
```

---

## Self-Review Notes

- All 6 spec resources are covered by Tasks 2–7.
- Collection field enhancements covered by Task 1.
- Test image bump covered by Task 0.
- Docs regeneration covered by Task 8.
- Deferred items (NL search, global synonym_sets, global curation_sets, async_reference/cascade_delete) are explicitly out of scope and documented in the spec.
- `stemming_dictionary` Delete-is-a-no-op limitation is called out in both the resource's MarkdownDescription and a runtime warning diagnostic, plus noted in the task.
- Analytics rule needs `TYPESENSE_ENABLE_SEARCH_ANALYTICS=true` in CI; Task 5 includes that workflow change.
- Naming convention is consistent: snake_case Terraform attrs, PascalCase Go struct fields, all `_test.go` files mirror their resource files.
