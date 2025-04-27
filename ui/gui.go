package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

type gui struct {
	Window *fyne.Window

	urlInput       *widget.Entry
	bindings       bindings
	tabs           map[string]*container.TabItem
	doctabs        *container.DocTabs
	sidebar        *fyne.Container
	requestHistory *[]map[string]string
	requestList    *widget.List
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
	g.tabs = make(map[string]*container.TabItem)
	g.doctabs = container.NewDocTabs()
	tabItem := g.makeTab(nil)
	g.doctabs.Append(tabItem)

	g.doctabs.CloseIntercept = func(ti *container.TabItem) {
		var deletable string
		for child, childItem := range g.tabs {
			if childItem == ti {
				deletable = child
			}
		}

		if deletable != "" {
			delete(g.tabs, deletable)
		}

		g.doctabs.Remove(ti)

		// If all tabs are closed we need to add a new empty tab
		if len(g.doctabs.Items) == 0 {
			g.doctabs.Append(g.makeTab(nil))
		}
	}

	g.doctabs.OnSelected = func(tabItem *container.TabItem) {
		// The structure of g.tabs is a map of [tabID]DocTabTab Items
		// And the structur of g.requestHistory is a map of [tabID][historyData]
		// List reuquestList does not corelate to openedTab thats why we need to make a check through requestHistory
		for tabID, opendTabItem := range g.tabs {
			if opendTabItem == tabItem {
				for index, history := range *g.requestHistory {
					if history["ID"] == tabID {
						g.requestList.Select(index)
						return
					}
				}
			}
		}

		// This will happen for a New Request tab as it does not gets saved until a request is sent.
		g.requestList.UnselectAll()
	}

	g.sidebar = g.makeSideBar()
	baseView := container.NewHSplit(g.sidebar, g.doctabs)
	baseView.Offset = 0.22

	return container.NewBorder(nil, container.NewHBox(widget.NewLabel("About")), nil, nil, baseView)
}

// request here can be nil as we might not want to send it here
func (g *gui) makeTab(request *core.Request) *container.TabItem {

	if request == nil {
		// Tab ID is the id which is the time a tab is created
		tabID := fmt.Sprintf("%d", time.Now().Unix())
		request = &core.Request{ID: tabID, Method: "GET", IsDirty: true}
	}

	requestUI := g.makeRequestUI(request)
	response := g.makeResponseUI()
	g.urlInput = widget.NewEntry()
	g.urlInput.SetPlaceHolder("Request URL")
	if request.URL != "" {
		g.urlInput.Text = request.URL
	}

	g.urlInput.OnChanged = func(s string) {
		request.URL = s
		request.IsDirty = true

		if s == "" {
			s = "New Request"
		} else {
			s = maybTruncateURL(s)
		}

		if request.IsDirty {
			s += " *"
		}

		g.doctabs.Selected().Text = s
		g.doctabs.Refresh()

	}

	requestType := widget.NewSelect([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}, func(value string) {
		request.Method = value
	})

	requestType.SetSelected(request.Method)
	requestType.Resize(fyne.NewSize(10, 40))

	var makeRequest *widget.Button
	makeRequest = widget.NewButton("Send", func() {
		makeRequest.Disable()

		fyne.Do(func() {
			res, err := request.SendRequest()

			if err != nil {
				errorDialoge := dialog.NewError(err, *g.Window)
				errorDialoge.Show()
				makeRequest.Enable()
				return
			}

			g.bindings.body.Set(res.Body)

			var headers []string
			for name, value := range res.Headers {
				headers = append(headers, name+"||"+value)
			}

			g.bindings.headers.Set(headers)
			g.bindings.size.Set(res.Size)
			g.bindings.status.Set(res.Status)
			makeRequest.Enable()

			// Updating the open tab list with our new tab data
			// This only should happen for the first request on a new tab
			if g.tabs[request.ID+".json"] == nil {
				g.tabs[request.ID+".json"] = g.doctabs.Selected()
			}

			// To update the current tab text as if it is dirty it set a *
			if request.IsDirty {
				request.IsDirty = false
				g.doctabs.Selected().Text = maybTruncateURL(g.urlInput.Text)
				g.doctabs.Refresh()
			}

			// Updating the List
			g.requestHistory = core.ListHistory()
			g.requestList.Refresh()

			// Setting 0 becuase when a request is made then the list item of that request
			// Comes to the top of the list which has a index of 0
			g.requestList.Select(0)
		})

	})

	requestAction := container.NewPadded(container.NewBorder(nil, nil, requestType, makeRequest, g.urlInput))
	requestResponseContainer := container.NewVSplit(requestUI, response)
	requestResponseContainer.Offset = 0.7

	tabName := "New Request *"
	if request.URL != "" {
		tabName = maybTruncateURL(request.URL)
	}

	tabItem := container.NewTabItem(tabName, container.NewBorder(requestAction, nil, nil, nil, requestResponseContainer))

	// Pushing the tab item to the open tab map
	g.tabs[request.ID+".json"] = tabItem

	return tabItem
}

