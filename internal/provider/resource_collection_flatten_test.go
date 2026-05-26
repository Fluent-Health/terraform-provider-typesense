package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"fluent-health-terraform-typesense/internal/typesense"
)

// TestFlattenFieldEmbed_PreservesSensitiveValues verifies the regression fix
// for the post-import "drop+re-add embedding field on every apply" bug. The
// Typesense API never echoes write-only credentials on a subsequent GET, so a
// fresh flatten without `prior` would silently blank them — creating a
// phantom diff and a destructive drop+add of the embedding field on Update.
func TestFlattenFieldEmbed_PreservesSensitiveValues(t *testing.T) {
	priv := func(s string) *string { return &s }

	// Server response: model_config WITHOUT the sensitive fields populated.
	apiEmbed := &typesense.FieldEmbed{
		From: []string{"display"},
		ModelConfig: typesense.FieldEmbedModelConfig{
			ModelName: "gcp/text-embedding-005",
			ProjectId: priv("my-proj"),
			Region:    priv("us-central1"),
			ServiceAccount: &typesense.GCPServiceAccount{
				ClientEmail: "vertex@my-proj.iam.gserviceaccount.com",
				// PrivateKey deliberately empty — the server doesn't echo it.
			},
		},
	}

	prior := &CollectionFieldEmbedModel{
		ModelConfig: &CollectionFieldEmbedModelConfigModel{
			ModelName:    types.StringValue("gcp/text-embedding-005"),
			AccessToken:  types.StringValue("at-secret"),
			ApiKey:       types.StringValue("ak-secret"),
			ClientSecret: types.StringValue("cs-secret"),
			RefreshToken: types.StringValue("rt-secret"),
			ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
				ClientEmail: types.StringValue("vertex@my-proj.iam.gserviceaccount.com"),
				PrivateKey:  types.StringValue("-----BEGIN PRIVATE KEY-----\n..."),
			},
		},
	}

	got := flattenFieldEmbed(apiEmbed, prior)
	if got == nil || got.ModelConfig == nil {
		t.Fatalf("flattenFieldEmbed returned nil ModelConfig")
	}

	cases := []struct {
		name string
		got  types.String
		want string
	}{
		{"access_token", got.ModelConfig.AccessToken, "at-secret"},
		{"api_key", got.ModelConfig.ApiKey, "ak-secret"},
		{"client_secret", got.ModelConfig.ClientSecret, "cs-secret"},
		{"refresh_token", got.ModelConfig.RefreshToken, "rt-secret"},
	}
	for _, tc := range cases {
		if tc.got.ValueString() != tc.want {
			t.Errorf("%s = %q, want %q (server didn't echo it; should have been preserved from prior)", tc.name, tc.got.ValueString(), tc.want)
		}
	}

	if got.ModelConfig.ServiceAccount == nil {
		t.Fatalf("service_account should be populated (server echoed structural object)")
	}
	if got.ModelConfig.ServiceAccount.PrivateKey.ValueString() != "-----BEGIN PRIVATE KEY-----\n..." {
		t.Errorf("service_account.private_key = %q, want preserved from prior",
			got.ModelConfig.ServiceAccount.PrivateKey.ValueString())
	}
}

// TestFlattenFieldEmbed_NoPrior verifies sensitive fields stay null when there
// is no prior state to preserve from (e.g. the Create path with a config that
// doesn't set them).
func TestFlattenFieldEmbed_NoPrior(t *testing.T) {
	apiEmbed := &typesense.FieldEmbed{
		From: []string{"display"},
		ModelConfig: typesense.FieldEmbedModelConfig{
			ModelName: "ts/clip-vit-b-p32",
		},
	}
	got := flattenFieldEmbed(apiEmbed, nil)
	if got == nil || got.ModelConfig == nil {
		t.Fatalf("flattenFieldEmbed returned nil ModelConfig")
	}
	for name, v := range map[string]types.String{
		"access_token":  got.ModelConfig.AccessToken,
		"api_key":       got.ModelConfig.ApiKey,
		"client_secret": got.ModelConfig.ClientSecret,
		"refresh_token": got.ModelConfig.RefreshToken,
	} {
		if !v.IsNull() {
			t.Errorf("%s should be null with no prior state, got %q", name, v.ValueString())
		}
	}
}

