package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConversationModelResource(t *testing.T) {
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		t.Skip("OPENAI_API_KEY not set; Typesense v30 validates the API key at model-create time, so this test needs a real key.")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConversationModelConfig("conv_history", 16384, "You are a helpful assistant.", openAIKey),
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
				Config: testAccConversationModelConfig("conv_history", 32768, "Updated prompt.", openAIKey),
				Check:  resource.TestCheckResourceAttr("typesense_conversation_model.test", "max_bytes", "32768"),
			},
		},
	})
}

func testAccConversationModelConfig(historyCollection string, maxBytes int, systemPrompt, apiKey string) string {
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
    type = "int32"
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
  api_key            = %[4]q
  max_bytes          = %[2]d
  system_prompt      = %[3]q
}
`, historyCollection, maxBytes, systemPrompt, apiKey)
}
