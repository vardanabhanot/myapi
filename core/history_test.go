package core

import (
	"strings"
	"testing"
)

// Round-trip: save → list (bare IDs, no ".json") → load → delete.
func TestHistoryRoundTrip(t *testing.T) {
	req := &Request{ID: "histroundtrip", Method: "POST", URL: "https://example.com/x"}
	if _, err := saveRequestData(req); err != nil {
		t.Fatal(err)
	}
	defer DeleteHistory("histroundtrip")

	var entry *HistoryEntry
	for _, e := range ListHistory() {
		if strings.Contains(e.ID, ".json") {
			t.Fatalf("ID leaks filename suffix: %q", e.ID)
		}
		if e.ID == "histroundtrip" {
			entry = e
		}
	}
	if entry == nil {
		t.Fatal("saved request missing from ListHistory")
	}

	entry.LoadMeta()
	if !entry.Loaded || entry.URL != "https://example.com/x" || entry.Method != "POST" || entry.MTime == "" {
		t.Fatalf("LoadMeta: %+v", entry)
	}

	loaded, err := LoadRequest("histroundtrip")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.URL != req.URL {
		t.Fatalf("loaded URL %q, want %q", loaded.URL, req.URL)
	}

	if err := DeleteHistory("histroundtrip"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRequest("histroundtrip"); err == nil {
		t.Fatal("LoadRequest should fail after delete")
	}
}
