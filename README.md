<div align="center">
  <h1>Terraform Provider for Typesense</h1>
  <strong>This is a Terraform provider for Typesense</strong>

  <br><br>

  [![Tests](https://github.com/Fluent-Health/terraform-provider-typesense/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/Fluent-Health/terraform-provider-typesense/actions/workflows/build-and-test.yml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/Fluent-Health/terraform-provider-typesense)](https://goreportcard.com/report/github.com/Fluent-Health/terraform-provider-typesense)
</div>

<hr>

> This repository is a **[Fluent Health](https://github.com/Fluent-Health)** fork of [ronati/terraform-provider-typesense](https://github.com/ronati/terraform-provider-typesense) (originally created by [Keisuke Yamashita](https://github.com/KeisukeYamashita)). It has been extended to target Typesense v30.1 and adds resources for synonym sets, curation sets, analytics rules, conversation models, NL search models, presets, stopwords, and stemming dictionaries.

## Support

- Supports Typesense v30.0+ (uses the global `synonym_sets` and `curation_sets` endpoints introduced in v30).

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= v0.12.0 (v0.11.x may work but not supported actively)

## Building The Provider

Clone repository to: `$GOPATH/src/github.com/Fluent-Health/terraform-provider-typesense`

```console
$ mkdir -p $GOPATH/src/github.com/Fluent-Health; cd $GOPATH/src/github.com/Fluent-Health
$ git clone git@github.com:Fluent-Health/terraform-provider-typesense
Enter the provider directory and build the provider

$ cd $GOPATH/src/github.com/Fluent-Health/terraform-provider-typesense
$ make build
```

## Testing

### Running Tests Locally

#### Unit Tests
```bash
go test -v -short ./...
```

#### Acceptance Tests

Acceptance tests require a running Typesense instance:

```bash
# Start Typesense
docker run -d --name typesense-test \
  -p 8108:8108 \
  -e TYPESENSE_DATA_DIR=/tmp \
  -e TYPESENSE_API_KEY=test-api-key \
  -e TYPESENSE_ENABLE_SEARCH_ANALYTICS=true \
  typesense/typesense:30.1

# Wait for it to start
sleep 5

# Run tests (will use localhost:8108 by default)
make testacc

# Clean up
docker stop typesense-test && docker rm typesense-test
```

**Note:** Tests will automatically connect to `http://localhost:8108` with API key `test-api-key` if environment variables are not set.

## Contributing

**All contributions are welcome!**

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for automated semantic versioning and releases. Please read our [Contributing Guide](CONTRIBUTING.md) for details on:

- Setting up your development environment
- Commit message format and validation
- Testing requirements
- Pull request process

### Quick Start for Contributors

```bash
# Clone and setup
git clone https://github.com/Fluent-Health/terraform-provider-typesense.git
cd terraform-provider-typesense
npm install

# Setup git hooks for commit validation (optional)
./scripts/setup-git-hooks.sh

# Make changes and commit following conventional commits format
git commit -m "feat: add new feature"
```

### Commit Message Format

All commits must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): subject

Examples:
  feat: add support for nested fields
  fix: resolve document update issue
  docs: update README
  test: add tests for synonym resource
```

**Note**: Commit messages are automatically validated in CI. PRs with invalid commit messages will fail the build.

## CI/CD

This project uses GitHub Actions for continuous integration and deployment:

- **Pull Requests**: Validates commit messages and runs all tests
- **Master/Beta Branch**: Automatically creates releases using semantic versioning
- **Version Tags**: Publishes provider to Terraform Registry

See [GitHub Workflows Documentation](.github/workflows/README.md) for more details.

## Notes for Maintainers

When you merge a PR from `beta` into `main` and it successfully publishes a new version on the `latest` channel, **don't forget to create a PR from `main` to `beta`**. This is mandatory for `semantic-release` to take it into account for next `beta` version.
