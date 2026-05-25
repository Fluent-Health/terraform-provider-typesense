resource "typesense_stopword" "common_en" {
  name      = "common_en"
  locale    = "en"
  stopwords = ["the", "a", "an", "of", "and"]
}
