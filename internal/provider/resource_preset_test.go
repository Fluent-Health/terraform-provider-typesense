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
				Check:  resource.TestCheckResourceAttr("typesense_preset.test", "value", `{"per_page":50}`),
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
