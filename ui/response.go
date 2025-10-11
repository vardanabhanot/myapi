package ui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

func (g *gui) makeResponseUI(request *core.Request) fyne.CanvasObject {
	bindings := g.tabs[request.ID+".json"].bindings
	bodyString, _ := bindings.body.Get()
	responseTab := widget.NewTextGridFromString(bodyString)
	responseTab.Scroll = fyne.ScrollBoth
	responseTab.ShowLineNumbers = true
	responseTab.ShowWhitespace = true

	headerMap, _ := bindings.headers.Get()
	var headerTable *widget.Table
	headerTable = widget.NewTable(
		func() (int, int) {
			return len(headerMap), 2
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("wide content")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			rows := [][]string{}
			for _, b := range headerMap {
				row := strings.Split(b, "||")
				rows = append(rows, row)
			}

			l := o.(*widget.Label)
			l.SetText(rows[i.Row][i.Col])
			l.Wrapping = fyne.TextWrapWord
			headerTable.SetColumnWidth(1, headerTable.Size().Width*0.73) // updating the column width when screen minimizes or vice versa
		},
	)

	headerTable.SetColumnWidth(0, 200)
	headerTable.SetColumnWidth(1, 550)

	// Copy Icon to copy the whole response to the clipboard
	// Gets updated to a check icon when clicked, for better visual feedback
	var copyIcon *tappableIcon
	copyIcon = newTappableIcon(theme.ContentCopyIcon(), func() {
		fyne.CurrentApp().Clipboard().SetContent(responseTab.Text())
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

	// To position the icon correctly at top right we need to place it in nested Border container.
	var copyIconWrapper *fyne.Container
	copyIconWrapper = container.NewBorder(
		container.NewBorder(
			layout.NewSpacer(),
			nil,
			layout.NewSpacer(),
			copyIcon),
		nil, nil, nil,
	)

	tabs := container.NewThemeOverride(container.NewAppTabs(
		container.NewTabItem(
			"Response",
			container.NewThemeOverride(
				container.NewStack(responseTab, copyIconWrapper),
				&overridePaddingTheme{}),
		),
		container.NewTabItem("Headers", container.NewThemeOverride(headerTable, &overridePaddingTheme{})),
		//container.NewTabItem("Cookies", widget.NewLabel("Cookies here")),
	), &overridePaddingTheme{padding: 1.5})

	bindings.headers.AddListener(binding.NewDataListener(func() {
		headerMap, _ = bindings.headers.Get()
	}))

	var responseBodyString string
	bindings.body.AddListener(binding.NewDataListener(func() {
		responseBodyString, _ = bindings.body.Get()

		responseTab.SetText(responseBodyString)

		if responseBodyString != "No response yet" && responseBodyString != "" {
			copyIcon.Show()
		}

	}))

	return container.NewBorder(nil, nil, nil, nil, container.NewBorder(nil, nil, nil, nil, tabs))
}
