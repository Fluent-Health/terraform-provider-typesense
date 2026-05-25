package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSynonymSetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymSetResourceConfig("test_synset_a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_synonym_set.test", "name", "test_synset_a"),
					resource.TestCheckResourceAttr("typesense_synonym_set.test", "items.#", "2"),
					resource.TestCheckResourceAttrSet("typesense_synonym_set.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_synonym_set.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccSynonymSetResourceConfigUpdated("test_synset_a"),
				Check:  resource.TestCheckResourceAttr("typesense_synonym_set.test", "items.#", "3"),
			},
		},
	})
}

func testAccSynonymSetResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "typesense_synonym_set" "test" {
  name = %[1]q

  items {
    id       = "clothing"
    synonyms = ["coat", "blazer", "jacket"]
  }

  items {
    id       = "sneaker_1way"
    root     = "sneaker"
    synonyms = ["shoe", "running shoe"]
  }
}
`, name)
}

func testAccSynonymSetResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "typesense_synonym_set" "test" {
  name = %[1]q

  items {
    id       = "clothing"
    synonyms = ["coat", "blazer", "jacket", "sweater"]
  }

  items {
    id       = "sneaker_1way"
    root     = "sneaker"
    synonyms = ["shoe", "running shoe", "trainer"]
  }

  items {
    id       = "colors"
    synonyms = ["red", "crimson", "scarlet"]
  }
}
`, name)
}
