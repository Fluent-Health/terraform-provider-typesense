package typesense

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetCollection_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/movies" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-TYPESENSE-API-KEY") != "secret" {
			t.Errorf("missing or wrong api key header: %q", r.Header.Get("X-TYPESENSE-API-KEY"))
		}
		_, _ = io.WriteString(w, `{"name":"movies","fields":[{"name":"title","type":"string"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	got, err := c.GetCollection(context.Background(), "movies")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "movies" || len(got.Fields) != 1 || got.Fields[0].Name != "title" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestClient_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	_, err := c.GetCollection(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected IsNotFound to be true, got %v", err)
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if ae.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", ae.StatusCode)
	}
}

func TestClient_ErrorMessageFromBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"Field `+"`tags`"+` must be a string array"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	_, err := c.CreateCollection(context.Background(), &CollectionCreateSchema{Name: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must be a string array") {
		t.Errorf("expected parsed message in error, got %q", err.Error())
	}
}
