package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAnalyticsRuleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalyticsRuleConfig("test_ana_src", "test_ana_popular", "test_popular_rule"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "name", "test_popular_rule"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "type", "popular_queries"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "collection", "test_ana_src"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "event_type", "search"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "params.destination_collection", "test_ana_popular"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "params.limit", "100"),
				),
			},
			{
				ResourceName:      "typesense_analytics_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAnalyticsRuleConfig(srcCollection, dstCollection, ruleName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "src" {
  name = %[1]q

  fields {
    name = "title"
    type = "string"
  }
}

resource "typesense_collection" "dst" {
  name = %[2]q

  fields {
    name = "q"
    type = "string"
  }

  fields {
    name = "count"
    type = "int32"
    sort = true
  }

  default_sorting_field = "count"
}

resource "typesense_analytics_rule" "test" {
  name       = %[3]q
  type       = "popular_queries"
  collection = typesense_collection.src.name
  event_type = "search"

  params = {
    destination_collection = typesense_collection.dst.name
    limit                  = 100
  }
}
`, srcCollection, dstCollection, ruleName)
}
