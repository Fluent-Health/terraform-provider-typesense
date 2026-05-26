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
