resource "typesense_preset" "high_per_page" {
  name  = "high_per_page"
  value = jsonencode({ per_page = 50 })
}
