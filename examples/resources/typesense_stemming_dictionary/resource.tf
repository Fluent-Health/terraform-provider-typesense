resource "typesense_stemming_dictionary" "irregulars" {
  name = "irregulars_en"

  words {
    word = "people"
    root = "person"
  }

  words {
    word = "children"
    root = "child"
  }
}
