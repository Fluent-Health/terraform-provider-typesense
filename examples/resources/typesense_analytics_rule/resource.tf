resource "typesense_analytics_rule" "popular" {
  name       = "popular_queries_rule"
  type       = "popular_queries"
  collection = typesense_collection.products.name
  event_type = "search"

  params = {
    destination_collection = typesense_collection.popular_queries.name
    limit                  = 100
  }
}
