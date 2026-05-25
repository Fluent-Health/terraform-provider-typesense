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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stopword.test", "stopwords.#", "4"),
				),
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
