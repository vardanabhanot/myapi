package ui

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"golang.org/x/net/html"
)

const softWrapCols = 200

// detectLang picks a chroma lexer name from the Content-Type header,
// falling back to sniffing the body's first character.
func detectLang(contentType, body string) string {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "json"):
		return "json"
	case strings.Contains(ct, "html"):
		return "html"
	case strings.Contains(ct, "xml"):
		return "xml"
	case strings.Contains(ct, "javascript"):
		return "javascript"
	case strings.Contains(ct, "css"):
		return "css"
	}

	trimmed := strings.TrimSpace(body)
	switch {
	case strings.HasPrefix(trimmed, "{"), strings.HasPrefix(trimmed, "["):
		return "json"
	case strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html"), strings.HasPrefix(strings.ToLower(trimmed), "<html"):
		return "html"
	case strings.HasPrefix(trimmed, "<"):
		return "xml"
	}
	return ""
}

// formatBody pretty-prints the body where the stdlib can do it.
// Anything it can't parse is returned unchanged.
func formatBody(body, lang string) string {
	switch lang {
	case "json":
		var out bytes.Buffer
		if json.Indent(&out, []byte(body), "", "  ") == nil {
			return out.String()
		}
	case "xml":
		return indentMarkup(body)
	case "html":
		return indentHTML(body)
	}
	return body
}

// indentHTML re-indents HTML with the x/net/html tokenizer, which is
// forgiving: scripts with raw "<", unclosed tags, and bogus markup all
// tokenize fine. Display only; copy always uses the original body.
func indentHTML(body string) string {
	voidTags := map[string]bool{
		"area": true, "base": true, "br": true, "col": true, "embed": true,
		"hr": true, "img": true, "input": true, "link": true, "meta": true,
		"param": true, "source": true, "track": true, "wbr": true,
	}

	z := html.NewTokenizer(strings.NewReader(body))
	var b strings.Builder
	depth := 0
	writeLine := func(s string) {
		for range depth {
			b.WriteString("  ")
		}
		b.WriteString(s)
		b.WriteByte('\n')
	}

	for {
		switch z.Next() {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return strings.TrimRight(b.String(), "\n")
			}
			return body

		case html.TextToken:
			// Script/style bodies come through here too; one line each,
			// original inner indentation dropped
			for _, line := range strings.Split(string(z.Raw()), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					writeLine(line)
				}
			}

		case html.StartTagToken:
			writeLine(string(z.Raw()))
			name, _ := z.TagName()
			if !voidTags[string(name)] {
				depth++
			}

		case html.EndTagToken:
			if depth > 0 {
				depth--
			}
			writeLine(string(z.Raw()))

		default: // self-closing tags, comments, doctype
			writeLine(string(z.Raw()))
		}
	}
}

// indentMarkup re-indents XML via a lenient stdlib xml round-trip.
// Display only: entities get re-encoded, which is fine because copy
// always uses the original body.
func indentMarkup(body string) string {
	var out bytes.Buffer
	dec := xml.NewDecoder(strings.NewReader(body))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	enc := xml.NewEncoder(&out)
	enc.Indent("", "  ")

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return body
		}

		// Drop whitespace-only text nodes so existing formatting
		// doesn't fight the new indentation
		if cd, ok := tok.(xml.CharData); ok && len(bytes.TrimSpace(cd)) == 0 {
			continue
		}

		if err := enc.EncodeToken(tok); err != nil {
			return body
		}
	}

	if err := enc.Flush(); err != nil {
		return body
	}

	return out.String()
}

// highlightGridRows tokenizes body with chroma and builds TextGrid rows with
// per-token colors, soft-wrapping at softWrapCols (see softWrap for why).
// Returns nil when highlighting isn't possible; caller falls back to plain text.
func highlightGridRows(body, lang string) []widget.TextGridRow {
	// ponytail: regex lexing a huge minified body can take seconds; plain
	// text beyond this cap. Raise if users complain, or lex asynchronously.
	const maxHighlightBytes = 512 * 1024
	if lang == "" || len(body) > maxHighlightBytes {
		return nil
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		return nil
	}
	iterator, err := chroma.Coalesce(lexer).Tokenise(nil, body)
	if err != nil {
		return nil
	}

	styles := map[chroma.TokenType]*widget.CustomTextGridStyle{}
	styleFor := func(tt chroma.TokenType) *widget.CustomTextGridStyle {
		if s, ok := styles[tt]; ok {
			return s
		}
		name := theme.ColorNameForeground
		switch {
		case tt == chroma.NameTag: // JSON keys, HTML/XML tags
			name = theme.ColorNamePrimary
		case tt == chroma.NameAttribute:
			name = theme.ColorNameHyperlink
		case tt.InCategory(chroma.LiteralString):
			name = theme.ColorNameSuccess
		case tt.InCategory(chroma.LiteralNumber), tt.InCategory(chroma.Keyword):
			name = theme.ColorNameWarning
		case tt.InCategory(chroma.Comment):
			name = theme.ColorNameDisabled
		}
		s := &widget.CustomTextGridStyle{FGColor: theme.Color(name)}
		styles[tt] = s
		return s
	}

	var rows []widget.TextGridRow
	var cur []widget.TextGridCell
	for _, tok := range iterator.Tokens() {
		style := styleFor(tok.Type)
		for _, r := range tok.Value {
			switch r {
			case '\r':
				continue
			case '\n':
				rows = append(rows, widget.TextGridRow{Cells: cur})
				cur = nil
				continue
			}
			if len(cur) >= softWrapCols {
				rows = append(rows, widget.TextGridRow{Cells: cur})
				cur = nil
			}
			cur = append(cur, widget.TextGridCell{Rune: r, Style: style})
		}
	}
	rows = append(rows, widget.TextGridRow{Cells: cur})
	return rows
}
