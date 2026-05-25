resource "typesense_synonym_set" "clothing" {
  name = "clothing"

  items {
    id       = "outerwear"
    synonyms = ["coat", "blazer", "jacket"]
  }

  items {
    id       = "sneaker_1way"
    root     = "sneaker"
    synonyms = ["shoe", "running shoe"]
  }
}
