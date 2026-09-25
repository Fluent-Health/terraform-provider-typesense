package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"fluent-health-terraform-typesense/internal/typesense"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &TypesenseProvider{}
)
var _ provider.ProviderWithFunctions = &TypesenseProvider{}

type TypesenseProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// TypesenseProviderModel is the provider implementation.
type TypesenseProviderModel struct {
	ApiKey              types.String `tfsdk:"api_key"`
	ApiAddress          types.String `tfsdk:"api_address"`
	RequestTimeout      types.String `tfsdk:"request_timeout"`
	SchemaChangeTimeout types.String `tfsdk:"schema_change_timeout"`
}

// parseDurationAttribute parses an optional Go-duration provider attribute.
// ok is false when the attribute is unset or invalid (invalid adds a diagnostic).
func parseDurationAttribute(v types.String, p path.Path, resp *provider.ConfigureResponse) (time.Duration, bool) {
	if v.IsNull() || v.IsUnknown() {
		return 0, false
	}
	d, err := time.ParseDuration(v.ValueString())
	if err != nil || d <= 0 {
		resp.Diagnostics.AddAttributeError(p, "Invalid duration",
			fmt.Sprintf("Expected a positive Go duration such as \"10m\", got %q.", v.ValueString()))
		return 0, false
	}
	return d, true
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TypesenseProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *TypesenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "typesense"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *TypesenseProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API Key to access the Typesense server. This can also be set via the `TYPESENSE_API_KEY` environment variable.",
			},
			"api_address": schema.StringAttribute{
				Optional:    true,
				Description: "URL of the Typesense server. This can also be set via the `TYPESENSE_API_ADDRESS` environment variable.",
			},
			"request_timeout": schema.StringAttribute{
				Optional:    true,
				Description: "Timeout for a single HTTP request to the Typesense server, as a Go duration (e.g. `10m`). Defaults to `5m`.",
			},
			"schema_change_timeout": schema.StringAttribute{
				Optional: true,
				Description: "How long to wait for a collection schema change to finish server-side after its PATCH request ended early " +
					"(gateway timeout, dropped connection) or was refused because another change was running, as a Go duration (e.g. `90m`). Defaults to `60m`.",
			},
		},
	}
}

// Configure prepares a typesense API client for data sources and resources.
func (p *TypesenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data TypesenseProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	api_key := os.Getenv("TYPESENSE_API_KEY")
	api_address := os.Getenv("TYPESENSE_API_ADDRESS")

	if !data.ApiKey.IsNull() {
		api_key = data.ApiKey.ValueString()
	}

	if !data.ApiAddress.IsNull() {
		api_address = data.ApiAddress.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if api_key == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Typesense API Key",
			"The provider cannot create the Typesense API client as there is a missing or empty value for the Typesense API key. "+
				"Set the api_key value in the configuration or use the TYPESENSE_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if api_address == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_address"),
			"Missing Typesense API Address",
			"The provider cannot create the Typesense API client as there is a missing or empty value for the Typesense API host. "+
				"Set the api_address value in the configuration or use the TYPESENSE_API_ADDRESS environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	var opts []typesense.Option
	if d, ok := parseDurationAttribute(data.RequestTimeout, path.Root("request_timeout"), resp); ok {
		opts = append(opts, typesense.WithRequestTimeout(d))
	}
	if d, ok := parseDurationAttribute(data.SchemaChangeTimeout, path.Root("schema_change_timeout"), resp); ok {
		opts = append(opts, typesense.WithSchemaChangeTimeout(d))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := typesense.NewClient(api_address, api_key, opts...)

	// Make the Typesense client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *TypesenseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCollectionResource,
		NewDocumentResource,
		NewAliasResource,
		NewApiKeyResource,
		NewPresetResource,
		NewStopwordResource,
		NewStemmingDictionaryResource,
		NewSynonymSetResource,
		NewCurationSetResource,
		NewAnalyticsRuleResource,
		NewConversationModelResource,
		NewNLSearchModelResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *TypesenseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Functions implements provider.ProviderWithFunctions.
func (p *TypesenseProvider) Functions(context.Context) []func() function.Function {
	return nil
}
