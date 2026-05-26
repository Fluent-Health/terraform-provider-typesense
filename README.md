<div align="center">
  <h1>Terraform Provider for Typesense</h1>

  [![Tests](https://github.com/Fluent-Health/terraform-provider-typesense/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/Fluent-Health/terraform-provider-typesense/actions/workflows/build-and-test.yml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/Fluent-Health/terraform-provider-typesense)](https://goreportcard.com/report/github.com/Fluent-Health/terraform-provider-typesense)
</div>

<hr>

A Terraform provider for managing [Typesense](https://typesense.org) collections, documents, API keys, aliases, presets, stopwords, synonym sets, curation sets, analytics rules, conversation models, NL search models, and stemming dictionaries.

> This repository is a **[Fluent Health](https://github.com/Fluent-Health)** fork of [ronati/terraform-provider-typesense](https://github.com/ronati/terraform-provider-typesense) (originally created by [Keisuke Yamashita](https://github.com/KeisukeYamashita)). It has been extended to target Typesense v30.1 and adds resources for the v29/v30-era endpoints.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.12
- [Go](https://golang.org/doc/install) 1.24+ (for building the provider)
- [Typesense](https://typesense.org) server v30.0+ (uses the global `synonym_sets` and `curation_sets` endpoints introduced in v30)

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

Acceptance tests require a running Typesense instance:

```sh
docker run -d --name typesense-test \
  -p 8108:8108 \
  -e TYPESENSE_DATA_DIR=/tmp \
  -e TYPESENSE_API_KEY=test-api-key \
  -e TYPESENSE_ENABLE_SEARCH_ANALYTICS=true \
  typesense/typesense:30.1

make testacc

docker rm -f typesense-test
```

Tests default to `http://localhost:8108` with API key `test-api-key` if `TYPESENSE_API_ADDRESS` / `TYPESENSE_API_KEY` are not set.

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
