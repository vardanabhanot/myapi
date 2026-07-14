package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApplyEnv(t *testing.T) {
	env := &Environment{Name: "dev", Variables: &[]FormType{
		{Checked: true, Key: "host", Value: "https://api.example.com"},
		{Checked: true, Key: "token", Value: "abc123"},
		{Checked: false, Key: "off", Value: "nope"},
	}}

	SetActiveVars(env.VarMap())
	defer SetActiveVars(nil)

	got := ApplyEnv("{{host}}/users?t={{token}}&o={{off}}&m={{missing}}")
	want := "https://api.example.com/users?t=abc123&o={{off}}&m={{missing}}"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	if got := ApplyEnv("no vars here"); got != "no vars here" {
		t.Fatalf("plain string changed: %q", got)
	}

	// nil-safe chain used by the UI when no env is active
	if vars := (&EnvStore{}).ActiveEnv().VarMap(); len(vars) != 0 {
		t.Fatalf("expected empty var map, got %v", vars)
	}
}

// End-to-end: {{var}} must reach the wire substituted, while the saved
// request keeps its placeholders.
func TestSendRequestAppliesEnv(t *testing.T) {
	var gotQuery, gotHeader, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("token")
		gotHeader = r.Header.Get("X-Api-Key")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer server.Close()

	env := &Environment{Name: "test", Variables: &[]FormType{
		{Checked: true, Key: "base", Value: server.URL},
		{Checked: true, Key: "key", Value: "s3cret"},
	}}
	SetActiveVars(env.VarMap())
	defer SetActiveVars(nil)

	req := &Request{
		ID:       "envtest",
		Method:   "POST",
		URL:      "{{base}}/things?token={{key}}",
		Headers:  &[]FormType{{Checked: true, Key: "X-Api-Key", Value: "{{key}}"}},
		BodyType: "JSON",
		Body:     Body{Json: `{"k":"{{key}}"}`},
	}
	defer DeleteHistory("envtest.json") // SendRequest saves into history

	if _, err := req.SendRequest(context.Background()); err != nil {
		t.Fatal(err)
	}

	if gotQuery != "s3cret" || gotHeader != "s3cret" || gotBody != `{"k":"s3cret"}` {
		t.Fatalf("substitution missed: query=%q header=%q body=%q", gotQuery, gotHeader, gotBody)
	}

	saved, err := LoadRequest("envtest.json")
	if err != nil {
		t.Fatal(err)
	}
	if saved.URL != "{{base}}/things?token={{key}}" {
		t.Fatalf("saved request was mutated: %q", saved.URL)
	}
}
