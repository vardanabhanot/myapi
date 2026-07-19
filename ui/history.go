package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

// filterHistory replaces the visible history with entries whose URL or
// method matches the query. Empty query restores the full list.
func (g *gui) filterHistory(query string) {
	all := core.ListHistory()

	if query != "" {
		query = strings.ToLower(query)
		filtered := all[:0]
		for _, entry := range all {
			// Search needs the metadata that list rows normally lazy-load
			if !entry.Loaded {
				entry.LoadMeta()
			}

			if strings.Contains(strings.ToLower(entry.URL), query) ||
				strings.Contains(strings.ToLower(entry.Method), query) {
				filtered = append(filtered, entry)
			}
		}
		all = filtered
	}

	g.requestHistory = all
	g.requestList.UnselectAll()
	g.requestList.Refresh()
}

func (g *gui) renderHistoryContent() {
	g.requestList = widget.NewList(
		func() int {
			return len(g.requestHistory)
		},
		func() fyne.CanvasObject {
			// Pill background; colors are placeholders, the update callback
			// sets the real per-method color
			mCol := methodColor("GET")
			pillBg := canvas.NewRectangle(color.NRGBA{R: mCol.R, G: mCol.G, B: mCol.B, A: 40})
			pillBg.CornerRadius = 4
			pillBg.SetMinSize(fyne.NewSize(42, 20))

			// Method text label
			label := canvas.NewText("GET", mCol)
			label.Alignment = fyne.TextAlignCenter
			label.TextStyle.Bold = true
			label.TextSize = 10

			badge := container.NewCenter(container.NewStack(pillBg, container.NewPadded(label)))
			url := widget.NewLabel("https://vardana.dev/myapi/")
			url.Truncation = fyne.TextTruncateEllipsis

			timeElapsed := canvas.NewText("1d", theme.Color(theme.ColorNameDisabled))
			timeElapsed.TextSize = 10
			timeElapsedPadded := container.NewPadded(timeElapsed)

			optionsIcon := newTappableIcon(theme.MoreHorizontalIcon(), func() {})
			optionsIcon.Hide()

			optionsStack := container.NewStack(timeElapsedPadded, optionsIcon)

			return newHoverableListItem(container.NewPadded(
				container.NewBorder(nil, nil, badge, optionsStack, url),
			))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {

			if o.Visible() && !g.requestHistory[i].Loaded {
				g.requestHistory[i].LoadMeta()
			}

			hoverableContainer, _ := o.(*hoverableListItem)
			paddedContainer, _ := hoverableContainer.content.(*fyne.Container)
			borderContainer, _ := paddedContainer.Objects[0].(*fyne.Container)
			// badgeContainer is container.NewCenter( container.NewStack(pillBg, container.NewPadded(label)) )
			badgeCenter := borderContainer.Objects[1].(*fyne.Container)
			badgeStack := badgeCenter.Objects[0].(*fyne.Container)
			pillBg := badgeStack.Objects[0].(*canvas.Rectangle)
			label := badgeStack.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text)
			optionsStack := borderContainer.Objects[2].(*fyne.Container)
			timeContainer := optionsStack.Objects[0].(*fyne.Container)

			optionsStack.Objects[1].(*tappableIcon).onTapped = func() {
				clone := fyne.NewMenuItem("Clone", func() {
					if err := core.CloneHistory(g.requestHistory[i].ID); err != nil {
						dialog.NewError(err, *g.Window).Show()
						return
					}

					go func() {
						g.requestHistory = core.ListHistory()
						fyne.Do(func() {
							g.requestList.Refresh()
						})
					}()
				})

				del := fyne.NewMenuItem("Delete", func() {
					dialog.NewConfirm("Delete Request", "Delete this request from history? This cannot be undone.", func(confirmed bool) {
						if !confirmed {
							return
						}

						if err := core.DeleteHistory(g.requestHistory[i].ID); err != nil {
							dialog.NewError(err, *g.Window).Show()
							return
						}

						g.requestHistory = append(g.requestHistory[:i], g.requestHistory[i+1:]...)
						g.requestList.Refresh()
					}, *g.Window).Show()
				})

				showIconMenu(optionsStack.Objects[1].(*tappableIcon), clone, del)
			}

			// Update pill color and text
			methodStr := g.requestHistory[i].Method
			if len(methodStr) > 4 {
				label.Text = methodStr[0:3]
			} else {
				label.Text = methodStr
			}
			mCol := methodColor(methodStr)
			label.Color = mCol
			pillBg.FillColor = color.NRGBA{R: mCol.R, G: mCol.G, B: mCol.B, A: 35}
			pillBg.Refresh()
			borderContainer.Objects[0].(*widget.Label).SetText(g.requestHistory[i].URL)
			timeContainer.Objects[0].(*canvas.Text).Text = g.requestHistory[i].MTime
		},
	)
	g.requestList.OnSelected = func(id widget.ListItemID) {
		for t, i := range g.tabs {
			// If the List Select is triggered by the select of the tab then we need to make sure
			// we do not end up reselecting the doctab as that got selected already.
			if t == g.requestHistory[id].ID && g.doctabs.Selected() == i.item {
				return
			}

			if t == g.requestHistory[id].ID {
				g.doctabs.Select(i.item)
				return
			}
		}

		request, err := core.LoadRequest(g.requestHistory[id].ID)

		if err != nil {
			dialog.NewError(err, *g.Window).Show()
			return
		}

		// makeTab registers g.tabs[request.ID] and sets .item itself
		tabItem := g.makeTab(request)
		g.doctabs.Append(tabItem)
		g.doctabs.Select(tabItem)
	}
}
