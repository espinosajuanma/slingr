package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/espinosajuanma/slingr/internal/api"
)

type memoryStore struct {
	values map[string]string
}

func (s *memoryStore) Load() (map[string]string, error) {
	return cloneConfig(s.values), nil
}

func (s *memoryStore) Save(values map[string]string) error {
	s.values = cloneConfig(values)
	return nil
}

func cloneConfig(values map[string]string) map[string]string {
	copy := make(map[string]string, len(values))
	for key, value := range values {
		copy[key] = value
	}
	return copy
}

func newTestApp(t *testing.T, store *memoryStore, client API, completionLine string) (*App, *strings.Builder) {
	return newTestAppWithInput(t, store, client, completionLine, strings.NewReader(""))
}

func newTestAppWithInput(t *testing.T, store *memoryStore, client API, completionLine string, input io.Reader) (*App, *strings.Builder) {
	t.Helper()
	output := &strings.Builder{}
	application, err := New(Options{
		Name:           "solutions",
		Store:          store,
		API:            client,
		Input:          input,
		Output:         output,
		ErrorOutput:    &strings.Builder{},
		CompletionLine: completionLine,
	})
	if err != nil {
		t.Fatal(err)
	}
	return application, output
}

func TestConfigCommandIsGenericAndPersistsValues(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	application, output := newTestApp(t, store, api.NewClient(nil, ""), "")
	if err := application.Run(context.Background(), []string{"config", "custom.key", "one", "value with spaces"}); err != nil {
		t.Fatal(err)
	}
	if got, want := store.values["custom.key"], "one value with spaces"; got != want {
		t.Fatalf("saved value = %q, want %q", got, want)
	}
	if err := application.Run(context.Background(), []string{"config", "custom.key"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "one value with spaces\n" {
		t.Fatalf("config read output = %q", got)
	}
}

func TestSetupWizardRetainsExistingValuesWhenInputIsEmpty(t *testing.T) {
	store := &memoryStore{values: map[string]string{
		"app": "existing-app", "env": "staging",
	}}
	application, output := newTestAppWithInput(
		t,
		store,
		api.NewClient(nil, ""),
		"",
		strings.NewReader("\n\nada@example.com\n"),
	)
	if err := application.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if store.values["app"] != "existing-app" || store.values["env"] != "staging" {
		t.Fatalf("wizard replaced existing values: %#v", store.values)
	}
	if store.values["email"] != "ada@example.com" {
		t.Fatalf("wizard email = %q", store.values["email"])
	}
	if !strings.Contains(output.String(), "App [existing-app]: Environment (dev|staging|prod) [staging]:") {
		t.Fatalf("wizard did not display existing defaults: %q", output.String())
	}
	if strings.Contains(output.String(), "Usage:") {
		t.Fatalf("incomplete configuration displayed usage instead of setup: %q", output.String())
	}
}

func TestConfiguredNoArgsShowsUsage(t *testing.T) {
	store := &memoryStore{values: map[string]string{
		"app": "existing-app", "env": "staging", "email": "ada@example.com",
	}}
	application, output := newTestApp(t, store, api.NewClient(nil, ""), "")
	if err := application.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(output.String(), "Usage: solutions {") {
		t.Fatalf("usage output = %q", output.String())
	}
}

func TestFlagParsingSupportsNestedObjectsAndJSONValues(t *testing.T) {
	got, err := parseFlags([]string{
		"--message", "hello world",
		"--daySettings.date", "2026-10-14",
		"--enabled", "true",
		"--count", "12",
		`--metadata={"source":"cli","tags":["a",2]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["message"] != "hello world" || got["enabled"] != true || got["count"].(json.Number).String() != "12" {
		t.Fatalf("unexpected parsed flags: %#v", got)
	}
	settings := got["daySettings"].(map[string]any)
	if settings["date"] != "2026-10-14" {
		t.Fatalf("nested date = %#v", settings["date"])
	}
	metadata := got["metadata"].(map[string]any)
	if metadata["source"] != "cli" {
		t.Fatalf("metadata.source = %#v", metadata["source"])
	}
	queryValues, err := parseFlagValues([]string{"--daySettings.date", "2026-10-14"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := queryValues["daySettings.date"]; !ok {
		t.Fatalf("dotted query key wasn't kept flat: %#v", queryValues)
	}
}

func TestDataCommandsAndDynamicActions(t *testing.T) {
	var editPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("token") != "test-token" {
			t.Errorf("token header = %q", r.Header.Get("token"))
		}
		if r.URL.Path == "/runtime/api/data/holidays" && r.Method == http.MethodGet {
			if r.URL.Query().Get("country") != "Argentina" || r.URL.Query().Get("active") != "true" || r.URL.Query().Get("daySettings.date") != "2026-10-14" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			io.WriteString(w, `[{"id":"r1"}]`)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /runtime/api/data/holidays":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if payload["enabled"] != true || payload["count"].(float64) != 3 {
				t.Errorf("unexpected create payload: %#v", payload)
			}
			io.WriteString(w, `{"id":"created"}`)
		case "GET /runtime/api/data/holidays/r1":
			io.WriteString(w, `{"country":"Argentina","daySettings":{"date":"old","zone":"UTC"},"active":true}`)
		case "PUT /runtime/api/data/holidays/r1":
			if err := json.NewDecoder(r.Body).Decode(&editPayload); err != nil {
				t.Error(err)
			}
			io.WriteString(w, `{"updated":true}`)
		case "DELETE /runtime/api/data/holidays/r1":
			io.WriteString(w, `{"deleted":true}`)
		case "PUT /runtime/api/data/holidays/approve":
			io.WriteString(w, `{}`)
		case "PUT /runtime/api/data/holidays/r1/approve":
			io.WriteString(w, `{}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &memoryStore{values: map[string]string{
		"app": "example", "env": "dev", "email": "user@example.com", "example.dev.token": "test-token",
	}}
	client := api.NewClient(server.Client(), server.URL+"/runtime/api")
	application, _ := newTestApp(t, store, client, "")
	commands := [][]string{
		{"query", "holidays", "--country", "Argentina", "--active", "true", "--daySettings.date", "2026-10-14"},
		{"new", "holidays", "--enabled", "true", "--count", "3"},
		{"get", "holidays", "r1"},
		{"edit", "holidays", "r1", "--daySettings.date", "new-date"},
		{"remove", "holidays", "r1"},
		{"action", "holidays", "approve", "--country", "USA"},
		{"action", "holidays", "approve", "r1", "--country", "USA"},
	}
	for _, args := range commands {
		if err := application.Run(context.Background(), args); err != nil {
			t.Errorf("run %v: %v", args, err)
		}
	}
	settings := editPayload["daySettings"].(map[string]any)
	if settings["date"] != "new-date" || settings["zone"] != "UTC" {
		t.Fatalf("edit did not preserve nested fields: %#v", editPayload)
	}
	if editPayload["country"] != "Argentina" || editPayload["active"] != true {
		t.Fatalf("edit did not preserve existing fields: %#v", editPayload)
	}
}

func TestLoginTokenAndLogout(t *testing.T) {
	var loginPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/runtime/api/auth/login":
			if err := json.NewDecoder(r.Body).Decode(&loginPayload); err != nil {
				t.Error(err)
			}
			io.WriteString(w, `{"token":"new-token","userName":"Ada"}`)
		case "/runtime/api/auth/logout":
			if r.Header.Get("token") != "new-token" {
				t.Errorf("logout token = %q", r.Header.Get("token"))
			}
			io.WriteString(w, `{}`)
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &memoryStore{values: map[string]string{
		"app": "example", "env": "staging", "email": "ada@example.com",
	}}
	application, _ := newTestApp(t, store, api.NewClient(server.Client(), server.URL+"/runtime/api"), "")
	if err := application.Run(context.Background(), []string{"login", "secret", "password"}); err != nil {
		t.Fatal(err)
	}
	if loginPayload["password"] != "secret password" || loginPayload["email"] != "ada@example.com" {
		t.Fatalf("login payload = %#v", loginPayload)
	}
	if store.values["example.staging.token"] != "new-token" || store.values["example.staging.person"] != "Ada" {
		t.Fatalf("login settings = %#v", store.values)
	}
	if err := application.Run(context.Background(), []string{"logout"}); err != nil {
		t.Fatal(err)
	}
	if _, exists := store.values["example.staging.token"]; exists {
		t.Fatal("logout did not remove saved token")
	}
}

func TestBashCompletion(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	application, output := newTestApp(t, store, api.NewClient(nil, ""), "solutions lo")
	if err := application.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "login\nlogout\n"; got != want {
		t.Fatalf("completion = %q, want %q", got, want)
	}
}
