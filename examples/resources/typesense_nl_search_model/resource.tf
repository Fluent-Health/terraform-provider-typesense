variable "openai_api_key" {
  type      = string
  sensitive = true
}

resource "typesense_nl_search_model" "rewriter" {
  model_name    = "openai/gpt-4o-mini"
  api_key       = var.openai_api_key
  system_prompt = "Rewrite the natural-language query into a Typesense filter_by clause."
}

# GCP Vertex AI via service-account credentials — the recommended auth path for
# managed Gemini models (no refresh-token rotation to babysit). Equivalent to
# what you'd otherwise do via a raw POST /nl_search_models call.
resource "typesense_nl_search_model" "gemini" {
  model_name  = "gcp/gemini-2.5-flash"
  project_id  = "my-gcp-project"
  region      = "us-central1"
  max_bytes   = 16000
  temperature = 0.0

  service_account {
    client_email = "vertex-nl@my-gcp-project.iam.gserviceaccount.com"
    private_key  = file("${path.module}/vertex-sa-private-key.pem")
    # token_uri defaults to https://oauth2.googleapis.com/token
  }
}
