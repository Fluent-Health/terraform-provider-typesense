# Terraform Provider for Typesense — v30.1 Upgrade Design

**Date:** 2026-05-25
**Status:** Draft, pending review

## Goal

Bring `terraform-provider-typesense` up to date with Typesense server **v30.1** features that are exposed by the stable `typesense-go` **v3.2.0** SDK, and verify every resource against a real Typesense container as part of CI.

## Background

The provider currently advertises support for Typesense `v28.0+` and tests against `typesense/typesense:29.0`. The `typesense-go` SDK at v3.2.0 already exposes several Typesense capabilities that the provider does **not** wrap as Terraform resources, and the `typesense_collection` resource omits several field-level attributes that the SDK supports.

The user is running Typesense 30.1 in production and wants:

1. Provider parity with what the v3.2.0 SDK can do against a v30.1 server.
2. Acceptance tests that run against a real Typesense container, covering all resources (existing + new).

## Non-goals

The following are explicitly **deferred** to a follow-up effort that upgrades to `typesense-go` v4 (currently `v4.0.0-alpha2`):

- `typesense_nl_search_model` — Natural Language Search (Typesense v29)
- `typesense_synonym_set` — global synonyms (Typesense v30)
- `typesense_curation_set` — global curation rules (Typesense v30)
- `async_reference` and `cascade_delete` JOIN attributes (Typesense v30)

These ride on the v4 SDK which is alpha. Holding them back keeps the upgrade on stable foundations.

## Scope

### A. Enhance `typesense_collection` (existing resource)

Add the following per-field attributes to the `fields` block. All five are already
present on `api.Field` in `typesense-go` v3.2.0:

| Attribute | Type | Notes |
|---|---|---|
| `reference` | string | JOIN reference (e.g. `"users.id"`); links this field to another collection's field |
| `range_index` | bool | Enables range-filter optimization for numeric fields |
| `vec_dist` | string | Vector distance metric: `cosine` (default) or `ip` |
| `symbols_to_index` | list(string) | Field-level override of collection setting (v28) |
| `token_separators` | list(string) | Field-level override of collection setting (v28) |

These are all `Optional` with `Computed` flattening on read. They participate in the
existing collection update flow (drop + re-add) via the existing
`fieldsEqual` mechanism in `resource_collection.go`.

### B. New resources (one Terraform resource per Typesense API)

All resources follow the existing pattern in `internal/provider/`:

- `terraform-plugin-framework` resource with embedded `*typesense.Client`
- `Configure` extracts the client from `req.ProviderData`
- `Create/Read/Update/Delete/ImportState`
- `"Not Found"` errors during `Read` → `resp.State.RemoveResource(ctx)`
- File per resource: `resource_<name>.go` + `resource_<name>_test.go`

| Resource | SDK entrypoint | Identity |
|---|---|---|
| `typesense_preset` | `client.Presets()` / `client.Preset(id)` | `name` (user-supplied id) |
| `typesense_stopword` | `client.Stopwords()` / `client.Stopword(id)` | `name` |
| `typesense_override` | `client.Collection(c).Overrides()` / `Override(id)` | `<collection>.<override_id>` |
| `typesense_stemming_dictionary` | `client.Stemming().Dictionaries()` / `Dictionary(id)` | `name` |
| `typesense_analytics_rule` | `client.Analytics().Rules()` / `Rule(name)` | `name` |
| `typesense_conversation_model` | `client.ConversationModels()` / `ConversationModel(id)` | server-assigned `id` |

#### `typesense_preset`

Schema body — `Required: name`, `Required: value` (JSON string of search parameters using
`jsontypes.Normalized` like the existing `typesense_document` resource). Update is supported
in place via `Presets().Upsert`. Import by `name`.

#### `typesense_stopword`

Schema — `Required: name`, `Required: stopwords` (list of strings), `Optional: locale` (string).
Upsert semantics (`Stopwords().Upsert`). Import by `name`.

#### `typesense_override`

Schema — `Required: collection_name` (RequiresReplace), `Required: name` (RequiresReplace),
`Required: rule` block (`query`, `match` ∈ {`exact`, `contains`}), optional blocks:
`includes` (list of `{id, position}`), `excludes` (list of `{id}`), `filter_by`,
`sort_by`, `replace_query`, `remove_matched_tokens`, `effective_from_ts`, `effective_to_ts`,
`stop_processing`. Compound ID `<collection>.<override_id>` follows the existing pattern in
`resource_synonym.go`.

#### `typesense_stemming_dictionary`

Schema — `Required: name`, `Required: words` (list of `{word, root}` objects).
Upsert via `Stemming().Dictionaries().Upsert(name, words)`. Import by `name`.
RequiresReplace on `name` (dictionary IDs are not renameable).

#### `typesense_analytics_rule`

