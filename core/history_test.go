package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateHistoryFiles(t *testing.T) {
	oldDir, newDir := t.TempDir(), t.TempDir()

	write := func(dir, name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(oldDir, "111.json", "old")
	write(oldDir, "222.json", "moved")
	write(oldDir, "desktop.ini", "stray")
	write(newDir, "111.json", "existing") // must not be clobbered

	migrateHistoryFiles(oldDir, newDir)

	if got, _ := os.ReadFile(filepath.Join(newDir, "111.json")); string(got) != "existing" {
		t.Fatalf("clobbered existing entry: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(newDir, "222.json")); string(got) != "moved" {
		t.Fatalf("222.json not migrated: %q", got)
	}
	if _, err := os.Stat(filepath.Join(oldDir, "222.json")); err == nil {
		t.Fatal("222.json left behind in old dir")
	}
	if _, err := os.Stat(filepath.Join(oldDir, "desktop.ini")); err != nil {
		t.Fatal("stray file should stay put")
	}
}

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
