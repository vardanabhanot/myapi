package main

import (
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type tabs struct {
	tab []fyne.URI
	api map[fyne.URI]api
}

type api struct {
	response      *http.Response
	duration      time.Duration
	headers       binding.StringList
	body          binding.String
	statusBinding binding.String
	sizeBinding   binding.String
	timeBinding   binding.String
	queries       []queryFields
	urlinput      *widget.Entry
}

type queryFields struct {
	id        int
	checked   bool
	parameter string
	value     string
}

type authOptHolder struct {
	none   fyne.CanvasObject
	basic  fyne.CanvasObject
	bearer fyne.CanvasObject
}

func main() {
	a := app.New()
	w := a.NewWindow("MyAPI")

	w.Resize(fyne.NewSize(1024, 600))
	w.CenterOnScreen()
	w.SetContent(makeGUI())
	w.ShowAndRun()
}

func makeGUI() fyne.CanvasObject {

	tabItem := makeTab()
	tabs := container.NewDocTabs(
		tabItem,
	)

	sidebar := makeSideBar(tabs)
	content := []fyne.CanvasObject{sidebar, tabs}

	return container.New(newBaseLayout(sidebar, tabs), content...)
}

func makeTab() *container.TabItem {

	a := &api{}
	request := a.makeRequestUI()
	response := a.makeResponseUI()
	a.urlinput = widget.NewEntry()
	a.urlinput.SetPlaceHolder("Request URL")

	requestType := widget.NewSelect([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}, func(value string) {
		//
	})

	requestType.SetSelected("GET")
	requestType.Resize(fyne.NewSize(10, 40))

	makeRequest := widget.NewButton("Send", func() {
		req, err := http.NewRequest(requestType.Selected, a.urlinput.Text, nil)

		if err != nil {
			log.Println(err)
			return
		}

		client := &http.Client{}

		startTime := time.Now()
		response, err := client.Do(req)

		if err != nil {
			log.Println(err)
			return
		}

		endTime := time.Now()
		duration := endTime.Sub(startTime)

		a.duration = duration
		a.response = response

		// Convert response headers to a bindable map
		headerMap := []string{}
		for key, values := range response.Header {
			headerMap = append(headerMap, key+"||"+values[0]) // Get the first value for simplicity
		}

		a.headers.Set(headerMap)

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			return
		}

		a.body.Set(string(body))

		var statusValue string
		if a.response != nil && a.response.Status != "" {
			statusValue = a.response.Status
		}

		status := statusValue

		a.statusBinding.Set(status)

		size := fmt.Sprint(len(body))
		a.sizeBinding.Set(size)

		time := a.duration.String()
		a.timeBinding.Set(time)

	})

	requestAction := container.NewPadded(container.NewBorder(nil, nil, requestType, makeRequest, a.urlinput))

	tabItem := container.NewTabItem("New Request*", container.NewBorder(requestAction, nil, nil, nil, container.NewGridWithColumns(2, request, response)))

	return tabItem
}

func makeSideBar(tabs *container.DocTabs) fyne.CanvasObject {

	requestButton := container.NewPadded(widget.NewButton("New Request", func() {
		newTab := makeTab()
		tabs.Append(newTab)
		tabs.Select(newTab)
	}))

	rightBorder := canvas.NewLine(color.RGBA{R: 240, G: 240, B: 240, A: 255})
	rightBorder.StrokeWidth = 0.7

	return container.NewBorder(
		nil,
		nil,
		nil,
		rightBorder,
		container.NewVBox(requestButton))
}

