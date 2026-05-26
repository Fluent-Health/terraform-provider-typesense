package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNLSearchModelResource(t *testing.T) {
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		t.Skip("OPENAI_API_KEY not set; Typesense v29+ validates the API key at NL-model-create time, so this test needs a real key.")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNLSearchModelConfig(openAIKey, "You are a search-query rewriter."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_nl_search_model.test", "id"),
					resource.TestCheckResourceAttr("typesense_nl_search_model.test", "model_name", "openai/gpt-4o-mini"),
				),
			},
			{
				ResourceName:            "typesense_nl_search_model.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
		},
	})
}

func testAccNLSearchModelConfig(apiKey, systemPrompt string) string {
	return fmt.Sprintf(`
resource "typesense_nl_search_model" "test" {
  model_name    = "openai/gpt-4o-mini"
  api_key       = %[1]q
  system_prompt = %[2]q
}
`, apiKey, systemPrompt)
}

// TestAccNLSearchModelResource_ServiceAccount verifies the service_account
// block on /nl_search_models. Skipped unless GCP_SA_CLIENT_EMAIL,
// GCP_SA_PRIVATE_KEY, GCP_PROJECT_ID, and GCP_REGION are set, because the
// Typesense server mints a real Vertex access token at create time.
func TestAccNLSearchModelResource_ServiceAccount(t *testing.T) {
	clientEmail := os.Getenv("GCP_SA_CLIENT_EMAIL")
	privateKey := os.Getenv("GCP_SA_PRIVATE_KEY")
	projectID := os.Getenv("GCP_PROJECT_ID")
	region := os.Getenv("GCP_REGION")
	if clientEmail == "" || privateKey == "" || projectID == "" || region == "" {
		t.Skip("GCP_SA_CLIENT_EMAIL/GCP_SA_PRIVATE_KEY/GCP_PROJECT_ID/GCP_REGION not set; Typesense validates the SA credential against Vertex at create time, so this test needs real values.")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNLSearchModelServiceAccountConfig(projectID, region, clientEmail, privateKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_nl_search_model.test", "id"),
					resource.TestCheckResourceAttr("typesense_nl_search_model.test", "model_name", "gcp/gemini-2.5-flash"),
					resource.TestCheckResourceAttr("typesense_nl_search_model.test", "service_account.client_email", clientEmail),
				),
			},
			{
				ResourceName:            "typesense_nl_search_model.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"service_account.private_key"},
			},
		},
	})
}

func testAccNLSearchModelServiceAccountConfig(projectID, region, clientEmail, privateKey string) string {
	return fmt.Sprintf(`
resource "typesense_nl_search_model" "test" {
  model_name  = "gcp/gemini-2.5-flash"
  project_id  = %[1]q
  region      = %[2]q
  max_bytes   = 16000
  temperature = 0.0

  service_account {
    client_email = %[3]q
    private_key  = %[4]q
  }
}
`, projectID, region, clientEmail, privateKey)
}