// TestFlattenFieldEmbed_PreservesServiceAccountBlockWhenServerOmits verifies
// that when the server doesn't echo the service_account block at all, the
// entire block is restored from prior — including non-sensitive fields like
// client_email — so we don't lose the user's SA configuration in state.
func TestFlattenFieldEmbed_PreservesServiceAccountBlockWhenServerOmits(t *testing.T) {
	priv := func(s string) *string { return &s }
	apiEmbed := &typesense.FieldEmbed{
		From: []string{"display"},
		ModelConfig: typesense.FieldEmbedModelConfig{
			ModelName: "gcp/text-embedding-005",
			ProjectId: priv("my-proj"),
			Region:    priv("us-central1"),
			// ServiceAccount intentionally nil — simulating a server that
			// strips the whole block on Read.
		},
	}
	prior := &CollectionFieldEmbedModel{
		ModelConfig: &CollectionFieldEmbedModelConfigModel{
			ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
				ClientEmail: types.StringValue("vertex@my-proj.iam.gserviceaccount.com"),
				PrivateKey:  types.StringValue("pk"),
			},
		},
	}
	got := flattenFieldEmbed(apiEmbed, prior)
	if got.ModelConfig.ServiceAccount == nil {
		t.Fatalf("service_account should be restored from prior when server omits it")
	}
	if got.ModelConfig.ServiceAccount.ClientEmail.ValueString() != "vertex@my-proj.iam.gserviceaccount.com" {
		t.Errorf("client_email = %q, want preserved from prior", got.ModelConfig.ServiceAccount.ClientEmail.ValueString())
	}
}

// TestFieldsEqual_PostImportNullStateMatchesNonNullPlan: the core regression
// guard. State has null sensitive (because of import), plan has the user's
// real values. fieldsEqual must return true so Update does NOT trigger
// drop+add (which would re-embed the whole collection).
func TestFieldsEqual_PostImportNullStateMatchesNonNullPlan(t *testing.T) {
	stateField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		Embed: &CollectionFieldEmbedModel{
			From: []types.String{types.StringValue("title")},
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("gcp/text-embedding-005"),
				// All sensitive fields null (post-import).
				ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
					ClientEmail: types.StringValue("vertex@my-proj.iam.gserviceaccount.com"),
					// PrivateKey null — server didn't echo it.
				},
			},
		},
	}
	planField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		Embed: &CollectionFieldEmbedModel{
			From: []types.String{types.StringValue("title")},
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName:    types.StringValue("gcp/text-embedding-005"),
				AccessToken:  types.StringValue("at"),
				ApiKey:       types.StringValue("ak"),
				ClientSecret: types.StringValue("cs"),
				RefreshToken: types.StringValue("rt"),
				ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
					ClientEmail: types.StringValue("vertex@my-proj.iam.gserviceaccount.com"),
					PrivateKey:  types.StringValue("real-pk"),
				},
			},
		},
	}
	if !fieldsEqual(stateField, planField) {
		t.Errorf("fieldsEqual should be true when state has null sensitive and plan has real values (post-import case); got false — this would trigger a destructive drop+add re-embed")
	}
}

// TestFieldsEqual_RotationDetected: user actively changes a sensitive value
// post-import. State has the old value, plan has a new one. fieldsEqual
// must return false so Update triggers drop+add and the new credential
// reaches the server.
func TestFieldsEqual_RotationDetected(t *testing.T) {
	stateField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("gcp/text-embedding-005"),
				ApiKey:    types.StringValue("old-key"),
			},
		},
	}
	planField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("gcp/text-embedding-005"),
				ApiKey:    types.StringValue("new-key"),
			},
		},
	}
	if fieldsEqual(stateField, planField) {
		t.Errorf("fieldsEqual should be false when state and plan have *different* non-null api_keys; got true — rotation would silently fail to reach the server")
	}
}