Schema — `Required: name` (RequiresReplace), `Required: type` ∈ {`popular_queries`,
`nohits_queries`, `counter`, `log`}, `Required: params` block:
- `source` block: `collections` (list of strings), optional `events` list
- `destination` block: `collection` (string), optional `counter_field`
- optional `limit`, `expand_query`, `enable_auto_aggregation`

Upsert via `Analytics().Rules().Upsert`. Import by `name`.

#### `typesense_conversation_model`

Schema — Computed `id`, `Required: model_name`, `Required: history_collection`,
`Required: api_key` (sensitive), `Optional: system_prompt`, `Optional: ttl`,
`Optional: max_bytes`, `Optional: account_id`, `Optional: vllm_url`.
Create assigns server-side `id`. Update via `ConversationModels(...).Update`. Import by `id`.

### C. Test infrastructure (minimal)

Per user preference: stay close to the current pattern.

- `.github/workflows/build-and-test.yml`: bump `typesense:29.0` → `typesense:30.1` (services block).
- `.github/workflows/release.yml`: same bump.
- `README.md`: update Docker snippet and support statement from `v28.0+` → `v30.1+`.
- `GNUmakefile`: no change needed (acceptance tests already keyed off `TF_ACC=1`).
- Each new resource gets an acceptance test in `resource_<name>_test.go`. Required coverage:
  - Create + Read round-trip with assertions on returned attributes
  - Import state matches
  - Destroy + `CheckDestroy` verifies server-side removal
  - For resources with Update (preset, stopword, stemming_dictionary, analytics_rule, conversation_model, override-via-replace): a TestStep that exercises the update path

### D. Documentation and examples

For each new resource:

- `examples/resources/typesense_<resource>/resource.tf` — minimal working example
- `examples/resources/typesense_<resource>/import.sh` — import command example
- Regenerate `docs/resources/<resource>.md` via `make doc`
  (this runs `go generate ./...` which invokes `terraform-plugin-docs`)

For the enhanced `typesense_collection`:

- Update `examples/resources/typesense_collection/resource.tf` with a JOIN reference example
- Regenerate `docs/resources/collection.md`

## Architecture notes

**Provider registration** — `internal/provider/provider.go`'s `Resources` slice grows by 6.

**Client surface** — no provider-level changes; the existing `*typesense.Client` carries
all the needed sub-clients (`Presets()`, `Stopwords()`, `Collection(c).Overrides()`,
`Stemming()`, `Analytics()`, `ConversationModels()`).

**Compound IDs** — `resource_override` follows the `<collection>.<override_id>` convention
already used by `resource_synonym` and `resource_document` via `util.go`'s
`createId` / `splitCollectionRelatedId`. No new util helpers needed.

**JSON payload resources** — `typesense_preset.value` and any future schema where the
Typesense API accepts a free-form JSON blob should use `jsontypes.Normalized` (matching
`resource_document.go`) so plan output is stable across whitespace/key-order differences.

## File layout after this change

```
internal/provider/
  provider.go                              (Resources list grows)
  resource_alias.go / _test.go             (unchanged)
  resource_api_key.go / _test.go           (unchanged)
  resource_collection.go / _test.go        (new field attrs, new tests)
  resource_document.go / _test.go          (unchanged)
  resource_synonym.go / _test.go           (unchanged)
  resource_preset.go / _test.go            NEW
  resource_stopword.go / _test.go          NEW
  resource_override.go / _test.go          NEW
  resource_stemming_dictionary.go / _test.go NEW
  resource_analytics_rule.go / _test.go    NEW
  resource_conversation_model.go / _test.go NEW
  util.go                                  (unchanged)
```

## Risks and trade-offs

- **Alpha SDK avoided** — by staying on v3.2.0 we cannot expose v29/v30-only features
  (NL search, global synonym/curation sets, `async_reference`, `cascade_delete`).
  Documented as deferred.
- **Test image bump** — Typesense 30.1 is forward-compatible with v3.2.0 SDK calls,
  but some response shapes may have new optional fields. Acceptance tests will catch
  any unmarshaling issue.
- **Conversation model API key** — stored as `Sensitive` in state. Users should be
  aware Terraform state is not encrypted by default.
- **`typesense_override` lifecycle** — overrides are per-collection but Typesense
  allows `RequiresReplace` semantics only on (collection, name). Renaming requires
  destroy+create.

## Sequencing

Suggested implementation order (each step independently testable):

1. Bump CI / README to typesense 30.1 — verify all existing tests still pass
2. Enhance `typesense_collection` with new field attributes + tests
3. `typesense_preset` (simplest, no compound ID)
4. `typesense_stopword`
5. `typesense_stemming_dictionary`
6. `typesense_analytics_rule`
7. `typesense_override` (compound ID — uses existing util helpers)
8. `typesense_conversation_model` (server-assigned ID, most complex schema)
9. Regenerate docs and examples for everything
10. Update support statement in README

## Open questions

None known at design time. Will revisit if SDK behaviors against a real v30.1 server
diverge from the v3.2.0 generated types during implementation.
