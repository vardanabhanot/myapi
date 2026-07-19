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

func TestBodyKind(t *testing.T) {
	pngHeader := "\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 20)
	cases := []struct {
		contentType, body, want string
	}{
		{"image/png", pngHeader, "image"},
		{"image/jpeg; charset=binary", "\xff\xd8\xff", "image"},
		{"image/svg+xml", "<svg/>", "text"},   // svg reads better as XML
		{"image/webp", "RIFF....WEBP", "binary"}, // Fyne can't decode webp
		{"application/json", `{"a":1}`, "text"},
		{"text/html; charset=utf-8", "<html>", "text"},
		{"application/pdf", "%PDF-1.4", "binary"},
		{"", pngHeader, "image"},                        // sniffed
		{"application/octet-stream", "plain words", "text"}, // sniffed back to text
		{"application/x-custom", "ab\x00cd", "binary"},  // NUL byte
		{"application/x-custom", "abcd", "text"},
	}
	for _, c := range cases {
		if got := bodyKind(c.contentType, c.body); got != c.want {
			t.Errorf("bodyKind(%q, %.10q) = %q, want %q", c.contentType, c.body, got, c.want)
		}
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