func (a *api) makeRequestUI() fyne.CanvasObject {

	var status, size, time string
	bodyResponse := "No response yet"
	a.statusBinding = binding.BindString(&status)
	a.sizeBinding = binding.BindString(&size)
	a.timeBinding = binding.BindString(&time)
	a.body = binding.BindString(&bodyResponse)
	a.headers = binding.NewStringList()

	a.queries = []queryFields{}
	a.queries = append(a.queries, queryFields{checked: true})
	fields := a.queryBlock()

	queryContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, widget.NewLabel("Query Parameters"), widget.NewButton("Add Parameter", func() {
				a.queries = append(a.queries, queryFields{id: len(a.queries), checked: true})
				fields.Refresh()
			}), nil),
			nil,
			nil,
			nil,
			fields,
		),
	)

	authOptIns := &authOptHolder{}

	authOptIns.none = container.NewVBox(widget.NewLabel("No Authentication Selected"))

	basicUsername := widget.NewEntry()
	basicUsername.SetPlaceHolder("Username")
	basicPassword := widget.NewEntry()
	basicPassword.SetPlaceHolder("Password")
	basicPassword.Password = true

	authOptIns.basic = container.NewBorder(
		widget.NewLabel("Basic Authentication"),
		nil,
		nil,
		nil,
		container.NewVBox(
			container.NewBorder(nil, nil, widget.NewLabel("Username"), nil, basicUsername),
			container.NewBorder(nil, nil, widget.NewLabel("Password"), nil, basicPassword),
		),
	)

	bearerPrefix := widget.NewEntry()
	bearerPrefix.SetText("Bearer")
	bearerPrefix.TextStyle.Bold = true

	bearerTokenArea := widget.NewEntry()
	bearerTokenArea.MultiLine = true
	bearerTokenArea.SetMinRowsVisible(7)

	authOptIns.bearer = container.NewBorder(
		widget.NewLabel("Bearer Authentication"),
		nil,
		nil,
		nil,
		container.NewVBox(
			bearerTokenArea,
			container.NewBorder(nil, nil, widget.NewLabel("Token Prefix"), nil, bearerPrefix),
		),
	)

	authOptIns.none.Hide()
	authOptIns.basic.Hide()
	authOptIns.bearer.Hide()

	authOptionView := container.NewStack(
		authOptIns.none,
		authOptIns.basic,
		authOptIns.bearer,
	)

	authOptions := widget.NewRadioGroup([]string{"None", "Basic", "Bearer", "AWS"}, func(value string) {
		switch value {
		case "None":
			authOptIns.none.Show()
			authOptIns.basic.Hide()
			authOptIns.bearer.Hide()

		case "Basic":
			authOptIns.basic.Show()
			authOptIns.none.Hide()
			authOptIns.bearer.Hide()

		case "Bearer":
			authOptIns.bearer.Show()
			authOptIns.basic.Hide()
			authOptIns.none.Hide()

		}
	})

	authOptions.Horizontal = true
	authOptions.SetSelected("None")

	authContainer := container.NewPadded(
		container.NewBorder(authOptions, nil, nil, nil, authOptionView),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Query", queryContainer),
		container.NewTabItem("Headers", widget.NewLabel("Headers here")),
		container.NewTabItem("Auth", authContainer),
		container.NewTabItem("Body", widget.NewLabel("Body input here")),
	)

	return container.NewBorder(nil, nil, nil, nil, tabs)
}

func (a *api) makeResponseUI() fyne.CanvasObject {

	max := container.NewPadded(container.NewAdaptiveGrid(
		3,
		widget.NewLabelWithData(a.statusBinding),
		widget.NewLabelWithData(a.sizeBinding),
		widget.NewLabelWithData(a.timeBinding),
	))

	bodyString, _ := a.body.Get()
	responseTab := widget.NewRichTextWithText(bodyString)
	responseTab.Wrapping = fyne.TextWrapBreak
	responseTab.Scroll = container.ScrollVerticalOnly
	headerMap, _ := a.headers.Get()
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

	a.headers.AddListener(binding.NewDataListener(func() {
		headerMap, _ = a.headers.Get()
	}))

	a.body.AddListener(binding.NewDataListener(func() {
		bodyString, _ = a.body.Get()

		responseTab.Segments = nil
		responseSegment := &widget.TextSegment{Text: bodyString, Style: widget.RichTextStyleCodeBlock}
		responseTab.Segments = append(responseTab.Segments, responseSegment)
		responseTab.Refresh()
	}))

	leftBorder := canvas.NewLine(color.RGBA{R: 240, G: 240, B: 240, A: 255})
	leftBorder.StrokeWidth = 0.7

	return container.NewBorder(nil, nil, leftBorder, nil, container.NewBorder(max, nil, nil, nil, tabs))
}

func (a *api) queryBlock() fyne.CanvasObject {
	return widget.NewList(func() int {
		return len(a.queries)
	}, func() fyne.CanvasObject {
		parameterEntry := widget.NewEntry()
		parameterEntry.SetPlaceHolder("parameter")
		valueEntry := widget.NewEntry()
		valueEntry.SetPlaceHolder("value")

		return container.NewBorder(nil, nil,
			widget.NewCheck("", func(b bool) {}),
			widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
			container.NewGridWithColumns(2, parameterEntry, valueEntry),
		)
	}, func(lii widget.ListItemID, co fyne.CanvasObject) {
		ctx, _ := co.(*fyne.Container)
		entry := ctx.Objects[1].(*widget.Check)
		entry.SetChecked(a.queries[lii].checked)
		entry.OnChanged = func(b bool) {
			a.queries[lii].checked = b
		}

		if lii == 0 {
			btn := ctx.Objects[2].(*widget.Button)
			btn.Hide()
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.SetText(a.queries[lii].parameter)
		parameter.OnChanged = func(s string) {
			a.queries[lii].parameter = s

			params := url.Values{}

			for _, q := range a.queries {
				if q.checked {
					params.Add(q.parameter, "")
				}
			}

			if a.urlinput.Text == "" {
				return
			}

			parsedURL, err := url.Parse(a.urlinput.Text)

			if err != nil {
				log.Println(err)
				return
			}
			a.urlinput.SetText(parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path + "?" + params.Encode())
		}
		value := entryCtx.Objects[1].(*widget.Entry)
		value.SetText(a.queries[lii].value)
		value.OnChanged = func(s string) {
			a.queries[lii].value = s

			params := url.Values{}

			for _, q := range a.queries {
				if q.checked {
					params.Add(q.parameter, q.value)
				}
			}

			if a.urlinput.Text == "" {
				return
			}

			parsedURL, err := url.Parse(a.urlinput.Text)
			if err != nil {
				log.Println(err)
				return
			}

			a.urlinput.SetText(parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path + "?" + params.Encode())

		}
	})
}
