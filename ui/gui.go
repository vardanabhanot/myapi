package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

type gui struct {
	Window *fyne.Window

	Tabs     *container.DocTabs
	urlInput *widget.Entry
	bindings bindings
	request  *core.Request
}

type bindings struct {
	headers binding.StringList
	body    binding.String
	status  binding.String
	size    binding.String
	time    binding.String
}

func MakeGUI(window *fyne.Window) fyne.CanvasObject {

	g := &gui{Window: window}
	g.request = &core.Request{}
	tabItem := g.makeTab()
	tabs := container.NewDocTabs(
		tabItem,
	)

	sidebar := g.makeSideBar(tabs)
	baseView := container.NewHSplit(sidebar, tabs)
	baseView.Offset = 0.22

	return container.NewBorder(nil, container.NewHBox(widget.NewLabel("About")), nil, nil, baseView)
}

func (g *gui) makeTab() *container.TabItem {

	request := g.makeRequestUI()
	response := g.makeResponseUI()
	g.urlInput = widget.NewEntry()
	g.urlInput.SetPlaceHolder("Request URL")
	g.urlInput.OnChanged = func(s string) {
		g.request.URL = s
	}

	requestType := widget.NewSelect([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}, func(value string) {
		g.request.Method = value
	})

	requestType.SetSelected("GET")
	requestType.Resize(fyne.NewSize(10, 40))
	var makeRequest *widget.Button
	makeRequest = widget.NewButton("Send", func() {
		makeRequest.Disable()

		fyne.Do(func() {
			res, err := g.request.SendRequest()

			if err != nil {
				fmt.Println(err)
				errorDialoge := dialog.NewError(err, *g.Window)
				errorDialoge.Show()
				makeRequest.Enable()
				return
			}

			g.bindings.body.Set(res.Body)
			//g.bindings.headers.Set(res.Headers)
			g.bindings.size.Set(res.Size)
			g.bindings.status.Set(res.Status)
			makeRequest.Enable()
		})

	})

	requestAction := container.NewPadded(container.NewBorder(nil, nil, requestType, makeRequest, g.urlInput))
	requestResponseContainer := container.NewVSplit(request, response)
	requestResponseContainer.Offset = 0.7
	tabItem := container.NewTabItem("New Request*", container.NewBorder(requestAction, nil, nil, nil, requestResponseContainer))

	return tabItem
}

func (g *gui) makeSideBar(tabs *container.DocTabs) fyne.CanvasObject {

	requestButton := container.NewPadded(widget.NewButton("New Request", func() {
		newTab := g.makeTab()
		tabs.Append(newTab)
		tabs.Select(newTab)
	}))

	requestHistory := core.ListHistory()

	var requestList *widget.List

	requestList = widget.NewList(
		func() int {
			return len(*requestHistory)
		},
		func() fyne.CanvasObject {

			bg := canvas.NewRectangle(color.RGBA{72, 180, 97, 255})
			bg.SetMinSize(fyne.NewSize(40, 15)) // Adjust size as needed
			bg.CornerRadius = 6

			// Text label
			label := canvas.NewText("GET", color.White)
			label.Alignment = fyne.TextAlignCenter
			label.TextStyle.Bold = true
			label.TextSize = 10

			badge := container.NewCenter(bg, container.NewPadded(label))
			url := widget.NewLabel("https://themyapi.com/")
			url.Truncation = fyne.TextTruncateEllipsis

			timeElapsed := canvas.NewText("1 day ago", theme.Color(theme.ColorNameForeground))
			timeElapsed.TextSize = 10
			timeElapsed.TextStyle.Italic = true

			return container.NewPadded(
				container.NewGridWithRows(2,
					container.NewBorder(nil, nil, badge, nil, url),
					container.NewBorder(nil, nil, timeElapsed, widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})),
				))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			paddedC, _ := o.(*fyne.Container)
			grid, _ := paddedC.Objects[0].(*fyne.Container)

			firstRow, _ := grid.Objects[0].(*fyne.Container)
			badge, _ := firstRow.Objects[1].(*fyne.Container)
			badge.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).Text = (*requestHistory)[i]["method"]
			grid.Objects[1].(*fyne.Container).Objects[1].(*widget.Button).OnTapped = func() {
				fmt.Println((*requestHistory)[i]["ID"])
				fileURI := storage.NewFileURI((*requestHistory)[i]["ID"])
				err := storage.Delete(fileURI)

				if err != nil {
					fmt.Println(err)
				}

				(*requestHistory) = append((*requestHistory)[:i], (*requestHistory)[i+1:]...)
				requestList.Refresh()
			}
			badge.Objects[0].(*canvas.Rectangle).FillColor = methodColor((*requestHistory)[i]["method"])
			firstRow.Objects[0].(*widget.Label).SetText((*requestHistory)[i]["requestURL"])
		},
	)

	requestList.OnSelected = func(id widget.ListItemID) {
		fmt.Println(id)
	}

	rightBorder := canvas.NewLine(theme.Color(theme.ColorNameSeparator))
	rightBorder.StrokeWidth = 0.7

	return container.NewBorder(
		nil,
		nil,
		nil,
		rightBorder,
		container.NewBorder(requestButton, nil, nil, nil, requestList))
}

func methodColor(method string) *color.RGBA {
	switch method {
	case "POST":
		return &color.RGBA{219, 114, 180, 255}

	case "PUT":
		return &color.RGBA{228, 155, 15, 255}

	case "PATCH":
		return &color.RGBA{142, 91, 185, 255}

	case "DELETE":
		return &color.RGBA{216, 90, 121, 255}

	case "OPTIONS":
		return &color.RGBA{181, 234, 215, 255}

	case "HEAD":
		return &color.RGBA{122, 84, 189, 255}

	case "CONNECT":
		return &color.RGBA{175, 203, 255, 255}

	case "TRACE":
		return &color.RGBA{248, 216, 168, 255}

	default:
		return &color.RGBA{72, 180, 97, 255}
	}
}
