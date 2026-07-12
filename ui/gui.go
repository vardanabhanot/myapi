package ui

import (
	"context"
	"errors"
	"image/color"
	"net/url"
	"strings"

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
	queryList      *widget.List
	syncingQuery   bool // true while updateURL writes the URL entry
	tabs           map[string]*tab
	doctabs        *container.DocTabs
	sidebar        *fyne.Container
	requestHistory *[]map[string]string
	requestList    *widget.List
	envStore       *core.EnvStore
	envList        *widget.List
	collections    []*core.Collection
	collectionTree *widget.Tree
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
	cookies binding.StringList
	body    binding.String
	status  binding.String
	size    binding.String
	time    binding.String
	timings binding.Untyped // holds core.Timings for the waterfall popup
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
			g.tabs[deletable].bindings.cookies = nil
			g.tabs[deletable].bindings.status = nil
			g.tabs[deletable].bindings.timings = nil
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
	baseView := NewHSplit(g.sidebar, g.doctabs)
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
		request = &core.Request{ID: core.NewRequestID(), Method: "GET", IsDirty: true}
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

		// Sync URL query params into the query tab — but not when the
		// change came from the query tab itself (updateURL), which would
		// mutate the rows the user is typing into.
		if !g.syncingQuery {
			if parsedURL, err := url.Parse(s); err == nil {
				syncQueryParams(request.QueryParams, parsedURL.Query())
				if g.queryList != nil {
					g.queryList.Refresh()
				}
			}
		}

		s = tabTitle(request.Method, s)
		if request.IsDirty {
			s += " *"
		}

		g.doctabs.Selected().Text = s
		g.doctabs.Refresh()

	}

	requestType := widget.NewSelect([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}, func(value string) {
		request.Method = value

		// Keep the tab title's method prefix in sync; only when this tab
		// is the selected one (SetSelected also fires during makeTab)
		if t := g.tabs[request.ID+".json"]; t != nil && t.item != nil && g.doctabs.Selected() == t.item {
			title := tabTitle(value, request.URL)
			if request.IsDirty {
				title += " *"
			}
			t.item.Text = title
			g.doctabs.Refresh()
		}
	})

	requestType.SetSelected(request.Method)

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

		makeRequest.SetText("Cancel")
		makeRequest.Importance = widget.DangerImportance
		makeRequest.Refresh()

		go func(ctx context.Context) {
			defer fyne.Do(func() {
				makeRequest.SetText("Send")
				makeRequest.Importance = widget.HighImportance
				makeRequest.Refresh()
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

			// Cap what we keep: the binding holds the body for the tab's
			// whole lifetime, and copy/raw only ever need this much.
			// ponytail: 2MB cap; stream-to-file if full downloads matter.
			const maxRetainedBody = 2 << 20
			if len(res.Body) > maxRetainedBody {
				res.Body = safeCut(res.Body, maxRetainedBody) + "\n\n... [Truncated: kept the first 2 MB of " + res.Size + "]"
			}

			var headers []string
			for name, value := range res.Headers {
				headers = append(headers, name+"||"+value)
			}

			var cookies []string
			for _, c := range res.Cookies {
				// name in col 0, value + attributes (Path, Expires, ...) in col 1
				cookies = append(cookies, c.Name+"||"+strings.TrimPrefix(c.String(), c.Name+"="))
			}

			// Headers before body: the body listener reads Content-Type
			// from the headers binding to pick syntax highlighting.
			bindings.headers.Set(headers)
			bindings.cookies.Set(cookies)
			bindings.body.Set(res.Body)
			bindings.size.Set(res.Size)
			bindings.status.Set(res.Status)
			bindings.time.Set(res.Duration.Abs().String())
			bindings.timings.Set(res.Timings)

			res.Body = ""

			// To update the current tab text as if it is dirty it set a *
			if request.IsDirty {
				request.IsDirty = false
				g.doctabs.Selected().Text = tabTitle(request.Method, request.URL)

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

	addToColBtn := widget.NewButtonWithIcon("", theme.FolderNewIcon(), func() {
		g.addToCollectionDialog(request)
	})
	addToColBtn.Importance = widget.LowImportance

	// One visually fused bar: method + URL + save-to-collection + Send
	// share a rounded background
	urlBarBg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	urlBarBg.CornerRadius = 6
	requestAction := container.NewPadded(container.NewStack(
		urlBarBg,
		container.NewBorder(nil, nil, requestType, container.NewHBox(addToColBtn, makeRequest), g.urlInput),
	))

	requestResponseContainer := container.NewStack(requestUI, response)

	tabName := tabTitle(request.Method, request.URL)
	if request.URL == "" {
		tabName += " *"
	}

	tabItem := container.NewTabItem(tabName, container.NewBorder(requestAction, nil, nil, nil, requestResponseContainer))

	g.tabs[request.ID+".json"].item = tabItem

	return tabItem
}

func (g *gui) makeSideBar() *fyne.Container {

	newRequestBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() {
		newTab := g.makeTab(nil)
		g.doctabs.Append(newTab)
		g.doctabs.Select(newTab)
	})
	newRequestBtn.Importance = widget.HighImportance
	newRequestButton := container.NewPadded(newRequestBtn)

	g.requestHistory = core.ListHistory()
	g.renderHistoryContent()

	rightBorder := canvas.NewLine(theme.Color(theme.ColorNameSeparator))
	rightBorder.StrokeWidth = 1.0

	sideBarLabel := sectionHeader("History")

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search history")
	searchEntry.OnChanged = func(s string) {
		g.filterHistory(s)
	}

	clearAllBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		dialog.NewConfirm("Clear History", "Delete all request history? This cannot be undone.", func(confirmed bool) {
			if !confirmed {
				return
			}

			if err := core.ClearHistory(); err != nil {
				dialog.NewError(err, *g.Window).Show()
				return
			}

			g.requestHistory = core.ListHistory()
			g.requestList.Refresh()
		}, *g.Window).Show()
	})
	clearAllBtn.Importance = widget.LowImportance

	sideBarHeader := container.NewBorder(nil, nil, container.NewPadded(sideBarLabel), container.NewHBox(clearAllBtn, newRequestButton), nil)
	historyTabContent := container.NewBorder(
		container.NewVBox(sideBarHeader, container.NewPadded(searchEntry)),
		nil, nil, nil,
		g.requestList,
	)
	collectionTabContent := g.makeCollectionContent()
	envTabContent := g.makeEnvContent()

	// Active state: a primary-coloured left accent bar, more visible than a grey background
	historyTabActive := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	historyTabActive.CornerRadius = 2
	historyTabActive.SetMinSize(fyne.NewSize(3, 0))
	collectionTabActive := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	collectionTabActive.CornerRadius = 2
	collectionTabActive.SetMinSize(fyne.NewSize(3, 0))
	envTabActive := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	envTabActive.CornerRadius = 2
	envTabActive.SetMinSize(fyne.NewSize(3, 0))

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
	// Active indicator: a 3px primary-coloured strip on the LEFT of the icon
	historyTabIconWrap := container.NewBorder(nil, nil, historyTabActive, nil, historyTab)

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
	collectionTabIconWrap := container.NewBorder(nil, nil, collectionTabActive, nil, collectionTab)

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
	envTabIconWrap := container.NewBorder(nil, nil, envTabActive, nil, envTab)

	shortCutIcon := widget.NewIcon(theme.NewThemedResource(resourceKeyboardSvg))
	shortcutsButton := widget.NewButtonWithIcon("", shortCutIcon.Resource, func() {
		shortcutsTitle := canvas.NewText("Keyboard Shortcuts", theme.Color(theme.ColorNameForeground))
		shortcutsTitle.TextSize = 14
		shortcutsTitle.TextStyle.Bold = true

		type shortcut struct{ keys, action string }
		shortcuts := []shortcut{
			{"Enter", "Send request"},
			{"Ctrl + T", "New tab (planned)"},
			{"Ctrl + W", "Close tab (planned)"},
		}

		rows := container.NewVBox()
		for _, s := range shortcuts {
			keyLabel := canvas.NewText(s.keys, theme.Color(theme.ColorNamePrimary))
			keyLabel.TextStyle.Monospace = true
			keyLabel.TextSize = 12
			actionLabel := widget.NewLabel(s.action)
			row := container.NewBorder(nil, nil, nil, actionLabel, keyLabel)
			rows.Add(row)
			rows.Add(widget.NewSeparator())
		}

		content := container.NewVBox(
			shortcutsTitle,
			widget.NewSeparator(),
			rows,
		)

		closeBtn := widget.NewButton("Close", nil)
		var popup *widget.PopUp
		closeBtn.OnTapped = func() { popup.Hide() }
		closeBtn.Importance = widget.LowImportance

		popupContent := container.NewBorder(nil, container.NewPadded(closeBtn), nil, nil, container.NewPadded(content))
		popup = widget.NewModalPopUp(popupContent, (*g.Window).Canvas())
		popup.Resize(fyne.NewSize(340, 0))
		popup.Show()
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

// sectionHeader is the one style used for every small section label
func sectionHeader(text string) *canvas.Text {
	t := canvas.NewText(text, theme.Color(theme.ColorNameDisabled))
	t.TextSize = 11
	t.TextStyle.Bold = true
	return t
}

// tabTitle renders a tab as "GET /users" instead of a truncated raw URL
func tabTitle(method, rawURL string) string {
	if rawURL == "" {
		return "New Request"
	}

	label := rawURL
	if u, err := url.Parse(rawURL); err == nil {
		if u.Path != "" && u.Path != "/" {
			label = u.Path
		} else if u.Host != "" {
			label = u.Host
		}
	}

	if r := []rune(label); len(r) > 22 {
		label = string(r[:22]) + "…"
	}

	return method + " " + label
}
