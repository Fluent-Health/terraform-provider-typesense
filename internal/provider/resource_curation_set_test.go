package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCurationSetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCurationSetConfig("test_curation_a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_curation_set.test", "name", "test_curation_a"),
					resource.TestCheckResourceAttr("typesense_curation_set.test", "items.#", "2"),
					resource.TestCheckResourceAttrSet("typesense_curation_set.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_curation_set.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCurationSetConfig(name string) string {
	return fmt.Sprintf(`
resource "typesense_curation_set" "test" {
  name = %[1]q

  items {
    id = "promote_apple"

    rule {
      query = "apple"
      match = "exact"
    }

    includes = [
      { id = "iphone-15", position = 1 }
    ]
  }

  items {
    id = "redirect_old_term"

    rule {
      query = "smartphone"
      match = "contains"
    }

    replace_query = "phone"
  }
}
`, name)
}
