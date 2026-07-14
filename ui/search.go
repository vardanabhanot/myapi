package ui

import (
	"fmt"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// searchEntry is a plain Entry that also reports Escape.
type searchEntry struct {
	widget.Entry
	onEsc func()
}

func newSearchEntry() *searchEntry {
	e := &searchEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *searchEntry) TypedKey(k *fyne.KeyEvent) {
	if k.Name == fyne.KeyEscape && e.onEsc != nil {
		e.onEsc()
		return
	}
	e.Entry.TypedKey(k)
}

// maxSearchMatches caps highlighting work on pathological queries ("e" in a
// minified body); the count label gets a "+" when the cap is hit.
const maxSearchMatches = 1000

type styledCell struct {
	row, col int
	prev     widget.TextGridStyle
}

// responseSearch is the Ctrl+F bar over a response TextGrid. It searches the
// DISPLAYED rows (soft-wrapped, display-capped), so what it finds is exactly
// what is on screen.
type responseSearch struct {
	bar     fyne.CanvasObject
	grid    *widget.TextGrid
	canvas  fyne.Canvas
	entry   *searchEntry
	label   *widget.Label
	matches [][2]int // {row, first cell} per match
	qlen    int      // query length in runes
	current int
	styled  []styledCell // cells we recoloured, with their previous style
}

func newResponseSearch(grid *widget.TextGrid, canvas fyne.Canvas) *responseSearch {
	s := &responseSearch{grid: grid, canvas: canvas}

	s.entry = newSearchEntry()
	s.entry.SetPlaceHolder("Search response")
	s.entry.OnChanged = func(string) { s.run() }
	s.entry.OnSubmitted = func(string) { s.step(1) }
	s.entry.onEsc = s.hide

	s.label = widget.NewLabel("")

	prevBtn := widget.NewButtonWithIcon("", theme.MenuDropUpIcon(), func() { s.step(-1) })
	prevBtn.Importance = widget.LowImportance
	nextBtn := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), func() { s.step(1) })
	nextBtn.Importance = widget.LowImportance
	closeBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), s.hide)
	closeBtn.Importance = widget.LowImportance

	s.bar = container.NewBorder(nil, nil, nil, container.NewHBox(s.label, prevBtn, nextBtn, closeBtn), s.entry)
	s.bar.Hide()

	return s
}

func (s *responseSearch) show() {
	s.bar.Show()
	s.canvas.Focus(s.entry)
	if s.entry.Text != "" {
		s.run()
	}
}

func (s *responseSearch) hide() {
	s.bar.Hide()
	s.clearStyles()
	s.matches = s.matches[:0]
	s.label.SetText("")
	s.grid.Refresh()
	s.canvas.Unfocus()
}

// contentChanged must be called whenever the grid's rows are rebuilt: styled
// cells and match positions point into the old rows, so drop them without
// restoring, then re-search if the bar is open.
func (s *responseSearch) contentChanged() {
	s.styled = s.styled[:0]
	s.matches = s.matches[:0]
	if s.bar.Visible() && s.entry.Text != "" {
		s.run()
	} else {
		s.label.SetText("")
	}
}

func (s *responseSearch) run() {
	s.clearStyles()
	s.matches = s.matches[:0]

	s.qlen = len([]rune(s.entry.Text))
	if s.qlen == 0 {
		s.label.SetText("")
		s.grid.Refresh()
		return
	}

	s.matches = findMatches(s.grid.Rows, s.entry.Text)
	s.current = 0
	s.applyStyles()
	s.updateLabel()
	if len(s.matches) > 0 {
		s.scrollTo(s.matches[0][0])
	}
	s.grid.Refresh()
}

// findMatches scans grid rows for case-insensitive, non-overlapping matches
// of query, returning {row, first cell} pairs capped at maxSearchMatches.
// ponytail: per-row scan, so a match spanning a soft-wrap boundary
// (softWrapCols) is missed; search the unwrapped string and map offsets to
// rows if that ever matters.
func findMatches(rows []widget.TextGridRow, query string) [][2]int {
	q := []rune(strings.ToLower(query))
	var matches [][2]int

	for ri := range rows {
		cells := rows[ri].Cells
		for ci := 0; ci+len(q) <= len(cells); ci++ {
			hit := true
			for k, qr := range q {
				if unicode.ToLower(cells[ci+k].Rune) != qr {
					hit = false
					break
				}
			}
			if !hit {
				continue
			}
			matches = append(matches, [2]int{ri, ci})
			if len(matches) == maxSearchMatches {
				return matches
			}
			ci += len(q) - 1
		}
	}

	return matches
}

// step moves the current match by delta, wrapping around.
func (s *responseSearch) step(delta int) {
	if len(s.matches) == 0 {
		return
	}
	s.clearStyles()
	s.current = (s.current + delta + len(s.matches)) % len(s.matches)
	s.applyStyles()
	s.updateLabel()
	s.scrollTo(s.matches[s.current][0])
	s.grid.Refresh()
}

func (s *responseSearch) updateLabel() {
	text := fmt.Sprintf("%d/%d", s.current+1, len(s.matches))
	if len(s.matches) == 0 {
		text = "0/0"
	} else if len(s.matches) == maxSearchMatches {
		text += "+"
	}
	s.label.SetText(text)
}

// applyStyles recolours every match, keeping each cell's syntax colour; the
// current match gets inverted emphasis instead.
func (s *responseSearch) applyStyles() {
	matchBG := theme.Color(theme.ColorNameSelection)
	currentBG := theme.Color(theme.ColorNamePrimary)
	currentFG := theme.Color(theme.ColorNameBackground)

	for mi, m := range s.matches {
		for c := m[1]; c < m[1]+s.qlen; c++ {
			cell := &s.grid.Rows[m[0]].Cells[c]
			st := &widget.CustomTextGridStyle{BGColor: matchBG}
			if cell.Style != nil {
				st.FGColor = cell.Style.TextColor()
			}
			if mi == s.current {
				st.BGColor = currentBG
				st.FGColor = currentFG
			}
			s.styled = append(s.styled, styledCell{m[0], c, cell.Style})
			cell.Style = st
		}
	}
}

// clearStyles restores the original style of every recoloured cell. Bounds
// are re-checked because the rows may have been replaced since.
func (s *responseSearch) clearStyles() {
	for _, sc := range s.styled {
		if sc.row < len(s.grid.Rows) && sc.col < len(s.grid.Rows[sc.row].Cells) {
			s.grid.Rows[sc.row].Cells[sc.col].Style = sc.prev
		}
	}
	s.styled = s.styled[:0]
}

// scrollTo brings row (plus two rows of context above) to the top of the
// grid's viewport. TextGrid keeps its scroller unexported, but the renderer's
// first object implements fyne.Scrollable, so we drive it with a synthetic
// wheel event — negative DY scrolls down, and the scroller clamps for us.
// test.WidgetRenderer is only a renderer-cache lookup, safe outside tests.
// ponytail: swap for a public TextGrid scroll API when fyne grows one.
func (s *responseSearch) scrollTo(row int) {
	row -= 2
	if row < 0 {
		row = 0
	}
	target := s.grid.PositionForCursorLocation(row, 0) // viewport-relative
	if sc, ok := test.WidgetRenderer(s.grid).Objects()[0].(fyne.Scrollable); ok {
		sc.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: -target.Y}})
	}
}