// TestFieldsEqual_NonSensitiveDiffStillTriggersUpdate: state has null
// sensitive (post-import) AND a different model_name. The non-sensitive
// diff must still trigger drop+add so the actual change reaches the server.
func TestFieldsEqual_NonSensitiveDiffStillTriggersUpdate(t *testing.T) {
	stateField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("ts/clip-vit-b-p32"),
			},
		},
	}
	planField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName:    types.StringValue("openai/text-embedding-3-small"),
				ApiKey:       types.StringValue("ak"),
				AccessToken:  types.StringValue("at"),
				ClientSecret: types.StringValue("cs"),
				RefreshToken: types.StringValue("rt"),
			},
		},
	}
	if fieldsEqual(stateField, planField) {
		t.Errorf("fieldsEqual should be false when model_name differs (even alongside post-import sensitive nulls); got true — non-sensitive change would never reach the server")
	}
}

// TestFieldsEqual_FullyEqual: standard case — state and plan match. Should
// return true so Update doesn't fire spuriously.
func TestFieldsEqual_FullyEqual(t *testing.T) {
	f := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("ts/clip-vit-b-p32"),
				ApiKey:    types.StringValue("ak"),
			},
		},
	}
	if !fieldsEqual(f, f) {
		t.Errorf("fieldsEqual should be true for identical state and plan")
	}
}

// TestAbsorbPostImportSensitive_DoesNotMutateInputs: critical correctness
// invariant — we deep-copy state's Embed/ModelConfig/ServiceAccount before
// filling fields. Without that, fieldsEqual would silently corrupt the
// caller's state map (which Update relies on for downstream iteration).
func TestAbsorbPostImportSensitive_DoesNotMutateInputs(t *testing.T) {
	state := CollectionResourceFieldModel{
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ApiKey: types.StringNull(),
				ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
					PrivateKey: types.StringNull(),
				},
			},
		},
	}
	plan := CollectionResourceFieldModel{
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ApiKey: types.StringValue("ak"),
				ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
					PrivateKey: types.StringValue("pk"),
				},
			},
		},
	}

	_ = absorbPostImportSensitive(state, plan)

	if !state.Embed.ModelConfig.ApiKey.IsNull() {
		t.Errorf("state.Embed.ModelConfig.ApiKey was mutated to %q; absorb must not touch its inputs", state.Embed.ModelConfig.ApiKey.ValueString())
	}
	if !state.Embed.ModelConfig.ServiceAccount.PrivateKey.IsNull() {
		t.Errorf("state.Embed.ModelConfig.ServiceAccount.PrivateKey was mutated to %q", state.Embed.ModelConfig.ServiceAccount.PrivateKey.ValueString())
	}
}

