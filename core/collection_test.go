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

func TestCollectionUpdateRequest(t *testing.T) {
	entry := &Request{URL: "https://old"}
	col := &Collection{Name: "c", Requests: []*Request{entry}}
	live := &Request{ID: "tab1", URL: "https://new"}

	if !col.UpdateRequest(entry, live) {
		t.Fatal("member entry not updated")
	}
	if entry.URL != "https://new" || entry.ID != "" {
		t.Fatalf("entry not synced as ID-less snapshot: %+v", entry)
	}

	// live request must not alias the entry after sync
	live.URL = "https://changed"
	if entry.URL != "https://new" {
		t.Fatal("entry aliases live request")
	}

	orphan := &Request{URL: "https://gone"}
	if col.UpdateRequest(orphan, live) {
		t.Fatal("updated an entry that is not in the collection")
	}
}
