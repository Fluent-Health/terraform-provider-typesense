package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"fluent-health-terraform-typesense/internal/typesense"
)

// fakeTypesense scripts the three endpoints an Update touches: the PATCH, the
// schema-change status poll, and the collection read-back.
type fakeTypesense struct {
	t *testing.T

	mu sync.Mutex
	// patchStatus holds the status for each successive PATCH (200 once exhausted).
	patchStatus []int
	patchBodies []typesense.CollectionUpdateSchema
	// busyPolls is how many schema_changes polls report the collection as altering.
	busyPolls int
	polls     int
	// live is what GET /collections/{name} returns.
	live []typesense.Field
}

func (f *fakeTypesense) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodPatch && r.URL.Path == "/collections/products":
		var body typesense.CollectionUpdateSchema
		data, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, &body); err != nil {
			f.t.Errorf("bad PATCH body: %v", err)
		}
		f.patchBodies = append(f.patchBodies, body)
		status := http.StatusOK
		if n := len(f.patchBodies); n <= len(f.patchStatus) {
			status = f.patchStatus[n-1]
		}
		w.WriteHeader(status)
		switch status {
		case http.StatusOK:
			_, _ = w.Write(data)
		case http.StatusUnprocessableEntity:
			_, _ = io.WriteString(w, `{"message":"Another collection update operation is in progress."}`)
		default:
			_, _ = io.WriteString(w, "upstream request timeout")
		}
	case r.Method == http.MethodGet && r.URL.Path == "/operations/schema_changes":
		f.polls++
		if f.polls <= f.busyPolls {
			_, _ = io.WriteString(w, `[{"collection":"products","validated_docs":100000,"altered_docs":5000,"alter_history":[]}]`)
			return
		}
		_, _ = io.WriteString(w, `[]`)
	case r.Method == http.MethodGet && r.URL.Path == "/collections/products":
		_ = json.NewEncoder(w).Encode(typesense.Collection{Name: "products", Fields: f.live})
	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func newFakeClient(t *testing.T, f *fakeTypesense, schemaChangeTimeout time.Duration) *typesense.Client {
	f.t = t
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return typesense.NewClient(srv.URL, "secret",
		typesense.WithPollInterval(time.Millisecond),
		typesense.WithSchemaChangeTimeout(schemaChangeTimeout))
}

func strPtr(s string) *string { return &s }

// liveFields is the server's view: an embedded field whose credentials come
// back masked or empty, as Typesense 30.x returns them.
func liveFields(withEmbedding bool) []typesense.Field {
	fields := []typesense.Field{{Name: "title", Type: "string"}}
	if withEmbedding {
		fields = append(fields, typesense.Field{
			Name: "embedding",
			Type: "float[]",
			Embed: &typesense.FieldEmbed{
				From: []string{"title"},
				ModelConfig: typesense.FieldEmbedModelConfig{
					ModelName: "gcp/gemini-embedding-001",
					ProjectId: strPtr("proj*****"),
					Region:    strPtr("us-central1"),
					ServiceAccount: &typesense.GCPServiceAccount{
						ClientEmail: "runner*****",
						PrivateKey:  "",
					},
				},
			},
		})
	}
	return fields
}

// fieldModels flattens the server shape the way Read does, carrying privateKey
// as the embed credential Terraform holds.
func fieldModels(fields []typesense.Field, privateKey string) []CollectionResourceFieldModel {
	prior := []CollectionResourceFieldModel{{
		Name: types.StringValue("embedding"),
		Embed: &CollectionFieldEmbedModel{ModelConfig: &CollectionFieldEmbedModelConfigModel{
			ServiceAccount: &CollectionFieldEmbedServiceAccountModel{
				ClientEmail: types.StringValue("runner@proj.iam.gserviceaccount.com"),
				PrivateKey:  types.StringValue(privateKey),
			},
		}},
	}}
	return flattenCollectionFields(fields, prior)
}

func keyRotation() (state, plan []CollectionResourceFieldModel) {
	return fieldModels(liveFields(true), "old-key"), fieldModels(liveFields(true), "new-key")
}

func TestBuildCollectionUpdateSchema_KeyRotationIsDropAndAddOfTheEmbedField(t *testing.T) {
	state, plan := keyRotation()
	schema := buildCollectionUpdateSchema(context.Background(), state, plan)

	if len(schema.Fields) != 2 {
		t.Fatalf("want drop+add of one field, got %d entries: %+v", len(schema.Fields), schema.Fields)
	}
	if schema.Fields[0].Name != "embedding" || schema.Fields[0].Drop == nil || !*schema.Fields[0].Drop {
		t.Errorf("first entry should drop embedding, got %+v", schema.Fields[0])
	}
	add := schema.Fields[1]
	if add.Name != "embedding" || add.Embed == nil || add.Embed.ModelConfig.ServiceAccount == nil ||
		add.Embed.ModelConfig.ServiceAccount.PrivateKey != "new-key" {
		t.Errorf("second entry should re-add embedding with the new key, got %+v", add)
	}
}

func TestApplyCollectionUpdate_GatewayTimeoutThenAlterDrains(t *testing.T) {
	f := &fakeTypesense{patchStatus: []int{http.StatusGatewayTimeout}, busyPolls: 4, live: liveFields(true)}
	client := newFakeClient(t, f, time.Minute)
	state, plan := keyRotation()

	err := applyCollectionUpdate(context.Background(), client, "products", buildCollectionUpdateSchema(context.Background(), state, plan), state, plan)
	if err != nil {
		t.Fatalf("want success once the alter drains, got %v", err)
	}
	if len(f.patchBodies) != 1 {
		t.Errorf("want exactly one PATCH (no re-send after a 504), got %d", len(f.patchBodies))
	}
	// 4 busy polls, then requiredIdlePolls idle ones.
	if f.polls != 4+3 {
		t.Errorf("want 7 schema_changes polls, got %d", f.polls)
	}
}

