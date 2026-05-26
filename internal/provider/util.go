package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"fluent-health-terraform-typesense/internal/typesense"
)

// configureClient pulls *typesense.Client out of req.ProviderData with the
// standard nil-and-type-assertion dance every resource's Configure needs.
// Returns nil when the provider hasn't been configured yet (early-apply); the
// caller should just return in that case. On a type mismatch it records a
// diagnostic and also returns nil.
func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *typesense.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*typesense.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *typesense.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return c
}

// convert []types.String to []string
func convertTerraformArrayToStringArray(array []types.String) []string {
	arrayString := make([]string, len(array))
	for i, item := range array {
		arrayString[i] = item.ValueString()
	}
	return arrayString
}

// convert []string to []types.String
func convertStringArrayToTerraformArray(array []string) []types.String {
	arrayString := make([]types.String, len(array))
	for i, item := range array {
		arrayString[i] = types.StringValue(item)
	}
	return arrayString
}

// parse string json to map[string]interface{}
func parseJsonStringToMap(jsonString string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// convert map[string]interface{} to string json
func parseMapToJsonString(data map[string]interface{}) (jsontypes.Normalized, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return jsontypes.NewNormalizedNull(), err
	}
	return jsontypes.NewNormalizedValue(string(jsonBytes)), nil
}

func splitCollectionRelatedId(input string) (string, string, error) {
	parts := strings.Split(input, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format, format should be <collection>.<resource>")
	}

	return parts[0], parts[1], nil
}

func createId(collection string, resource string) string {
	return fmt.Sprintf("%s.%s", collection, resource)
}
