package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"net/url"
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

var appversion string

type gui struct {
	Window *fyne.Window

	urlInput       *widget.Entry
	tabs           map[string]*tab
	doctabs        *container.DocTabs
	sidebar        *fyne.Container
	requestHistory *[]map[string]string
	requestList    *widget.List
	requestCtx     context.Context
	cancelRequest  context.CancelFunc
}

type tab struct {
	bindings    *bindings
	item        *container.TabItem
	request     *core.Request
	bodyListner binding.DataListener
}

type bindings struct {
	headers binding.StringList
	body    binding.String
	status  binding.String
	size    binding.String
	time    binding.String
}

func MakeGUI(window *fyne.Window, version string) fyne.CanvasObject {

	g := &gui{Window: window}
	appversion = version
	g.tabs = make(map[string]*tab)
	g.doctabs = container.NewDocTabs()
	tabItem := g.makeTab(nil)
	g.doctabs.Append(tabItem)

	// Need to clean g.tabs when the tab is closed
	g.doctabs.CloseIntercept = func(ti *container.TabItem) {
		var deletable string
		for child, tab := range g.tabs {
			if tab.item == ti {
				deletable = child
			}
		}

		// Clean Up
		if deletable != "" {
			g.tabs[deletable].bindings.body.RemoveListener(g.tabs[deletable].bodyListner)
			g.tabs[deletable].bindings.body = nil
			g.tabs[deletable].bindings.headers = nil
			g.tabs[deletable].bindings.status = nil
			g.tabs[deletable].bindings.time = nil
			g.tabs[deletable].bodyListner = nil
			g.tabs[deletable].bindings = nil
			delete(g.tabs, deletable)
		}

		g.doctabs.Remove(ti)
		ti.Content = nil
		ti = nil

		// If all tabs are closed we need to add a new empty tab
		if len(g.doctabs.Items) == 0 {
			g.doctabs.Append(g.makeTab(nil))
		}
	}

	g.doctabs.OnSelected = func(tabItem *container.TabItem) {
		// The structure of g.tabs is a map of [tabID]DocTabTab Items
		// And the structur of g.requestHistory is a map of [tabID][historyData]
		// List reuquestList does not corelate to openedTab thats why we need to make a check through requestHistory
		for tabID, openedTab := range g.tabs {
			if openedTab.item == tabItem {
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
	baseView := NewHSplit(g.sidebar, container.NewThemeOverride(g.doctabs, &overridePaddingTheme{padding: 1.5}))
	baseView.Offset = 0.22

	footerSeperator := widget.NewSeparator()
	versionLabel := widget.NewLabel("Version: " + version)
	siteURL, _ := url.Parse("https://themyapi.com")
	myAPISite := widget.NewHyperlink("Website", siteURL)
	footerContent := container.NewHBox(
		myAPISite,
		canvas.NewCircle(theme.Color(theme.ColorNameDisabled)),
		versionLabel,
		canvas.NewRectangle(theme.Color(theme.ColorNameBackground)),
	)

	footer := container.NewThemeOverride(container.NewBorder(footerSeperator, nil, nil, footerContent, nil), &footerTheme{})

	return container.NewBorder(nil, footer, nil, nil, baseView)
}

// request here can be nil as we might not want to send it here
func (g *gui) makeTab(request *core.Request) *container.TabItem {

	if request == nil {
		// Tab ID is the id which is the time a tab is created
		tabID := fmt.Sprintf("%d", time.Now().Unix())
		request = &core.Request{ID: tabID, Method: "GET", IsDirty: true}
	}

	// Pushing the tab item to the open tab map
	g.tabs[request.ID+".json"] = &tab{item: nil, bindings: &bindings{}, request: request}

	requestUI := g.makeRequestUI(g.tabs[request.ID+".json"].request)
	response := g.makeResponseUI(g.tabs[request.ID+".json"].request)
	g.urlInput = widget.NewEntry()
	g.urlInput.SetPlaceHolder("Request URL")
	if request.URL != "" {
		g.urlInput.Text = request.URL
	}

	g.urlInput.OnChanged = func(s string) {
		request.URL = s
		request.IsDirty = true

		// Handling, query input blocks get added and deleted when
		// when query params gets added/removed directly to the URL input
		// Making query blocks and url input be 2 way binding.
		if parsedURL, err := url.Parse(request.URL); err == nil {
			parameters := parsedURL.Query()

			if len(parameters) > 0 && len(*request.QueryParams) > 0 {
				for i, params := range *request.QueryParams {
					if params.Value == "" || params.Key == "" {
						*request.QueryParams = append((*request.QueryParams)[:i], (*request.QueryParams)[i+1:]...)
						continue
					}

					if params.Checked && params.Key != "" && len(parameters[params.Key]) != 0 {
						if len(parameters[params.Key]) == 1 {
							*request.QueryParams = nil
							continue
						}
						*request.QueryParams = append((*request.QueryParams)[:i], (*request.QueryParams)[i+1:]...)
						continue
					}
				}
			}

			for key, value := range parameters {
				if value[0] != "" {
					var skipAppend bool = false
					for i, params := range *request.QueryParams {
						if params.Key == "" {
							(*request.QueryParams)[i].Key = key
							(*request.QueryParams)[i].Value = value[0]
							(*request.QueryParams)[i].Checked = true
							skipAppend = true
							break
						} else if params.Key == key {
							(*request.QueryParams)[i].Value = value[0]
							(*request.QueryParams)[i].Checked = true
							skipAppend = true
							break
						} else if params.Value == "" && params.Key != "" {
							*request.QueryParams = append((*request.QueryParams)[:i], (*request.QueryParams)[i+1:]...)
							skipAppend = true
							break
						}
					}
					if !skipAppend {
						*request.QueryParams = append(*request.QueryParams, core.FormType{Checked: true, Key: key, Value: value[0]})
					}
				}

			}
		}

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
	//requestType.Resize(fyne.NewSize(10, 40))

	var makeRequest *widget.Button
	makeRequest = widget.NewButton("Send", func() {
		if makeRequest.Text == "Cancel" {
			go func() {
				g.cancelRequest()
			}()
			return
		}

		// Create a cancelable context
		g.requestCtx, g.cancelRequest = context.WithCancel(context.Background())

		//makeRequest.Disable()
		makeRequest.SetText("Cancel")

		go func(ctx context.Context) {
			defer fyne.Do(func() {
				makeRequest.SetText("Send")
				g.cancelRequest = nil
			})

			res, err := request.SendRequest(ctx)

			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}

				errorDialoge := dialog.NewError(err, *g.Window)
				errorDialoge.Show()

				return
			}

			// Updating the open tab list with our new tab data
			// This only should happen for the first request on a new tab
			if g.tabs[request.ID+".json"] == nil {
				g.tabs[request.ID+".json"] = &tab{item: g.doctabs.Selected(), bindings: &bindings{}, request: &core.Request{}}
			}

			bindings := g.tabs[request.ID+".json"].bindings
			if request.Method == "HEAD" {
				res.Body = "Head Request do not have a body"
			}

			bindings.body.Set(res.Body)

			var headers []string
			for name, value := range res.Headers {
				headers = append(headers, name+"||"+value)
			}

			bindings.headers.Set(headers)
			bindings.size.Set(res.Size)
			bindings.status.Set(res.Status)
			bindings.time.Set(res.Duration.Abs().String())

			res.Body = ""

			// To update the current tab text as if it is dirty it set a *
			if request.IsDirty {
				request.IsDirty = false
				g.doctabs.Selected().Text = maybTruncateURL(g.urlInput.Text)

				fyne.Do(func() {
					g.doctabs.Refresh()
				})
			}

			// Updating the List
			g.requestHistory = core.ListHistory()

			fyne.Do(func() {
				g.requestList.Refresh()

				// Setting 0 becuase when a request is made then the list item of that request
				// Comes to the top of the list which has a index of 0
				g.requestList.Select(0)
			})
		}(g.requestCtx)
	})

	makeRequest.Importance = widget.HighImportance // Using it for button to have the theme color
	// Triggring request on enter
	g.urlInput.OnSubmitted = func(s string) {
		makeRequest.Tapped(&fyne.PointEvent{})
	}

	//statusDataLabel := widget.NewLabelWithData(g.tabs[request.ID+".json"].bindings.status)
	requestAction := container.NewPadded(container.NewBorder(nil, nil, requestType, makeRequest, g.urlInput))
	responseMetaData := container.NewHBox(
		widget.NewLabelWithData(g.tabs[request.ID+".json"].bindings.status),
		widget.NewLabelWithData(g.tabs[request.ID+".json"].bindings.size),
		widget.NewLabelWithData(g.tabs[request.ID+".json"].bindings.time),
	)

	requestMetaFloat := container.NewBorder(nil, nil, nil, responseMetaData, nil)
	requestResponseContainer := container.NewStack(requestUI, NewResponseContainer(container.NewStack(response, requestMetaFloat)))

	tabName := "New Request *"
	if request.URL != "" {
		tabName = maybTruncateURL(request.URL)
	}

	tabItem := container.NewTabItem(tabName, container.NewThemeOverride(container.NewBorder(requestAction, nil, nil, nil, requestResponseContainer), &overridePaddingTheme{}))

	g.tabs[request.ID+".json"].item = tabItem

	return tabItem
}

func (g *gui) makeSideBar() *fyne.Container {

	newRequestButton := container.NewPadded(widget.NewButton("New", func() {
		newTab := g.makeTab(nil)
		g.doctabs.Append(newTab)
		g.doctabs.Select(newTab)
	}))

	// newCollectionBtn := container.NewPadded(widget.NewButton("New", func() {

	// }))

	g.requestHistory = core.ListHistory()
	g.renderHistoryContent()

	rightBorder := canvas.NewLine(theme.Color(theme.ColorNameSeparator))
	rightBorder.StrokeWidth = 0.3

	sideBarLabel := widget.NewLabel("History")
	sideBarLabel.TextStyle.Bold = true

	collections := widget.NewTree(
		func(tni widget.TreeNodeID) []widget.TreeNodeID {
			switch tni {
			case "":
				return []widget.TreeNodeID{"a", "b", "c"}
			}
			return []string{}
		}, func(tni widget.TreeNodeID) bool {
			return tni == "" || tni == "a"
		}, func(b bool) fyne.CanvasObject {
			if b {
				return widget.NewLabel("Branch template")
			}
			return widget.NewLabel("Leaf template")
		}, func(tni widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			text := tni
			if b {
				text += " (branch)"
			}
			co.(*widget.Label).SetText(text)
		},
	)

	sideBarHeader := container.NewBorder(nil, nil, sideBarLabel, newRequestButton, nil)
	historyTabContent := container.NewBorder(sideBarHeader, nil, nil, nil, g.requestList)
	collectionTabContent := container.NewBorder(nil, nil, nil, nil, collections)
	envTabContent := container.NewBorder(widget.NewLabel("Environment"), nil, nil, nil)
	
	// Sidebar Icon tabs active state
	// TODO:: will need to make a custom widget to handle this
	historyTabActive := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	historyTabActive.CornerRadius = 7
	collectionTabActive := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	collectionTabActive.CornerRadius = 7
	envTabActive := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	envTabActive.CornerRadius = 7

	// History is the default tab so it will stay active on startup
	collectionTabActive.Hide()
	envTabActive.Hide()

	var historyTab *widget.Button
	historyTab = widget.NewButtonWithIcon("", theme.HistoryIcon(), func() {
		historyTabContent.Show()
		historyTabActive.Show()

		if collectionTabContent.Visible() {
			collectionTabContent.Hide()
			collectionTabActive.Hide()
		}

		if envTabContent.Visible() {
			envTabContent.Hide()
			envTabActive.Hide()
		}
	})

	historyTab.Importance = widget.LowImportance
	historyTabIconWrap := container.NewStack(historyTabActive, historyTab)

	collectionTab := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		collectionTabContent.Show()
		collectionTabActive.Show()

		if historyTabContent.Visible() {
			historyTabContent.Hide()
			historyTabActive.Hide()
		}

		if envTabContent.Visible() {
			envTabContent.Hide()
			envTabActive.Hide()
		}

	})
	collectionTab.Importance = widget.LowImportance
	collectionTabIconWrap := container.NewStack(collectionTabActive, collectionTab)

	envTab := widget.NewButtonWithIcon("", theme.ComputerIcon(), func() {
		envTabContent.Show()
		envTabActive.Show()

		if collectionTabContent.Visible() {
			collectionTabContent.Hide()
			collectionTabActive.Hide()
		}

		if historyTabContent.Visible() {
			historyTabContent.Hide()
			historyTabActive.Hide()
		}

	})
	envTab.Importance = widget.LowImportance
	envTabIconWrap := container.NewStack(envTabActive, envTab)

	shortCutIcon := widget.NewIcon(theme.NewThemedResource(resourceKeyboardSvg))
	shortcutsButton := widget.NewButtonWithIcon("", shortCutIcon.Resource, func() {
		keyboardShortcuts := widget.NewModalPopUp(widget.NewLabel("Keyboard Shortcuts"), (*g.Window).Canvas())
		keyboardShortcuts.Show()

		time.AfterFunc(3*time.Second, func() {
			fyne.Do(func() {
				keyboardShortcuts.Hide()
			})
		})
	})

	shortcutsButton.Importance = widget.LowImportance

	sideSwitcher := container.NewVBox(historyTabIconWrap, collectionTabIconWrap, envTabIconWrap)

	collectionTabContent.Hide()
	envTabContent.Hide()

	sideBarTabs := container.NewStack(
		historyTabContent,
		collectionTabContent,
		envTabContent,
	)

	return container.NewBorder(
		nil,
		nil,
		container.NewBorder(nil, shortcutsButton, nil, rightBorder, sideSwitcher),
		nil,
		sideBarTabs,
	)
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
