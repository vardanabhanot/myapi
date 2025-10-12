package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

func (g *gui) renderHistoryContent() {
	g.requestList = widget.NewList(
		func() int {
			return len(*g.requestHistory)
		},
		func() fyne.CanvasObject {

			//bg := canvas.NewRectangle(color.RGBA{72, 180, 97, 255})
			//bg.SetMinSize(fyne.NewSize(40, 15)) // Adjust size as needed
			//bg.CornerRadius = 6

			// Text label
			label := canvas.NewText("GET", color.White)
			label.Alignment = fyne.TextAlignCenter
			label.TextStyle.Bold = true
			label.TextSize = 10
			label.Color = color.RGBA{72, 180, 97, 10}

			badge := container.NewCenter(label)
			url := widget.NewLabel("https://themyapi.com/")
			url.Truncation = fyne.TextTruncateEllipsis

			timeElapsed := canvas.NewText("1d", theme.Color(theme.ColorNameForeground))
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

			if o.Visible() && (*g.requestHistory)[i]["mtime"] == "" {
				core.LoadMetaData((*g.requestHistory)[i]["ID"], &(*g.requestHistory)[i])
			}

			hoverableContainer, _ := o.(*hoverableListItem)
			paddedContainer, _ := hoverableContainer.content.(*fyne.Container)
			borderContainer, _ := paddedContainer.Objects[0].(*fyne.Container)
			badgeContainer := borderContainer.Objects[1].(*fyne.Container)
			optionsStack := borderContainer.Objects[2].(*fyne.Container)
			timeContainer := optionsStack.Objects[0].(*fyne.Container)

			optionsStack.Objects[1].(*tappableIcon).onTapped = func() {
				var items []*fyne.MenuItem
				delete := fyne.NewMenuItem("Delete", func() {
					err := core.DeleteHistory((*g.requestHistory)[i]["ID"])

					if err != nil {
						dialog.NewError(err, *g.Window)
						return
					}

					(*g.requestHistory) = append((*g.requestHistory)[:i], (*g.requestHistory)[i+1:]...)
					g.requestList.Refresh()
				})

				items = append(items, delete)

				clone := fyne.NewMenuItem("Clone", func() {

					if err := core.CloneHistory((*g.requestHistory)[i]["ID"]); err != nil {
						fmt.Println(err)
						dialog.NewError(err, (*g.Window))
					}

					go func() {
						g.requestHistory = core.ListHistory()
						fyne.Do(func() {
							g.requestList.Refresh()
						})
					}()

				})
				items = append(items, clone)

				d := fyne.CurrentApp().Driver()
				icon := optionsStack.Objects[1].(*tappableIcon)
				c := d.CanvasForObject(icon)
				popUpMenu := widget.NewPopUpMenu(fyne.NewMenu("", items...), c)
				buttonPos := d.AbsolutePositionForObject(icon)
				buttonSize := icon.Size()

				var popUpPos fyne.Position
				popUpPos.X = buttonPos.X + (buttonSize.Width / 2)
				popUpPos.Y = buttonPos.Y + (buttonSize.Height / 1.2)

				if popUpPos.X < 0 {
					popUpPos.X = 0
				}
				if popUpPos.Y < 0 {
					popUpPos.Y = 0
				}

				// Position the popup near the icon
				popUpMenu.ShowAtPosition(popUpPos)
				popUpMenu.Resize(fyne.NewSize(120, popUpMenu.MinSize().Height))

			}

			// Setting values
			//badgeContainer.Objects[0].(*canvas.Rectangle).FillColor = methodColor((*g.requestHistory)[i]["method"])
			if len((*g.requestHistory)[i]["method"]) > 4 {
				badgeContainer.Objects[0].(*canvas.Text).Text = (*g.requestHistory)[i]["method"][0:3]
			} else {
				badgeContainer.Objects[0].(*canvas.Text).Text = (*g.requestHistory)[i]["method"]
			}

			badgeContainer.Objects[0].(*canvas.Text).Color = methodColor((*g.requestHistory)[i]["method"])
			borderContainer.Objects[0].(*widget.Label).SetText((*g.requestHistory)[i]["requestURL"])
			timeContainer.Objects[0].(*canvas.Text).Text = (*g.requestHistory)[i]["mtime"] // Last Request
		},
	)

	// Handling loading of the history
	g.requestList.OnSelected = func(id widget.ListItemID) {
		for t, i := range g.tabs {
			// If the List Select is triggered by the select of the tab then we need to make sure
			// we do not end up reselecting the doctab as that got selected already.
			if t == (*g.requestHistory)[id]["ID"] && g.doctabs.Selected() == i.item {
				return
			}

			if t == (*g.requestHistory)[id]["ID"] {
				g.doctabs.Select(i.item)
				return
			}
		}

		request, err := core.LoadRequest((*g.requestHistory)[id]["ID"])

		if err != nil {
			dialog.NewError(err, *g.Window)
			return
		}

		g.tabs[(*g.requestHistory)[id]["ID"]] = &tab{bindings: &bindings{}}
		tabItem := g.makeTab(request)
		g.tabs[(*g.requestHistory)[id]["ID"]].item = tabItem
		g.doctabs.Append(tabItem)
		g.doctabs.Select(tabItem)
	}
}
