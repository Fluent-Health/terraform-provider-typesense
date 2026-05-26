# Contributing

Thanks for your interest in contributing.

## Reporting bugs

Open an issue describing the problem, steps to reproduce, the provider version, and the Typesense server version.

## Proposing changes

1. Fork the repository.
2. Create a branch: `git checkout -b my-fix`.
3. Make your changes and commit with a clear message.
4. Open a pull request against `main`.

## Development setup

### Prerequisites

- Go 1.22+
- Docker (for running Typesense locally during acceptance tests)
- Terraform CLI

### Build

```sh
make build
```

### Run a local Typesense

```sh
docker run -d --name typesense \
  -p 8108:8108 \
  -e TYPESENSE_DATA_DIR=/tmp \
  -e TYPESENSE_API_KEY=test-api-key \
  -e TYPESENSE_ENABLE_SEARCH_ANALYTICS=true \
  typesense/typesense:30.1
```

### Run tests

```sh
# Unit tests
go test -v -short ./...

# Acceptance tests (needs Typesense running)
make testacc

# A single test
TF_ACC=1 go test -v ./internal/provider/ -run TestAccCollectionResource
```

### Regenerate documentation

```sh
make doc
```

## Pull request checklist

- [ ] `make build` passes
- [ ] `make testacc` passes against a local Typesense
- [ ] New resources / attributes have acceptance tests covering create, read, update, import
- [ ] `make doc` was run if schemas changed
- [ ] PR description explains the change and links any related issues

## Release process

Releases are cut manually by a maintainer:

1. Update `CHANGELOG.md`.
2. Tag the commit: `git tag vX.Y.Z && git push origin vX.Y.Z`.
3. The [release workflow](.github/workflows/release-go.yml) builds, GPG-signs, and publishes the GitHub release.
4. The Terraform Registry picks up the new version via its GitHub webhook.

## No CLA required

You do not need to sign a Contributor License Agreement. By submitting a pull request, you agree to license your contribution under the repository's existing [LICENSE](./LICENSE).
