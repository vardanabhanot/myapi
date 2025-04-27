package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func (g *gui) makeResponseUI() fyne.CanvasObject {

	// max := container.NewPadded(container.NewAdaptiveGrid(
	// 	3,
	// 	widget.NewLabelWithData(a.statusBinding),
	// 	widget.NewLabelWithData(a.sizeBinding),
	// 	widget.NewLabelWithData(a.timeBinding),
	// ))

	bodyString, _ := g.bindings.body.Get()
	responseTab := widget.NewTextGridFromString(bodyString)
	responseTab.Scroll = fyne.ScrollBoth
	responseTab.ShowLineNumbers = true
	responseTab.ShowWhitespace = true

	headerMap, _ := g.bindings.headers.Get()
	headerTable := widget.NewTable(
		func() (int, int) {
			return len(headerMap), 2
		},
		func() fyne.CanvasObject {
			return container.NewStack(widget.NewLabel("wide content"))
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			rows := [][]string{}
			for _, b := range headerMap {
				row := strings.Split(b, "||")
				rows = append(rows, row)
			}

			l := o.(*fyne.Container).Objects[0].(*widget.Label)
			l.SetText(rows[i.Row][i.Col])
			l.Wrapping = fyne.TextWrapWord
		},
	)

	headerTable.SetColumnWidth(0, 200)
	headerTable.SetColumnWidth(1, 300)

	tabs := container.NewAppTabs(
		container.NewTabItem("Response", responseTab),
		container.NewTabItem("Headers", headerTable),
		container.NewTabItem("Cookies", widget.NewLabel("Cookies here")),
	)

	g.bindings.headers.AddListener(binding.NewDataListener(func() {
		headerMap, _ = g.bindings.headers.Get()
	}))

	var responseBodyString string
	g.bindings.body.AddListener(binding.NewDataListener(func() {
		responseBodyString, _ = g.bindings.body.Get()

		responseTab.SetText(responseBodyString)
		responseTab.Refresh()
	}))

	return container.NewBorder(nil, nil, nil, nil, container.NewBorder(nil, nil, nil, nil, tabs))
}
