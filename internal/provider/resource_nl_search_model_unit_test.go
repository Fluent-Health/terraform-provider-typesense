package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"fluent-health-terraform-typesense/internal/typesense"
)

// TestApplyNLSchemaToModel_PreservesPriorWhenServerReturnsMaskedProjectId:
// Typesense 30.x masks credential-like fields on GET /nl_search_models —
// project_id, client_id, service_account.client_email come back as
// "fh-de*****" / "***********" / "searc***********" rather than the
// real value. applyNLSchemaToModel must keep the caller's prior real
// value when the server returns the masking pattern; otherwise
// Terraform's Create/Update returns inconsistent state ("planned
// project_id = fh-dev-svc, got fh-de*****") and the apply fails with
// "Provider produced inconsistent result after apply".
func TestApplyNLSchemaToModel_PreservesPriorWhenServerReturnsMaskedProjectId(t *testing.T) {
	priv := func(s string) *string { return &s }
	maskedProjectId := "fh-de*****"
	maskedClientId := "***********"
	maskedClientEmail := "searc***********************"

	server := &typesense.NLSearchModel{
		Id:        "nl-search-model",
		ModelName: priv("gcp/gemini-2.5-flash"),
		ProjectId: &maskedProjectId,
		ClientId:  &maskedClientId,
		Region:    priv("asia-south1"),
		ServiceAccount: &typesense.GCPServiceAccount{
			ClientEmail: maskedClientEmail,
			// Server doesn't echo PrivateKey — it's "" on the wire.
			PrivateKey: "",
		},
	}

	prior := NLSearchModelResourceModel{
		Id:        types.StringValue("nl-search-model"),
		ModelName: types.StringValue("gcp/gemini-2.5-flash"),
		ProjectId: types.StringValue("fh-dev-svc"),
		Region:    types.StringValue("asia-south1"),
		ServiceAccount: &GCPServiceAccountModel{
			ClientEmail: types.StringValue("vertex@fh-dev-svc.iam.gserviceaccount.com"),
			PrivateKey:  types.StringValue("-----BEGIN PRIVATE KEY-----\nreal-pk-bytes\n-----END PRIVATE KEY-----\n"),
		},
	}

	applyNLSchemaToModel(server, &prior)

	if prior.ProjectId.ValueString() != "fh-dev-svc" {
		t.Errorf("project_id = %q, want %q preserved from prior — server returned masked value, must not overwrite real value in state",
			prior.ProjectId.ValueString(), "fh-dev-svc")
	}
	if prior.ServiceAccount == nil {
		t.Fatalf("service_account should remain populated")
	}
	if prior.ServiceAccount.ClientEmail.ValueString() != "vertex@fh-dev-svc.iam.gserviceaccount.com" {
		t.Errorf("client_email = %q, want preserved from prior — server returned masked value",
			prior.ServiceAccount.ClientEmail.ValueString())
	}
	if prior.ServiceAccount.PrivateKey.ValueString() != "-----BEGIN PRIVATE KEY-----\nreal-pk-bytes\n-----END PRIVATE KEY-----\n" {
		t.Errorf("private_key was %q, want preserved from prior — server doesn't echo private_key, must not blank state",
			prior.ServiceAccount.PrivateKey.ValueString())
	}
	// client_id: prior was unset (Optional+Computed scenario, HCL doesn't
	// set it for the SA flow). Server returns masked. With no prior real
	// value to preserve, we accept the masked server value — there's no
	// better option. This matches the collection flatten behaviour.
	if prior.ClientId.ValueString() != "***********" {
		t.Errorf("client_id = %q, want %q from server (no prior to preserve)",
			prior.ClientId.ValueString(), "***********")
	}
}

