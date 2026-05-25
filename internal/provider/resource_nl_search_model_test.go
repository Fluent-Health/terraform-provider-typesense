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
