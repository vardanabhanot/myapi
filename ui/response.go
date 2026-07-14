package ui

import (
	"image/color"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

// saveFileName suggests a download name: the URL's last path segment, with
// an extension from Content-Type when the segment doesn't already carry one.
// Deliberate switch instead of mime.ExtensionsByType: on Windows that reads
// the registry and can return junk like ".bat" for text/plain.
func saveFileName(rawURL string, headers []string) string {
	name := "response"
	if u, err := url.Parse(rawURL); err == nil {
		if base := path.Base(u.Path); base != "." && base != "/" {
			name = base
		}
	}

	if strings.Contains(name, ".") {
		return name
	}

	ext := ".txt"
	for _, h := range headers {
		k, v, _ := strings.Cut(h, "||")
		if !strings.EqualFold(k, "Content-Type") {
			continue
		}

		ct, _, _ := strings.Cut(v, ";")
		switch strings.TrimSpace(ct) {
		case "application/json":
			ext = ".json"
		case "application/xml", "text/xml":
			ext = ".xml"
		case "text/html":
			ext = ".html"
		case "text/css":
			ext = ".css"
		case "application/javascript", "text/javascript":
			ext = ".js"
		}
		break
	}

	return name + ext
}

// safeCut truncates s to at most max bytes without splitting a rune.
func safeCut(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

// softWrap breaks lines longer than max characters. TextGrid allocates
// 3 canvas objects per column of the *longest* line for every visible row,
// so a single minified 50k-char HTML/JSON line freezes the UI.
func softWrap(s string) string {
	const max = softWrapCols
	if len(s) <= max {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/max + 1)
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		for len(line) > max {
			cut := max
			for cut > max-utf8.UTFMax && !utf8.RuneStart(line[cut]) {
				cut-- // don't split a multi-byte rune
			}
			b.WriteString(line[:cut])
			b.WriteByte('\n')
			line = line[cut:]
		}
		b.WriteString(line)
	}
	return b.String()
}

// keyValueTable renders a StringList of "key||value" rows as a two-column
// table, growing row heights to fit wrapped values (Table never grows rows
// on its own, so long values would overflow the rows below).
func keyValueTable(list binding.StringList) *widget.Table {
	rows, _ := list.Get()
	var table *widget.Table
	rowHeights := map[int]float32{}
	table = widget.NewTable(
		func() (int, int) {
			return len(rows), 2
		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("wide content")
			l.Wrapping = fyne.TextWrapWord
			return l
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			key, value, _ := strings.Cut(rows[i.Row], "||")
			l := o.(*widget.Label)
			if i.Col == 0 {
				l.SetText(key)
			} else {
				l.SetText(value)
			}

			valueWidth := float32(550)
			if w := table.Size().Width; w > 0 {
				valueWidth = w * 0.73
				table.SetColumnWidth(1, valueWidth) // track window resizes
			}

			// measure the wrapped value text and grow the row to fit
			m := widget.NewLabel(value)
			m.Wrapping = fyne.TextWrapWord
			m.Resize(fyne.NewSize(valueWidth, 0))
			if h := m.MinSize().Height; h != rowHeights[i.Row] {
				rowHeights[i.Row] = h
				table.SetRowHeight(i.Row, h)
			}
		},
	)

	table.SetColumnWidth(0, 200)
	table.SetColumnWidth(1, 550)

	list.AddListener(binding.NewDataListener(func() {
		rows, _ = list.Get()
		rowHeights = map[int]float32{}
		table.Refresh()
	}))

	return table
}

func (g *gui) makeResponseUI(request *core.Request) fyne.CanvasObject {
	bindings := g.tabs[request.ID+".json"].bindings
	bodyString, _ := bindings.body.Get()
	responseTab := widget.NewTextGridFromString(softWrap(bodyString))
	responseTab.Scroll = fyne.ScrollBoth
	responseTab.ShowLineNumbers = true
	responseTab.ShowWhitespace = true // toggle available in toolbar

	search := newResponseSearch(responseTab, (*g.Window).Canvas())
	g.tabs[request.ID+".json"].showSearch = search.show

	headerMap, _ := bindings.headers.Get() // render() reads Content-Type from it
	headerTable := keyValueTable(bindings.headers)
	cookieTable := keyValueTable(bindings.cookies)

	// Copy Icon to copy the whole response to the clipboard
	// Gets updated to a check icon when clicked, for better visual feedback
	var copyIcon *tappableIcon
	copyIcon = newTappableIcon(theme.ContentCopyIcon(), func() {
		original, _ := bindings.body.Get() // not responseTab.Text(): that contains soft-wrap newlines
		fyne.CurrentApp().Clipboard().SetContent(original)
		copyIcon.icon.Resource = theme.ConfirmIcon()
		copyIcon.Refresh()

		time.AfterFunc(2*time.Second, func() {
			fyne.Do(func() {
				copyIcon.icon.Resource = theme.ContentCopyIcon()
				copyIcon.Refresh()
			})
		})
	})
	copyIcon.Hide()

	// Save-to-file. ponytail: writes the retained body (2MB cap, 4MB read
	// cap) — stream-to-file download is a later feature.
	var saveIcon *tappableIcon
	saveIcon = newTappableIcon(theme.DownloadIcon(), func() {
		body, _ := bindings.body.Get()
		if body == "" {
			return
		}

		fileSave := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()

			if _, err := writer.Write([]byte(body)); err != nil {
				dialog.NewError(err, *g.Window).Show()
			}
		}, *g.Window)
		fileSave.SetFileName(saveFileName(request.URL, headerMap))
		fileSave.Show()
	})
	saveIcon.Hide()

	searchIcon := newTappableIcon(theme.SearchIcon(), search.show)
	searchIcon.Hide()

	// Whitespace toggle button
	var wsToggle *widget.Button
	wsToggle = widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
		responseTab.ShowWhitespace = !responseTab.ShowWhitespace
		if responseTab.ShowWhitespace {
			wsToggle.SetIcon(theme.VisibilityIcon())
		} else {
			wsToggle.SetIcon(theme.VisibilityOffIcon())
		}
		responseTab.Refresh()
	})
	wsToggle.Importance = widget.LowImportance
	wsToggle.Hide()

	// Status pill: coloured background, green 2xx, yellow 3xx, red 4xx/5xx
	statusText := canvas.NewText("", color.White)
	statusText.TextStyle.Bold = true
	statusText.TextSize = 12
	pillBg := canvas.NewRectangle(theme.Color(theme.ColorNameSuccess))
	pillBg.CornerRadius = 10
	statusPill := container.NewStack(pillBg, container.NewPadded(statusText))
	statusPill.Hide()
	bindings.status.AddListener(binding.NewDataListener(func() {
		s, _ := bindings.status.Get()
		if s == "" {
			return
		}
		statusText.Text = s
		pillBg.FillColor = theme.Color(theme.ColorNameSuccess)
		switch s[0] {
		case '3':
			pillBg.FillColor = theme.Color(theme.ColorNameWarning)
		case '4', '5':
			pillBg.FillColor = theme.Color(theme.ColorNameError)
		}
		statusPill.Show()
		statusPill.Refresh()
	}))

	// No ThemeOverride wrapper: each ThemeOverride mints a fresh font-cache
	// scope on every apply/refresh and Fyne never evicts it, so wrapping
	// refreshing widgets leaks font faces (the response area refreshes on
	// every request). Padding comes from the global theme instead.
	tabs := container.NewAppTabs(
		container.NewTabItem("Response", responseTab),
		container.NewTabItem("Headers", headerTable),
		container.NewTabItem("Cookies", cookieTable),
	)

	bindings.headers.AddListener(binding.NewDataListener(func() {
		headerMap, _ = bindings.headers.Get()
	}))

	var responsePlaceholder *fyne.Container
	emptyIcon := widget.NewIcon(theme.UploadIcon())
	emptyTitle := canvas.NewText("No Response Yet", theme.Color(theme.ColorNameForeground))
	emptyTitle.TextSize = 15
	emptyTitle.TextStyle.Bold = true
	emptyTitle.Alignment = fyne.TextAlignCenter
	emptySubtitle := canvas.NewText("Send a request to see the response here", theme.Color(theme.ColorNameDisabled))
	emptySubtitle.TextSize = 11
	emptySubtitle.Alignment = fyne.TextAlignCenter
	emptyState := container.NewCenter(
		container.NewVBox(
			container.NewCenter(emptyIcon),
			container.NewCenter(emptyTitle),
			container.NewCenter(emptySubtitle),
		),
	)
	responsePlaceholder = emptyState

	showRaw := false
	var rawToggle *widget.Button

	render := func() {
		responseBodyString, _ := bindings.body.Get()

		// If empty, clear the TextGrid to free memory and return
		if responseBodyString == "" {
			responseTab.SetText("")
			search.contentChanged()
			return
		}

		// Display cap: highlighted TextGrid rows cost far more than raw bytes
		// (a TextGridCell per rune plus a style pointer), so keep this modest.
		// Copy always returns the full retained body.
		const maxDisplayChars = 150_000
		if len(responseBodyString) > maxDisplayChars {
			responseBodyString = safeCut(responseBodyString, maxDisplayChars) + "\n\n... [Display truncated to keep the UI responsive. Copy returns the full retained body.]"
		}

		responsePlaceholder.Hide()
		tabs.Show()

		if showRaw {
			responseTab.SetText(softWrap(responseBodyString))
		} else {
			var contentType string
			for _, h := range headerMap {
				if k, v, ok := strings.Cut(h, "||"); ok && strings.EqualFold(k, "Content-Type") {
					contentType = v
					break
				}
			}

			lang := detectLang(contentType, responseBodyString)
			display := formatBody(responseBodyString, lang)
			if rows := highlightGridRows(display, lang); rows != nil {
				responseTab.Rows = rows
				responseTab.Refresh()
			} else {
				responseTab.SetText(softWrap(display))
			}
		}

		copyIcon.Show()
		saveIcon.Show()
		searchIcon.Show()
		wsToggle.Show()
		rawToggle.Show()

		// rows were rebuilt; stale match styles must not be restored
		search.contentChanged()
	}

	rawToggle = widget.NewButton("Raw", func() {
		showRaw = !showRaw
		if showRaw {
			rawToggle.Importance = widget.HighImportance
		} else {
			rawToggle.Importance = widget.LowImportance
		}
		rawToggle.Refresh()
		render()
	})
	rawToggle.Importance = widget.LowImportance
	rawToggle.Hide()

	g.tabs[request.ID+".json"].bodyListner = binding.NewDataListener(render)
	bindings.body.AddListener(g.tabs[request.ID+".json"].bodyListner)

	tabs.Hide()
	stackedTabs := container.NewStack(responsePlaceholder, tabs)

	// Meta + actions share the tab-header row: floated top-right over the
	// AppTabs bar so the response area only has one header row.
	// Collapse toggle hides the body so only that row stays visible.
	var rc *ResponseContainer
	var collapseBtn *widget.Button
	collapseBtn = widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		if stackedTabs.Visible() {
			stackedTabs.Hide()
			rc.SetOffset(1)
			collapseBtn.SetIcon(theme.MoveUpIcon())
		} else {
			stackedTabs.Show()
			rc.SetOffset(0.7)
			collapseBtn.SetIcon(theme.MoveDownIcon())
		}
	})
	collapseBtn.Importance = widget.LowImportance

	// Timing waterfall shown while hovering the time label. Lives in this
	// Stack (not a widget.PopUp — its overlay steals hover and flickers).
	waterfallHolder := container.NewStack()
	waterfallHolder.Hide()
	// anchor owns the topRight layout; refresh *it* on show, not the
	// holder — the holder was sized 0x0 while empty and only the anchor's
	// layout pass gives it its real size
	waterfallAnchor := container.New(topRight{}, waterfallHolder)
	timeLabel := newTimingLabel(bindings.time, func(hovering bool) {
		if !hovering {
			waterfallHolder.Hide()
			return
		}
		v, _ := bindings.timings.Get()
		t, ok := v.(core.Timings)
		if !ok || t.Total <= 0 {
			return
		}
		waterfallHolder.Objects = []fyne.CanvasObject{timingPanel(t)}
		waterfallHolder.Show()
		waterfallAnchor.Refresh()
	})

	toolbar := container.NewBorder(nil, nil, nil,
		container.NewHBox(
			container.NewCenter(statusPill),
			timeLabel,
			widget.NewLabelWithData(bindings.size),
			rawToggle, wsToggle, searchIcon, copyIcon, saveIcon, collapseBtn,
		),
	)

	rc = NewResponseContainer(container.NewBorder(search.bar, nil, nil, nil, container.NewStack(
		stackedTabs,
		container.NewBorder(toolbar, nil, nil, nil,
			waterfallAnchor,
		),
	)))
	return rc
}