func (g *gui) makeSideBar() *fyne.Container {

	requestButton := container.NewPadded(widget.NewButton("New Request", func() {
		newTab := g.makeTab(nil)
		g.doctabs.Append(newTab)
		g.doctabs.Select(newTab)
	}))

	g.requestHistory = core.ListHistory()
	g.requestList = widget.NewList(
		func() int {
			return len(*g.requestHistory)
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

			if o.Visible() {
				core.LoadMetaData((*g.requestHistory)[i]["ID"], &(*g.requestHistory)[i])
			}
			paddedC, _ := o.(*fyne.Container)
			grid, _ := paddedC.Objects[0].(*fyne.Container)

			firstRow, _ := grid.Objects[0].(*fyne.Container)
			badge, _ := firstRow.Objects[1].(*fyne.Container)
			badge.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).Text = (*g.requestHistory)[i]["method"]
			grid.Objects[1].(*fyne.Container).Objects[1].(*widget.Button).OnTapped = func() {
				err := core.DeleteHistory((*g.requestHistory)[i]["ID"])

				if err != nil {
					dialog.NewError(err, *g.Window)
					return
				}

				(*g.requestHistory) = append((*g.requestHistory)[:i], (*g.requestHistory)[i+1:]...)
				g.requestList.Refresh()
			}

			grid.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text).Text = (*g.requestHistory)[i]["mtime"] // Last Request
			badge.Objects[0].(*canvas.Rectangle).FillColor = methodColor((*g.requestHistory)[i]["method"])
			firstRow.Objects[0].(*widget.Label).SetText((*g.requestHistory)[i]["requestURL"])
		},
	)

	// Handling loading of the history
	g.requestList.OnSelected = func(id widget.ListItemID) {
		for t, i := range g.tabs {
			// If the List Select is triggered by the select of the tab then we need to make sure
			// we do not end up reselecting the doctab as that got selected already.
			if t == (*g.requestHistory)[id]["ID"] && g.doctabs.Selected() == i {
				return
			}

			if t == (*g.requestHistory)[id]["ID"] {
				g.doctabs.Select(i)
				return
			}
		}

		request, err := core.LoadRequest((*g.requestHistory)[id]["ID"])

		if err != nil {
			dialog.NewError(err, *g.Window)
			return
		}

		tabItem := g.makeTab(request)
		g.tabs[(*g.requestHistory)[id]["ID"]] = tabItem
		g.doctabs.Append(tabItem)
		g.doctabs.Select(tabItem)
	}

	rightBorder := canvas.NewLine(theme.Color(theme.ColorNameSeparator))
	rightBorder.StrokeWidth = 0.7

	return container.NewBorder(
		nil,
		nil,
		nil,
		rightBorder,
		container.NewBorder(requestButton, nil, nil, nil, g.requestList))
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

func maybTruncateURL(url string) string {
	if len(url) > 20 {
		tabRune := []rune(url)
		url = string(tabRune[0:20])
		url += "..."
	}

	return url
}
