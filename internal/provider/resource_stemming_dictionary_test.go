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
