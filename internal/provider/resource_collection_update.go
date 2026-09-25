package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"fluent-health-terraform-typesense/internal/typesense"
)

// buildCollectionUpdateSchema diffs the prior field set against the planned
// one and returns the PATCH body: new fields are added, changed fields are
// dropped and re-added (Typesense has no in-place field update), and removed
// fields are dropped.
func buildCollectionUpdateSchema(ctx context.Context, stateFields, planFields []CollectionResourceFieldModel) *typesense.CollectionUpdateSchema {
	stateItems := make(map[string]CollectionResourceFieldModel)

	for i := 0; i < len(stateFields); i += 1 {
		stateItems[stateFields[i].Name.ValueString()] = stateFields[i]
	}

	schema := &typesense.CollectionUpdateSchema{}

	var drop = new(bool)
	*drop = true

	for _, field := range planFields {
		// item not exists, need to create
		if _, ok := stateItems[field.Name.ValueString()]; !ok {
			schema.Fields = append(schema.Fields, filedModelToApiField(field))

			tflog.Info(ctx, "###Field will be created: "+field.Name.ValueString())

		} else if !fieldsEqual(stateItems[field.Name.ValueString()], field) {
			// item was changed, need to update

			schema.Fields = append(schema.Fields,
				typesense.Field{
					Drop: drop,
					Name: field.Name.ValueString(),
				},
				filedModelToApiField(field))
			tflog.Info(ctx, "###Field will be updated: "+field.Name.ValueString())

		} else {
			// item was not changed, do nothing
			tflog.Info(ctx, "###Field remaining the same: "+field.Name.ValueString())
		}

		// delete processed field from the state object
		delete(stateItems, field.Name.ValueString())
	}

	for _, field := range stateItems {
		schema.Fields = append(schema.Fields,
			typesense.Field{
				Drop: drop,
				Name: field.Name.ValueString(),
			})
		tflog.Info(ctx, "###Field will be deleted: "+field.Name.ValueString())
	}

	return schema
}

// applyCollectionUpdate sends the PATCH and survives the two ways a long alter
// (e.g. re-indexing an embedded field over a large collection) outlives its
// HTTP request:
//
//   - The request itself ends first (gateway 504/408/502, client timeout,
//     dropped connection). Typesense keeps running the alter, so failing here
//     would leave state behind the server: the next plan re-sends the same
//     change and starts the whole alter again. Instead, wait for the alter to
//     finish and check that the live schema has the planned shape.
//   - A previous alter on the same collection is still running (422 "Another
//     collection update operation is in progress"). Wait for it, re-diff
//     against the live schema, and send whatever is still needed once.
//
// Verification is structural (field names, types, and whether an embed block
// is present). Write-only embed credentials are never echoed by the server, so
// a credential-only change is confirmed by the alter completing, not by
// reading it back.
func applyCollectionUpdate(ctx context.Context, client *typesense.Client, name string, schema *typesense.CollectionUpdateSchema, stateFields, planFields []CollectionResourceFieldModel) error {
	_, err := client.UpdateCollection(ctx, name, schema)
	switch {
	case err == nil:
		return nil
	case typesense.IsRequestOutlived(err):
		return awaitAndVerifyCollectionUpdate(ctx, client, name, schema, planFields, err)
	case typesense.IsAlterInProgress(err):
		tflog.Warn(ctx, "another schema change is running on this collection; waiting for it before re-checking", map[string]interface{}{"collection": name})
		if werr := client.WaitForSchemaChange(ctx, name); werr != nil {
			return fmt.Errorf("%w; waiting for the running schema change: %v", err, werr)
		}
		live, gerr := client.GetCollection(ctx, name)
		if gerr != nil {
			return fmt.Errorf("%w; reading the collection after the running schema change finished: %v", err, gerr)
		}
		retry := buildCollectionUpdateSchema(ctx, flattenCollectionFields(live.Fields, stateFields), planFields)
		if len(retry.Fields) == 0 {
			return nil
		}
		_, rerr := client.UpdateCollection(ctx, name, retry)
		switch {
		case rerr == nil:
			return nil
		case typesense.IsRequestOutlived(rerr):
			return awaitAndVerifyCollectionUpdate(ctx, client, name, retry, planFields, rerr)
		default:
			return fmt.Errorf("retrying after a concurrent schema change finished: %w", rerr)
		}
	default:
		return err
	}
}

func awaitAndVerifyCollectionUpdate(ctx context.Context, client *typesense.Client, name string, schema *typesense.CollectionUpdateSchema, planFields []CollectionResourceFieldModel, cause error) error {
	tflog.Warn(ctx, "collection PATCH ended before the server answered; waiting for the schema change to finish server-side", map[string]interface{}{
		"collection": name,
		"cause":      cause.Error(),
	})
	if err := client.WaitForSchemaChange(ctx, name); err != nil {
		return fmt.Errorf("PATCH ended early (%v) and the schema change could not be confirmed: %w", cause, err)
	}
	live, err := client.GetCollection(ctx, name)
	if err != nil {
		return fmt.Errorf("PATCH ended early (%v); reading the collection after the schema change: %w", cause, err)
	}
	if mismatches := liveSchemaMismatches(live.Fields, schema, planFields); len(mismatches) > 0 {
		return fmt.Errorf("PATCH ended early (%v) and after the schema change finished the live schema does not match the plan: %s", cause, strings.Join(mismatches, "; "))
	}
	tflog.Info(ctx, "schema change finished server-side and the live schema matches the plan", map[string]interface{}{"collection": name})
	return nil
}

// liveSchemaMismatches compares the live field set with the planned one: every
// planned field must be live with the planned type and embed presence, and
// every field the PATCH dropped without re-adding must be gone.
func liveSchemaMismatches(live []typesense.Field, schema *typesense.CollectionUpdateSchema, planFields []CollectionResourceFieldModel) []string {
	liveByName := make(map[string]typesense.Field, len(live))
	for _, f := range live {
		liveByName[f.Name] = f
	}

	var mismatches []string
	planned := make(map[string]bool, len(planFields))
	for _, p := range planFields {
		name := p.Name.ValueString()
		planned[name] = true
		l, ok := liveByName[name]
		if !ok {
			mismatches = append(mismatches, fmt.Sprintf("field %q is missing", name))
			continue
		}
		if want := p.Type.ValueString(); l.Type != want {
			mismatches = append(mismatches, fmt.Sprintf("field %q is type %q, planned %q", name, l.Type, want))
		}
		if wantEmbed, gotEmbed := p.Embed != nil, l.Embed != nil; wantEmbed != gotEmbed {
			mismatches = append(mismatches, fmt.Sprintf("field %q embed present=%t, planned %t", name, gotEmbed, wantEmbed))
		}
	}

	for _, f := range schema.Fields {
		if f.Drop == nil || !*f.Drop || planned[f.Name] {
			continue
		}
		if _, ok := liveByName[f.Name]; ok {
			mismatches = append(mismatches, fmt.Sprintf("field %q was dropped but is still present", f.Name))
		}
	}

	return mismatches
}
