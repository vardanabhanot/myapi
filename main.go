package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type api struct {
	id            string
	response      *http.Response
	duration      time.Duration
	headers       binding.StringList
	body          binding.String
	statusBinding binding.String
	sizeBinding   binding.String
	timeBinding   binding.String
	queries       []queryFields
	headersF      []headerFields
	bodyF         *bodyFields
	urlinput      *widget.Entry
	requestType   *widget.Select
}

type headerFields struct {
	ID        int    `json:"id"`
	Checked   bool   `json:"checked"`
	Parameter string `json:"parameters"`
	Value     string `json:"values"`
}

type bodyFields struct {
	content_type string
	content      string
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

type bodyOptHolder struct {
	json fyne.CanvasObject
	xml  fyne.CanvasObject
	text fyne.CanvasObject
}

type savedRequest struct {
	ID          string         `json:"id"`
	Method      string         `json:"method"`
	RequestURL  string         `json:"requestURL"`
	QueryParams []queryFields  `json:"queryParams"`
	Headers     []headerFields `json:"headers"`
	Body        *bodyFields    `json:"body"`
}

const VERSION = "0.0.1"

var window fyne.Window

func main() {
	a := app.New()
	window = a.NewWindow("MyAPI")

	window.Resize(fyne.NewSize(1024, 600))
	window.CenterOnScreen()
	window.SetContent(makeGUI())
	window.ShowAndRun()
}

func makeGUI() fyne.CanvasObject {

	tabItem := makeTab()
	tabs := container.NewDocTabs(
		tabItem,
	)

	sidebar := makeSideBar(tabs)
	baseView := container.NewHSplit(sidebar, tabs)
	baseView.Offset = 0.22

	return container.NewBorder(nil, container.NewHBox(widget.NewLabel("About")), nil, nil, baseView)
}

func makeTab() *container.TabItem {

	a := &api{}
	request := a.makeRequestUI()
	response := a.makeResponseUI()
	a.urlinput = widget.NewEntry()
	a.urlinput.SetPlaceHolder("Request URL")

	a.requestType = widget.NewSelect([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}, func(value string) {
		//
	})

	a.requestType.SetSelected("GET")
	a.requestType.Resize(fyne.NewSize(10, 40))

	makeRequest := widget.NewButton("Send", func() {
		req, err := http.NewRequest(a.requestType.Selected, a.urlinput.Text, nil)

		if err != nil {
			log.Println(err)
			return
		}

		for _, header := range a.headersF {
			if !header.Checked || header.Parameter == "" || header.Value == "" {
				continue
			}

			req.Header.Set(header.Parameter, header.Value)
		}

		if a.bodyF.content != "" {
			switch a.bodyF.content_type {
			case "JSON":
				req.Header.Set("Content-Type", "application/json")

			case "XML":
				req.Header.Set("Content-Type", "application/xml")

			case "Text":
				req.Header.Set("Content-Type", "text/plain")

			}

			req.Body = io.NopCloser(bytes.NewBuffer([]byte(a.bodyF.content)))
		}

		client := &http.Client{}

		startTime := time.Now()
		a.response, err = client.Do(req)

		if err != nil {
			log.Println(err)
			return
		}

		endTime := time.Now()
		a.duration = endTime.Sub(startTime)

		// Convert response headers to a bindable map
		headerMap := []string{}
		for key, values := range a.response.Header {
			headerMap = append(headerMap, key+"||"+values[0]) // Get the first value for simplicity
		}

		a.headers.Set(headerMap)

		defer a.response.Body.Close()

		body, err := io.ReadAll(a.response.Body)
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

		a.saveRequestData()

	})

	requestAction := container.NewPadded(container.NewBorder(nil, nil, a.requestType, makeRequest, a.urlinput))
	requestResponseContainer := container.NewVSplit(request, response)
	requestResponseContainer.Offset = 0.7
	tabItem := container.NewTabItem("New Request*", container.NewBorder(requestAction, nil, nil, nil, requestResponseContainer))

	return tabItem
}

func makeSideBar(tabs *container.DocTabs) fyne.CanvasObject {

	requestButton := container.NewPadded(widget.NewButton("New Request", func() {
		newTab := makeTab()
		tabs.Append(newTab)
		tabs.Select(newTab)
	}))

	requestHistory := listHistory()

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
			url := widget.NewLabel("https://myapi.io/")
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

func (a *api) makeRequestUI() fyne.CanvasObject {

	var status, size, time string
	bodyResponse := "No response yet"
	a.statusBinding = binding.BindString(&status)
	a.sizeBinding = binding.BindString(&size)
	a.timeBinding = binding.BindString(&time)
	a.body = binding.BindString(&bodyResponse)
	a.headers = binding.NewStringList()

	// Query options
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

	// Header Options
	a.headersF = []headerFields{}
	a.headersF = append(a.headersF, headerFields{Parameter: "Accept", Value: "*/*", Checked: true})
	a.headersF = append(a.headersF, headerFields{Parameter: "User-Agent", Value: "MyAPI/" + VERSION, Checked: true})
	a.headersF = append(a.headersF, headerFields{Parameter: "Accept-Encoding", Value: "gzip, deflate, br", Checked: true})
	a.headersF = append(a.headersF, headerFields{Parameter: "Connection", Value: "keep-alive", Checked: true})
	headerFieldss := a.headerBlock()

	headerContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, widget.NewLabel("Request Headers"), widget.NewButton("Add Header", func() {
				a.headersF = append(a.headersF, headerFields{ID: len(a.headersF), Checked: true})
				headerFieldss.Refresh()
			}), nil),
			nil,
			nil,
			nil,
			headerFieldss,
		),
	)

	// Auth Options
	authOptIns := &authOptHolder{}
	authOptIns.none = container.NewVBox(widget.NewLabel("No Authentication Selected"))

	basicUsername := widget.NewEntry()
	basicUsername.SetPlaceHolder("Username")
	basicPassword := widget.NewEntry()
	basicPassword.SetPlaceHolder("Password")
	basicPassword.Password = true
	basicHeading := widget.NewLabel("Basic Authentication")
	basicHeading.TextStyle.Bold = true

	authOptIns.basic = container.NewBorder(
		basicHeading,
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

	bearerHeading := widget.NewLabel("Bearer Authentication")
	bearerHeading.TextStyle.Bold = true
	bearerTokenArea := widget.NewEntry()
	bearerTokenArea.MultiLine = true
	bearerTokenArea.SetMinRowsVisible(7)

	authOptIns.bearer = container.NewBorder(
		bearerHeading,
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

	// Body Options
	bodyOptIns := &bodyOptHolder{}
	a.bodyF = &bodyFields{content_type: "JSON"}

	jsonHeading := widget.NewLabel("JSON")
	jsonHeading.TextStyle.Bold = true
	jsonTextArea := widget.NewEntry()
	jsonTextArea.MultiLine = true
	jsonTextArea.SetMinRowsVisible(7)
	jsonTextArea.TextStyle.Monospace = true
	jsonTextArea.OnChanged = func(s string) {
		a.bodyF.content = s
	}

	bodyOptIns.json = container.NewBorder(
		jsonHeading,
		nil,
		nil,
		nil,
		jsonTextArea,
	)

	xmlHeading := widget.NewLabel("XML")
	xmlHeading.TextStyle.Bold = true
	xmlTextArea := widget.NewEntry()
	xmlTextArea.MultiLine = true
	xmlTextArea.SetMinRowsVisible(7)
	xmlTextArea.TextStyle.Monospace = true
	xmlTextArea.OnChanged = func(s string) {
		a.bodyF.content = s
	}

	bodyOptIns.xml = container.NewBorder(
		xmlHeading,
		nil,
		nil,
		nil,
		xmlTextArea,
	)

	textHeading := widget.NewLabel("Text")
	textHeading.TextStyle.Bold = true
	textTextArea := widget.NewEntry()
	textTextArea.OnChanged = func(s string) {
		a.bodyF.content = s
	}
	textTextArea.MultiLine = true
	textTextArea.SetMinRowsVisible(7)
	textTextArea.TextStyle.Monospace = true

	bodyOptIns.text = container.NewBorder(
		textHeading,
		nil,
		nil,
		nil,
		textTextArea,
	)

	bodyOptions := widget.NewRadioGroup([]string{"JSON", "Form", "XML", "Text"}, func(value string) {
		switch value {
		case "JSON":
			bodyOptIns.json.Show()
			bodyOptIns.xml.Hide()
			bodyOptIns.text.Hide()

		case "Form":
			bodyOptIns.json.Hide()
			bodyOptIns.xml.Hide()
			bodyOptIns.text.Hide()

		case "XML":
			bodyOptIns.xml.Show()
			bodyOptIns.json.Hide()
			bodyOptIns.text.Hide()

		case "Text":
			bodyOptIns.text.Show()
			bodyOptIns.json.Hide()
			bodyOptIns.xml.Hide()
		}

		a.bodyF.content_type = value // Settings the value to use it when submitting the request
	})

	bodyOptions.Horizontal = true
	bodyOptions.SetSelected("JSON")

	bodyOptIns.json.Show()
	//bodyOptIns.form.Hide() // TODO:: Need to implement forms, am lazy to do it now as it will need key value inputs
	bodyOptIns.xml.Hide()
	bodyOptIns.text.Hide()

	bodyOptionView := container.NewStack(
		bodyOptIns.json,
		bodyOptIns.xml,
		bodyOptIns.text,
	)

	bodyContainer := container.NewPadded(
		container.NewBorder(bodyOptions, nil, nil, nil, bodyOptionView),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Query", queryContainer),
		container.NewTabItem("Headers", headerContainer),
		container.NewTabItem("Auth", authContainer),
		container.NewTabItem("Body", bodyContainer),
	)

	return container.NewBorder(nil, nil, nil, nil, tabs)
}

func (a *api) makeResponseUI() fyne.CanvasObject {

	// max := container.NewPadded(container.NewAdaptiveGrid(
	// 	3,
	// 	widget.NewLabelWithData(a.statusBinding),
	// 	widget.NewLabelWithData(a.sizeBinding),
	// 	widget.NewLabelWithData(a.timeBinding),
	// ))

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

	return container.NewBorder(nil, nil, nil, nil, container.NewBorder(nil, nil, nil, nil, tabs))
}

func (a *api) queryBlock() fyne.CanvasObject {

	var list *widget.List
	list = widget.NewList(func() int {
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
		if len(ctx.Objects) == 0 {
			return
		}

		entry := ctx.Objects[1].(*widget.Check)
		entry.SetChecked(a.queries[lii].checked)
		entry.OnChanged = func(b bool) {
			a.queries[lii].checked = b
		}

		if lii == 0 {
			btn := ctx.Objects[2].(*widget.Button)
			btn.Hide()
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			a.queries = append(a.queries[:lii], a.queries[lii+1:]...)
			list.Refresh()
			a.urlinput.Refresh()
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

	return list
}

func (a *api) headerBlock() fyne.CanvasObject {

	var list *widget.List
	list = widget.NewList(func() int {
		return len(a.headersF)
	}, func() fyne.CanvasObject {
		parameterEntry := widget.NewEntry()
		parameterEntry.SetPlaceHolder("Header")
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
		entry.SetChecked(a.headersF[lii].Checked)
		entry.OnChanged = func(b bool) {
			a.headersF[lii].Checked = b
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			a.headersF = append(a.headersF[:lii], a.headersF[lii+1:]...)
			list.Refresh()
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.SetText(a.headersF[lii].Parameter)
		parameter.OnChanged = func(s string) {
			a.headersF[lii].Parameter = s
		}

		value := entryCtx.Objects[1].(*widget.Entry)
		value.SetText(a.headersF[lii].Value)
		value.OnChanged = func(s string) {
			a.headersF[lii].Value = s

		}
	})

	return list
}

func (a *api) saveRequestData() {
	localDir, err := os.UserCacheDir()

	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	myapiPath := filepath.Join(localDir, "/myapi")

	_, err = os.Stat(myapiPath)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = os.Mkdir(myapiPath, os.ModeDir)
		}

		// If we still have an error, we need to let the user know
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
	}

	var filename string
	if a.id == "" {
		filename = fmt.Sprintf("%d", time.Now().Unix())
		a.id = filename
	} else {
		filename = a.id
	}

	requestFile := filepath.Join(myapiPath, "/"+filename+".json")

	uri := storage.NewFileURI(requestFile)

	writer, _ := storage.Writer(uri)
	defer writer.Close()

	data := &savedRequest{
		ID:          a.id,
		Method:      a.requestType.Selected,
		RequestURL:  a.urlinput.Text,
		QueryParams: a.queries,
		Headers:     a.headersF,
		Body:        a.bodyF,
	}

	jsondata, err := json.Marshal(data)

	if err != nil {
		fmt.Println(err)
		return
	}

	writer.Write(jsondata)

}

func listHistory() *[]map[string]string {
	localDir, err := os.UserCacheDir()

	var requests []map[string]string

	if err != nil {
		return &requests
	}

	myapiPath := filepath.Join(localDir, "/myapi")

	_, err = os.Stat(myapiPath)

	if err != nil {
		return &requests
	}

	uri := storage.NewFileURI(myapiPath)

	files, err := storage.List(uri)

	if err != nil {
		return &requests
	}

	for _, file := range files {
		if _, err := storage.CanRead(file); err != nil {
			return &requests
		}

		reader, _ := storage.Reader(file)

		defer reader.Close()

		var fileContent []byte
		fileContent, err = io.ReadAll(reader)

		if err != nil {
			continue
		}

		content := &savedRequest{}
		if err = json.Unmarshal(fileContent, content); err != nil {
			continue
		}

		var request = make(map[string]string)

		request["ID"] = file.String()
		request["requestURL"] = content.RequestURL
		request["method"] = content.Method
		requests = append(requests, request)
	}

	return &requests
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
