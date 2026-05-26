<div align="center">
  <h1>Terraform Provider for Typesense</h1>

  [![Tests](https://github.com/Fluent-Health/terraform-provider-typesense/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/Fluent-Health/terraform-provider-typesense/actions/workflows/build-and-test.yml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/Fluent-Health/terraform-provider-typesense)](https://goreportcard.com/report/github.com/Fluent-Health/terraform-provider-typesense)
</div>

<hr>

A Terraform provider for managing [Typesense](https://typesense.org) collections, documents, API keys, aliases, presets, stopwords, synonym sets, curation sets, analytics rules, conversation models, NL search models, and stemming dictionaries.

> This repository is a **[Fluent Health](https://github.com/Fluent-Health)** fork of [ronati/terraform-provider-typesense](https://github.com/ronati/terraform-provider-typesense) (originally created by [Keisuke Yamashita](https://github.com/KeisukeYamashita)). It targets Typesense v30+ and exposes a few server-side fields that the upstream Go SDK is missing.

## Resources

| Resource | Typesense API |
|---|---|
| `typesense_collection` | [collections](https://typesense.org/docs/30.2/api/collections.html) |
| `typesense_document` | [documents](https://typesense.org/docs/30.2/api/documents.html) |
| `typesense_alias` | [collection alias](https://typesense.org/docs/30.2/api/collection-alias.html) |
| `typesense_api_key` | [API keys](https://typesense.org/docs/30.2/api/api-keys.html) |
| `typesense_preset` | [search presets](https://typesense.org/docs/30.2/api/search.html#presets) |
| `typesense_stopword` | [stopwords](https://typesense.org/docs/30.2/api/stopwords.html) |
| `typesense_stemming_dictionary` | [stemming dictionaries](https://typesense.org/docs/30.2/api/stemming.html) |
| `typesense_synonym_set` | [synonym sets](https://typesense.org/docs/30.2/api/synonyms.html) (v30+) |
| `typesense_curation_set` | [curation sets](https://typesense.org/docs/30.2/api/curation.html) (v30+) |
| `typesense_analytics_rule` | [analytics rules](https://typesense.org/docs/30.2/api/analytics-query-suggestions.html) (v30+ shape) |
| `typesense_conversation_model` | [conversational search](https://typesense.org/docs/30.2/api/conversational-search-rag.html) |
| `typesense_nl_search_model` | [natural language search](https://typesense.org/docs/30.2/api/natural-language-search.html) (v29+) |

Full per-resource docs in [`docs/resources/`](./docs/resources/).

## What's different in this fork

The provider talks to Typesense over plain HTTP (`internal/typesense` package) rather than going through `github.com/typesense/typesense-go`. The SDK is generated from the upstream OpenAPI spec, which lags the server on a few embedder/auth fields — owning a thin HTTP layer lets us surface those without waiting on regenerations:

- **`typesense_collection.fields.embed.model_config.region`** — GCP region for Vertex AI embedders.
- **`typesense_collection.fields.embed.model_config.service_account`** — service-account auth path for managed Vertex AI embedders (`client_email`, `private_key`, optional `token_uri`). No refresh-token rotation.
- **`typesense_collection.fields.async_reference`** — index documents whose referenced row doesn't exist yet.
- **`typesense_nl_search_model.service_account`** — same SA-based auth for Vertex-backed NL models (e.g. `gcp/gemini-2.5-flash`).

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.12
- [Go](https://golang.org/doc/install) 1.24+ (for building the provider)
- [Typesense](https://typesense.org) server v30.0+ (uses the global `synonym_sets` and `curation_sets` endpoints introduced in v30)

## Quick start

```hcl
terraform {
  required_providers {
    typesense = {
      source = "Fluent-Health/typesense"
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
  }

  default_sorting_field = "year"
}
```

## Building

```sh
git clone https://github.com/Fluent-Health/terraform-provider-typesense
cd terraform-provider-typesense
make build
```

## Testing

### Unit tests

```sh
go test -v -short ./...
```

### Acceptance tests

Acceptance tests need a real Typesense server:

```sh
docker run -d --name typesense-test -p 8108:8108 \
  -v "$(docker volume create typesense-test-data):/data" \
  typesense/typesense:30.2 \
  --data-dir /data --api-key=test-api-key --enable-cors

make testacc

docker rm -f typesense-test
docker volume rm typesense-test-data
```

Tests default to `http://localhost:8108` with API key `test-api-key` if `TYPESENSE_API_ADDRESS` / `TYPESENSE_API_KEY` are not set. A handful of tests are env-var-gated because Typesense validates credentials at create time (`OPENAI_API_KEY` for NL models with OpenAI; `GCP_SA_CLIENT_EMAIL` + `GCP_SA_PRIVATE_KEY` + `GCP_PROJECT_ID` + `GCP_REGION` for the Vertex service-account path).

## Releasing

Releases are cut by pushing a `v*` tag. [GoReleaser](https://goreleaser.com) builds the per-platform zips, signs `SHA256SUMS` with GPG, and publishes a GitHub Release that the Terraform Registry can ingest.

```sh
git tag v1.2.3
git push origin v1.2.3
```

The release workflow lives in [`.github/workflows/release-go.yml`](.github/workflows/release-go.yml) and requires the `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets to be configured in the `release` environment.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Bug reports and pull requests welcome.

## License

[MIT](./LICENSE)
