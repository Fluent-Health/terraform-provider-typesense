---
layout: ""
page_title: "Provider: Typesense"
description: |-
  Manage Typesense collections, documents, synonyms, curation, analytics rules, and GCP Vertex AI-backed embedders / NL search models declaratively.
---

# Typesense Provider

A Terraform provider for [Typesense](https://typesense.org), an open-source typo-tolerant search engine. Use it to declaratively manage collections, indexed documents, API keys, synonyms and curation sets, analytics rules, and conversational / natural-language search models — including the GCP Vertex AI service-account auth path used by managed embedders.

The provider talks directly to the Typesense REST API (no `typesense-go` SDK dependency), so server-supported fields show up here without waiting on SDK regenerations.

## Quick start

```terraform
terraform {
  required_providers {
    typesense = {
      source  = "Fluent-Health/typesense"
      version = "~> 2.0"
    }
  }
}

provider "typesense" {
  api_address = "https://typesense.example.com"
  api_key     = var.typesense_admin_key # or TYPESENSE_API_KEY env var
}

resource "typesense_collection" "movies" {
  name = "movies"

  fields {
    name = "title"
    type = "string"
  }

  fields {
    name  = "year"
    type  = "int32"
    facet = true
    sort  = true
  }

  default_sorting_field = "year"
}
```

## Authentication

The provider needs two values:

| Argument | Environment variable | Description |
|---|---|---|
| `api_key` | `TYPESENSE_API_KEY` | Admin API key for the Typesense cluster. Treat as a secret. |
| `api_address` | `TYPESENSE_API_ADDRESS` | Full base URL of the Typesense server (e.g. `https://typesense.example.com`). |

Either inline or via env vars — env vars take precedence in CI/CD pipelines so you don't have to thread secrets through Terraform variables.

## Long-running schema changes

Changing a field on a large collection (especially re-adding an auto-embedded field) can take far longer than a load balancer or proxy will hold the request open. Typesense keeps running the alter after the connection is gone, so the provider does not treat that as a failure: when a collection `PATCH` ends in a 504/408/502, a client timeout or a dropped connection, it polls `GET /operations/schema_changes` until the alter is done, checks that the live schema has the planned field names, types and embed blocks, and records the planned values in state. When a `PATCH` is refused with 422 "Another collection update operation is in progress", it waits for that alter, re-diffs against the live schema and sends what is still needed once.

`schema_change_timeout` (default `60m`) bounds the wait and `request_timeout` (default `5m`) bounds a single HTTP request. The post-alter check is structural: write-only embed credentials are never returned by the server, so a credential-only change is taken as applied once the alter completes.

## Resources

### Collections & data
- [`typesense_collection`](./resources/collection.md) — schema, fields, embed/vector configuration
- [`typesense_document`](./resources/document.md) — individual indexed documents
- [`typesense_alias`](./resources/alias.md) — virtual collection-name pointers

### Access
- [`typesense_api_key`](./resources/api_key.md) — scoped admin/search keys

### Search tuning
- [`typesense_preset`](./resources/preset.md) — named search-parameter bundles
- [`typesense_stopword`](./resources/stopword.md) — stopwords sets
- [`typesense_stemming_dictionary`](./resources/stemming_dictionary.md) — custom stemming
- [`typesense_synonym_set`](./resources/synonym_set.md) — global synonym sets (v30+)
- [`typesense_curation_set`](./resources/curation_set.md) — global curation sets (v30+)

### Analytics & AI
- [`typesense_analytics_rule`](./resources/analytics_rule.md) — popular / nohits / counter rules (v30+ shape)
- [`typesense_conversation_model`](./resources/conversation_model.md) — RAG-style conversational search
- [`typesense_nl_search_model`](./resources/nl_search_model.md) — natural-language → structured-query translation (v29+)

## GCP Vertex AI auth (service account)

Both `typesense_collection.fields.embed.model_config` and `typesense_nl_search_model` accept a `service_account` block — the recommended auth path for managed Vertex AI models since it doesn't require refresh-token rotation.

```terraform
resource "typesense_nl_search_model" "gemini" {
  model_name  = "gcp/gemini-2.5-flash"
  project_id  = "my-gcp-project"
  region      = "us-central1"
  max_bytes   = 16000
  temperature = 0.0

  service_account {
    client_email = "vertex-nl@my-gcp-project.iam.gserviceaccount.com"
    private_key  = file("${path.module}/vertex-sa.pem")
  }
}
```

## Versioning

| This provider | Typesense server |
|---|---|
| `2.x` | `30.0`+ (uses global `synonym_sets`, `curation_sets`, v30-shape `analytics/rules`) |
| `1.x` | `29.x` |

## Source, issues, contributing

Source on [GitHub](https://github.com/Fluent-Health/terraform-provider-typesense). Bug reports and PRs welcome — see [CONTRIBUTING.md](https://github.com/Fluent-Health/terraform-provider-typesense/blob/main/CONTRIBUTING.md). This repository is a [Fluent Health](https://github.com/Fluent-Health) fork of [ronati/terraform-provider-typesense](https://github.com/ronati/terraform-provider-typesense), originally created by [Keisuke Yamashita](https://github.com/KeisukeYamashita).

## Example Usage

```terraform
provider "typesense" {
  api_key     = "xxxxxxxxxxxxxxxxxx"            // Or TYPESENSE_API_KEY environment variable
  api_address = "https://your.typesense.server" // Or TYPESENSE_API_ADDRESS environment variable
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `api_address` (String) URL of the Typesense server. This can also be set via the `TYPESENSE_API_ADDRESS` environment variable.
- `api_key` (String, Sensitive) API Key to access the Typesense server. This can also be set via the `TYPESENSE_API_KEY` environment variable.
- `request_timeout` (String) Timeout for a single HTTP request to the Typesense server, as a Go duration (e.g. `10m`). Defaults to `5m`.
- `schema_change_timeout` (String) How long to wait for a collection schema change to finish server-side after its PATCH request ended early (gateway timeout, dropped connection) or was refused because another change was running, as a Go duration (e.g. `90m`). Defaults to `60m`.
