package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
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
	headerTable := widget.NewTable(
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
		},
	)

	headerTable.SetColumnWidth(0, 200)
	headerTable.SetColumnWidth(1, 500)

	tabs := container.NewAppTabs(
		container.NewTabItem("Response", responseTab),
		container.NewTabItem("Headers", headerTable),
		container.NewTabItem("Cookies", widget.NewLabel("Cookies here")),
	)

	bindings.headers.AddListener(binding.NewDataListener(func() {
		headerMap, _ = bindings.headers.Get()
	}))

	var responseBodyString string
	bindings.body.AddListener(binding.NewDataListener(func() {
		responseBodyString, _ = bindings.body.Get()

		responseTab.SetText(responseBodyString)
	}))

	return container.NewBorder(nil, nil, nil, nil, container.NewBorder(nil, nil, nil, nil, tabs))
}