// TestFieldsEqual_PostImportAutoEmbedNumDimMatchesNullPlan: the auto-embed
// counterpart of TestFieldsEqual_PostImportNullStateMatchesNonNullPlan. For
// an auto-embedded float[] field, Typesense computes num_dim from the
// embedder's output dimension and echoes it on GET. HCL never sets num_dim
// on auto-embedded fields (the embedder owns that value), so post-import
// state has a concrete num_dim while plan stays null. Without asymmetric
// absorption, fieldsEqual returns false and Update appends a drop+add for
// the embedding field — which on Typesense's side re-embeds every document
// (PM-3851 dev incident: ~100k LOINC + ~300k SNOMED docs re-embedded for
// no user-visible change).
func TestFieldsEqual_PostImportAutoEmbedNumDimMatchesNullPlan(t *testing.T) {
	stateField := CollectionResourceFieldModel{
		Name:    types.StringValue("embedding"),
		Type:    types.StringValue("float[]"),
		NumDim:  types.Int64Value(768), // server-computed from gcp/gemini-embedding-001
		VecDist: types.StringValue("cosine"),
		Embed: &CollectionFieldEmbedModel{
			From: []types.String{types.StringValue("title")},
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("gcp/gemini-embedding-001"),
				ProjectId: types.StringValue("my-proj"),
				Region:    types.StringValue("asia-south1"),
				ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
					ClientEmail: types.StringValue("vertex@my-proj.iam.gserviceaccount.com"),
					// PrivateKey null — server didn't echo it.
				},
			},
		},
	}
	planField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		// NumDim null — HCL doesn't set it on auto-embedded fields.
		VecDist: types.StringValue("cosine"),
		Embed: &CollectionFieldEmbedModel{
			From: []types.String{types.StringValue("title")},
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("gcp/gemini-embedding-001"),
				ProjectId: types.StringValue("my-proj"),
				Region:    types.StringValue("asia-south1"),
				ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
					ClientEmail: types.StringValue("vertex@my-proj.iam.gserviceaccount.com"),
					PrivateKey:  types.StringValue("real-pk"),
				},
			},
		},
	}
	if !fieldsEqual(stateField, planField) {
		t.Errorf("fieldsEqual should be true when state has server-computed num_dim and plan has null on an auto-embedded field; got false — this would trigger a destructive drop+add re-embed of every document")
	}
}

// TestFieldsEqual_NumDimDiffersWithoutEmbedStillTriggersUpdate: manual
// vector field path — the user is the source of truth for num_dim, so a
// state-vs-plan mismatch is a real change and must still trigger drop+add.
// The absorb logic is scoped to fields that have an embed block on both
// sides so this case is unaffected.
func TestFieldsEqual_NumDimDiffersWithoutEmbedStillTriggersUpdate(t *testing.T) {
	stateField := CollectionResourceFieldModel{
		Name:    types.StringValue("vec"),
		Type:    types.StringValue("float[]"),
		NumDim:  types.Int64Value(384),
		VecDist: types.StringValue("cosine"),
	}
	planField := CollectionResourceFieldModel{
		Name:    types.StringValue("vec"),
		Type:    types.StringValue("float[]"),
		NumDim:  types.Int64Value(768),
		VecDist: types.StringValue("cosine"),
	}
	if fieldsEqual(stateField, planField) {
		t.Errorf("fieldsEqual should be false when num_dim differs on a manual vector field (no embed); got true — user's dim change would silently fail to reach the server")
	}
}

// TestFieldsEqual_NumDimAbsorbDoesNotFireWhenPlanRemovesEmbed: edge case —
// user removes the embed block (switching from auto-embedded to regular
// field). State still has num_dim from the old auto-embedded shape; plan
// has null. The absorb must NOT fire here, because the change is real
// (field type is effectively changing) and the drop+add is what we want.
func TestFieldsEqual_NumDimAbsorbDoesNotFireWhenPlanRemovesEmbed(t *testing.T) {
	stateField := CollectionResourceFieldModel{
		Name:   types.StringValue("embedding"),
		Type:   types.StringValue("float[]"),
		NumDim: types.Int64Value(768),
		Embed: &CollectionFieldEmbedModel{
			ModelConfig: &CollectionFieldEmbedModelConfigModel{
				ModelName: types.StringValue("gcp/gemini-embedding-001"),
			},
		},
	}
	planField := CollectionResourceFieldModel{
		Name: types.StringValue("embedding"),
		Type: types.StringValue("float[]"),
		// Embed removed by user; num_dim null.
	}
	if fieldsEqual(stateField, planField) {
		t.Errorf("fieldsEqual should be false when plan removes the embed block; got true — user's structural change would silently fail to reach the server")
	}
}