// TestApplyNLSchemaToModel_RealServerValueOverridesPrior: when the
// server returns a real (non-masked) value, it must override the prior
// even if the prior was non-null. Otherwise external changes (e.g.
// someone editing the model directly via the Typesense API) wouldn't
// show up on the next refresh.
func TestApplyNLSchemaToModel_RealServerValueOverridesPrior(t *testing.T) {
	priv := func(s string) *string { return &s }
	server := &typesense.NLSearchModel{
		Id:        "nl-search-model",
		ProjectId: priv("fh-prod-svc"), // user moved the model to a different GCP project externally
		Region:    priv("us-central1"),
	}
	prior := NLSearchModelResourceModel{
		Id:        types.StringValue("nl-search-model"),
		ProjectId: types.StringValue("fh-dev-svc"),
		Region:    types.StringValue("asia-south1"),
	}

	applyNLSchemaToModel(server, &prior)

	if prior.ProjectId.ValueString() != "fh-prod-svc" {
		t.Errorf("project_id = %q, want %q from server — real (non-masked) server value must override prior",
			prior.ProjectId.ValueString(), "fh-prod-svc")
	}
	if prior.Region.ValueString() != "us-central1" {
		t.Errorf("region = %q, want %q from server", prior.Region.ValueString(), "us-central1")
	}
}

// TestApplyNLSchemaToModel_FreshImportNoPriorAcceptsServerMasked: on a
// freshly-imported state, prior is null. The function falls through and
// stores the server's masked value — there's no better alternative
// (nothing real to preserve). The first apply after import will then
// send the user's real HCL value to the server and the post-apply
// state will overwrite the masked entry, so this only persists for one
// plan cycle.
func TestApplyNLSchemaToModel_FreshImportNoPriorAcceptsServerMasked(t *testing.T) {
	maskedProjectId := "fh-de*****"
	server := &typesense.NLSearchModel{
		Id:        "nl-search-model",
		ProjectId: &maskedProjectId,
	}
	prior := NLSearchModelResourceModel{
		Id: types.StringValue("nl-search-model"),
		// ProjectId is unset / null — fresh-import shape
	}

	applyNLSchemaToModel(server, &prior)

	if prior.ProjectId.ValueString() != "fh-de*****" {
		t.Errorf("project_id = %q, want %q — with no prior to preserve, server's masked value should land in state",
			prior.ProjectId.ValueString(), "fh-de*****")
	}
}

// TestApplyNLSchemaToModel_PreservesSensitiveFieldsAcrossRead: api_key,
// access_token, refresh_token, client_secret are write-only — Typesense
// doesn't echo them. applyNLSchemaToModel must not blank these on
// state from server, regardless of the server response. Same contract
// the existing comment claims; this test pins it.
func TestApplyNLSchemaToModel_PreservesSensitiveFieldsAcrossRead(t *testing.T) {
	priv := func(s string) *string { return &s }
	server := &typesense.NLSearchModel{
		Id:        "nl-search-model",
		ModelName: priv("openai/gpt-4o-mini"),
		// Server doesn't echo any of the sensitive write-only fields.
	}
	prior := NLSearchModelResourceModel{
		Id:           types.StringValue("nl-search-model"),
		ModelName:    types.StringValue("openai/gpt-4o-mini"),
		ApiKey:       types.StringValue("sk-real-openai-key"),
		AccessToken:  types.StringValue("at-real"),
		RefreshToken: types.StringValue("rt-real"),
		ClientSecret: types.StringValue("cs-real"),
	}

	applyNLSchemaToModel(server, &prior)

	cases := []struct {
		name string
		got  types.String
		want string
	}{
		{"api_key", prior.ApiKey, "sk-real-openai-key"},
		{"access_token", prior.AccessToken, "at-real"},
		{"refresh_token", prior.RefreshToken, "rt-real"},
		{"client_secret", prior.ClientSecret, "cs-real"},
	}
	for _, tc := range cases {
		if tc.got.ValueString() != tc.want {
			t.Errorf("%s = %q, want %q preserved from prior", tc.name, tc.got.ValueString(), tc.want)
		}
	}
}
