package core

import "testing"

// A clone that aliases the original would let tab edits silently rewrite
// collection snapshots.
func TestRequestClone(t *testing.T) {
	orig := &Request{
		ID:      "orig",
		URL:     "https://a",
		Headers: &[]FormType{{Checked: true, Key: "X", Value: "1"}},
		Auth:    &Auth{BearerAuth: "tok"},
		Body:    Body{Form: &[]FormType{{Key: "f"}}},
	}

	clone := orig.Clone()

	(*orig.Headers)[0].Value = "changed"
	orig.Auth.BearerAuth = "changed"
	(*orig.Body.Form)[0].Key = "changed"

	if (*clone.Headers)[0].Value != "1" || clone.Auth.BearerAuth != "tok" || (*clone.Body.Form)[0].Key != "f" {
		t.Fatalf("clone aliases original: %+v", clone)
	}

	if clone.ID != "" {
		t.Fatalf("clone kept ID %q", clone.ID)
	}
}
