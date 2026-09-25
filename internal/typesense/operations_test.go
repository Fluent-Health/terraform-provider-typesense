package typesense

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_GetSchemaChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/operations/schema_changes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"collection":"products","validated_docs":100000,"altered_docs":12000,"alter_history":[]}]`)
	}))
	defer srv.Close()

	got, err := NewClient(srv.URL, "secret").GetSchemaChanges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Collection != "products" || got[0].AlteredDocs != 12000 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestClient_WaitForSchemaChange_IgnoresOtherCollectionsAndPollErrors(t *testing.T) {
	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		polls++
		switch polls {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			_, _ = io.WriteString(w, `[{"collection":"products"}]`)
		default:
			_, _ = io.WriteString(w, `[{"collection":"orders"}]`)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret", WithPollInterval(time.Millisecond))
	if err := c.WaitForSchemaChange(context.Background(), "products"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if polls != 2+requiredIdlePolls {
		t.Errorf("want %d polls, got %d", 2+requiredIdlePolls, polls)
	}
}

func TestClient_WaitForSchemaChange_HonoursContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"collection":"products"}]`)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	c := NewClient(srv.URL, "secret", WithPollInterval(time.Millisecond))
	err := c.WaitForSchemaChange(ctx, "products")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestIsRequestOutlived(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"504", &APIError{StatusCode: http.StatusGatewayTimeout}, true},
		{"408", &APIError{StatusCode: http.StatusRequestTimeout}, true},
		{"502", &APIError{StatusCode: http.StatusBadGateway}, true},
		{"400", &APIError{StatusCode: http.StatusBadRequest}, false},
		{"422", &APIError{StatusCode: http.StatusUnprocessableEntity}, false},
		{"canceled", context.Canceled, false},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"nil", nil, false},
	} {
		if got := IsRequestOutlived(tc.err); got != tc.want {
			t.Errorf("%s: got %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestIsRequestOutlived_DroppedConnectionAndClientTimeout(t *testing.T) {
	drop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		_ = conn.Close()
	}))
	defer drop.Close()
	_, err := NewClient(drop.URL, "secret").UpdateCollection(context.Background(), "products", &CollectionUpdateSchema{})
	if !IsRequestOutlived(err) {
		t.Errorf("dropped connection should count as outlived, got %v", err)
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	}))
	defer slow.Close()
	_, err = NewClient(slow.URL, "secret", WithRequestTimeout(10*time.Millisecond)).UpdateCollection(context.Background(), "products", &CollectionUpdateSchema{})
	if !IsRequestOutlived(err) {
		t.Errorf("client timeout should count as outlived, got %v", err)
	}
}

func TestIsAlterInProgress(t *testing.T) {
	if !IsAlterInProgress(&APIError{StatusCode: 422, Message: "Another collection update operation is in progress."}) {
		t.Error("want true for the concurrent-alter 422")
	}
	if IsAlterInProgress(&APIError{StatusCode: 422, Message: "Skipping writes."}) {
		t.Error("want false for an unrelated 422")
	}
}
