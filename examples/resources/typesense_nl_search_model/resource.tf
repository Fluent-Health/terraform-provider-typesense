variable "openai_api_key" {
  type      = string
  sensitive = true
}

resource "typesense_nl_search_model" "rewriter" {
  model_name    = "openai/gpt-4o-mini"
  api_key       = var.openai_api_key
  system_prompt = "Rewrite the natural-language query into a Typesense filter_by clause."
}
