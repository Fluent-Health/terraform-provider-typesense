resource "typesense_curation_set" "promotions" {
  name = "promotions"

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
    id = "redirect_smartphone"

    rule {
      query = "smartphone"
      match = "contains"
    }

    replace_query = "phone"
  }
}
