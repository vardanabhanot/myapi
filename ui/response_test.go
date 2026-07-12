package ui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSafeCut(t *testing.T) {
	if got := safeCut("hello", 10); got != "hello" {
		t.Errorf("short string changed: %q", got)
	}
	// "é" is 2 bytes; cutting at 3 lands mid-rune and must back off
	if got := safeCut("ééé", 3); got != "é" || !utf8.ValidString(got) {
		t.Errorf("cut split a rune: %q", got)
	}
}

func TestSoftWrap(t *testing.T) {
	if got := softWrap("short\nlines"); got != "short\nlines" {
		t.Errorf("short input changed: %q", got)
	}

	long := strings.Repeat("é", 500) // multi-byte runes across cut points
	wrapped := softWrap(long)
	if !utf8.ValidString(wrapped) {
		t.Error("softWrap split a multi-byte rune")
	}
	if strings.ReplaceAll(wrapped, "\n", "") != long {
		t.Error("softWrap lost or altered content")
	}
	for _, line := range strings.Split(wrapped, "\n") {
		if len(line) > 200 {
			t.Errorf("line still %d bytes long", len(line))
		}
	}
}