func TestApplyCollectionUpdate_GatewayTimeoutThenLiveSchemaMismatch(t *testing.T) {
	// The alter finished but the embedding field never came back.
	f := &fakeTypesense{patchStatus: []int{http.StatusGatewayTimeout}, busyPolls: 1, live: liveFields(false)}
	client := newFakeClient(t, f, time.Minute)
	state, plan := keyRotation()

	err := applyCollectionUpdate(context.Background(), client, "products", buildCollectionUpdateSchema(context.Background(), state, plan), state, plan)
	if err == nil {
		t.Fatal("want an error when the live schema does not match the plan")
	}
	if !strings.Contains(err.Error(), `field "embedding" is missing`) {
		t.Errorf("error should name the missing field, got %v", err)
	}
}

func TestApplyCollectionUpdate_GatewayTimeoutAndAlterNeverFinishes(t *testing.T) {
	f := &fakeTypesense{patchStatus: []int{http.StatusGatewayTimeout}, busyPolls: 1 << 30, live: liveFields(true)}
	client := newFakeClient(t, f, 50*time.Millisecond)
	state, plan := keyRotation()

	err := applyCollectionUpdate(context.Background(), client, "products", buildCollectionUpdateSchema(context.Background(), state, plan), state, plan)
	if err == nil || !strings.Contains(err.Error(), "still running") {
		t.Fatalf("want a still-running timeout error, got %v", err)
	}
}

func TestApplyCollectionUpdate_AlterInProgressThenRetriesOnce(t *testing.T) {
	// A credential change can't be seen in the live schema, so after the other
	// alter finishes the re-diff still wants it and the PATCH is re-sent once.
	f := &fakeTypesense{patchStatus: []int{http.StatusUnprocessableEntity, http.StatusOK}, busyPolls: 2, live: liveFields(true)}
	client := newFakeClient(t, f, time.Minute)
	state, plan := keyRotation()

	err := applyCollectionUpdate(context.Background(), client, "products", buildCollectionUpdateSchema(context.Background(), state, plan), state, plan)
	if err != nil {
		t.Fatalf("want success after one retry, got %v", err)
	}
	if len(f.patchBodies) != 2 {
		t.Fatalf("want 2 PATCHes (original + one retry), got %d", len(f.patchBodies))
	}
	if got := f.patchBodies[1].Fields; len(got) != 2 || got[1].Embed.ModelConfig.ServiceAccount.PrivateKey != "new-key" {
		t.Errorf("retry should re-send the embed field with the new key, got %+v", got)
	}
}

func TestApplyCollectionUpdate_AlterInProgressAndChangeAlreadyLive(t *testing.T) {
	// The running alter was adding the same field this plan adds (e.g. a
	// previous apply's PATCH that outlived its request). Once it finishes the
	// re-diff is empty, so nothing is re-sent.
	f := &fakeTypesense{patchStatus: []int{http.StatusUnprocessableEntity}, busyPolls: 2, live: liveFields(true)}
	client := newFakeClient(t, f, time.Minute)
	state := fieldModels(liveFields(false), "new-key")
	plan := fieldModels(liveFields(true), "new-key")

	err := applyCollectionUpdate(context.Background(), client, "products", buildCollectionUpdateSchema(context.Background(), state, plan), state, plan)
	if err != nil {
		t.Fatalf("want success with no retry, got %v", err)
	}
	if len(f.patchBodies) != 1 {
		t.Errorf("want only the original PATCH, got %d", len(f.patchBodies))
	}
}

func TestApplyCollectionUpdate_AlterInProgressTwiceFails(t *testing.T) {
	f := &fakeTypesense{patchStatus: []int{http.StatusUnprocessableEntity, http.StatusUnprocessableEntity}, busyPolls: 1, live: liveFields(true)}
	client := newFakeClient(t, f, time.Minute)
	state, plan := keyRotation()

	err := applyCollectionUpdate(context.Background(), client, "products", buildCollectionUpdateSchema(context.Background(), state, plan), state, plan)
	if err == nil || !typesense.IsAlterInProgress(err) {
		t.Fatalf("want the second 422 surfaced, got %v", err)
	}
	if len(f.patchBodies) != 2 {
		t.Errorf("want exactly one retry, got %d PATCHes", len(f.patchBodies))
	}
}

func TestApplyCollectionUpdate_OtherErrorsAreNotRetried(t *testing.T) {
	f := &fakeTypesense{patchStatus: []int{http.StatusBadRequest}, live: liveFields(true)}
	client := newFakeClient(t, f, time.Minute)
	state, plan := keyRotation()

	err := applyCollectionUpdate(context.Background(), client, "products", buildCollectionUpdateSchema(context.Background(), state, plan), state, plan)
	if err == nil {
		t.Fatal("want the 400 surfaced")
	}
	if f.polls != 0 || len(f.patchBodies) != 1 {
		t.Errorf("a 400 must not wait or retry: polls=%d patches=%d", f.polls, len(f.patchBodies))
	}
}

func TestLiveSchemaMismatches_DroppedFieldStillPresent(t *testing.T) {
	plan := fieldModels([]typesense.Field{{Name: "title", Type: "string"}}, "k")
	drop := true
	schema := &typesense.CollectionUpdateSchema{Fields: []typesense.Field{{Name: "embedding", Drop: &drop}}}

	got := liveSchemaMismatches(liveFields(true), schema, plan)
	if len(got) != 1 || !strings.Contains(got[0], `"embedding" was dropped but is still present`) {
		t.Errorf("want one dropped-but-present mismatch, got %v", got)
	}
}
