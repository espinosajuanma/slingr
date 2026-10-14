package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientBuildsRequestAndEncodesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/runtime/api/data/orders/o-1/approve" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("token") != "secret" {
			t.Errorf("token header = %q", r.Header.Get("token"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", r.Header.Get("Content-Type"))
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["approved"] != true {
			t.Errorf("payload = %#v", payload)
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/runtime/api")
	got, err := client.Do(context.Background(), Request{
		App: "example", Env: "dev", Method: http.MethodPut,
		Path: "/data/orders/o-1/approve", Payload: map[string]any{"approved": true}, Token: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("response = %s", got)
	}
}

func TestClientUsesDefaultSlingrURL(t *testing.T) {
	client := NewClient(nil, "")
	endpoint, err := client.endpoint("my-app", "staging", "/data/holidays")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := endpoint.String(), "https://my-app.slingrs.io/staging/runtime/api/data/holidays"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}

func TestClientSurfacesAPIMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"invalid token"}`, http.StatusUnauthorized)
	}))
	defer server.Close()
	client := NewClient(server.Client(), server.URL)
	_, err := client.Do(context.Background(), Request{
		App: "example", Env: "dev", Method: http.MethodGet, Path: "/resource", Token: "bad",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("error = %v, want API message", err)
	}
}
