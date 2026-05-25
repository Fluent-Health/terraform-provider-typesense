variable "openai_api_key" {
  type      = string
  sensitive = true
}

resource "typesense_conversation_model" "rag" {
  model_name         = "openai/gpt-4"
  history_collection = typesense_collection.conversations.name
  api_key            = var.openai_api_key
  max_bytes          = 16384
  system_prompt      = "Answer based on the provided documents."
}
