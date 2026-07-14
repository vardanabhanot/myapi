package ui

import (
	"testing"

	"fyne.io/fyne/v2/widget"
)

func gridRows(lines ...string) []widget.TextGridRow {
	rows := make([]widget.TextGridRow, len(lines))
	for i, line := range lines {
		for _, r := range line {
			rows[i].Cells = append(rows[i].Cells, widget.TextGridCell{Rune: r})
		}
	}
	return rows
}

func TestFindMatches(t *testing.T) {
	rows := gridRows(
		`{"name": "Ada"}`,
		`{"NAME": "ada lovelace"}`,
		"no hits here",
	)

	got := findMatches(rows, "name")
	want := [][2]int{{0, 2}, {1, 2}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("case-insensitive match: got %v, want %v", got, want)
	}

	// non-overlapping: "aa" in "aaaa" is 2 matches, not 3
	if got := findMatches(gridRows("aaaa"), "aa"); len(got) != 2 {
		t.Fatalf("overlap handling: got %v", got)
	}

	if got := findMatches(rows, "zzz"); len(got) != 0 {
		t.Fatalf("no-hit query returned %v", got)
	}

	// query longer than a row must not panic or match
	if got := findMatches(gridRows("ab"), "abcdef"); len(got) != 0 {
		t.Fatalf("long query returned %v", got)
	}
}

func TestSaveFileName(t *testing.T) {
	headers := []string{"Content-Type||application/json; charset=utf-8"}

	if got := saveFileName("https://api.example.com/users", headers); got != "users.json" {
		t.Errorf("got %q, want users.json", got)
	}
	// existing extension wins over Content-Type
	if got := saveFileName("https://example.com/data.csv", headers); got != "data.csv" {
		t.Errorf("got %q, want data.csv", got)
	}
	// no path, no headers → generic fallback
	if got := saveFileName("https://example.com", nil); got != "response.txt" {
		t.Errorf("got %q, want response.txt", got)
	}
}
