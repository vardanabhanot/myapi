package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestDetectAndFormat(t *testing.T) {
	cases := []struct{ ct, body, want string }{
		{"application/json; charset=utf-8", "", "json"},
		{"text/html", "", "html"},
		{"application/xml", "", "xml"},
		{"", `{"a":1}`, "json"},
		{"", "<!DOCTYPE html><html>", "html"},
		{"", "<note/>", "xml"},
		{"text/plain", "hello", ""},
	}
	for _, c := range cases {
		if got := detectLang(c.ct, c.body); got != c.want {
			t.Errorf("detectLang(%q, %q) = %q, want %q", c.ct, c.body, got, c.want)
		}
	}

	if got := formatBody(`{"a":1}`, "json"); got != "{\n  \"a\": 1\n}" {
		t.Errorf("formatBody json = %q", got)
	}
	if got := formatBody("not json", "json"); got != "not json" {
		t.Errorf("invalid json should pass through, got %q", got)
	}

	xmlGot := formatBody(`<a><b>hi</b><c attr="1"/></a>`, "xml")
	if !strings.Contains(xmlGot, "\n") || !strings.Contains(xmlGot, "  <b>") {
		t.Errorf("minified xml should be indented, got %q", xmlGot)
	}
	if again := formatBody(xmlGot, "xml"); again != xmlGot {
		t.Errorf("indenting should be idempotent, got %q", again)
	}

	htmlGot := formatBody(`<html><body><p>hi<br>there</p></body></html>`, "html")
	if !strings.Contains(htmlGot, "\n") || !strings.Contains(htmlGot, "    <p>") {
		t.Errorf("minified html should be indented, got %q", htmlGot)
	}
	if !strings.Contains(htmlGot, "      hi") || !strings.Contains(htmlGot, "      there") {
		t.Errorf("void <br> must not increase depth, got %q", htmlGot)
	}

	// Script bodies with raw "<" killed the old xml-based formatter
	scriptGot := formatBody(`<html><script>if(a<b){go()}</script></html>`, "html")
	if !strings.Contains(scriptGot, "\n") || !strings.Contains(scriptGot, "if(a<b){go()}") {
		t.Errorf("html with script should still indent, got %q", scriptGot)
	}
}

func TestHighlightGridRows(t *testing.T) {
	test.NewApp() // theme.Color needs a running app
	rows := highlightGridRows("{\n  \"key\": \"value\"\n}", "json")
	if rows == nil {
		t.Fatal("expected highlighted rows for json")
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}
	// key and value should get different colors
	var keyStyle, valStyle interface{}
	for _, cell := range rows[1].Cells {
		if cell.Rune == 'k' {
			keyStyle = cell.Style
		}
		if cell.Rune == 'a' {
			valStyle = cell.Style
		}
	}
	if keyStyle == nil || valStyle == nil || keyStyle == valStyle {
		t.Errorf("key and value styles should differ: %v vs %v", keyStyle, valStyle)
	}

	if highlightGridRows("plain text", "") != nil {
		t.Error("unknown lang should return nil")
	}
}
