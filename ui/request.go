package ui

import (
	"log"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

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

func (g *gui) makeRequestUI() fyne.CanvasObject {

	var status, size, time string
	bodyResponse := "No response yet"
	g.bindings.status = binding.BindString(&status)
	g.bindings.size = binding.BindString(&size)
	g.bindings.time = binding.BindString(&time)
	g.bindings.body = binding.BindString(&bodyResponse)
	g.bindings.headers = binding.NewStringList()

	// Query options
	g.request.QueryParams = &[]core.FormType{}
	g.request.Headers = &[]core.FormType{}
	*g.request.QueryParams = append(*g.request.QueryParams, core.FormType{Checked: true})
	fields := g.queryBlock(g.request.QueryParams)

	queryContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, widget.NewLabel("Query Parameters"), widget.NewButton("Add Parameter", func() {
				*g.request.QueryParams = append(*g.request.QueryParams, core.FormType{Checked: true})
				fields.Refresh()
			}), nil),
			nil,
			nil,
			nil,
			fields,
		),
	)

	// Header Options
	*g.request.Headers = append(*g.request.Headers, core.FormType{Key: "Accept", Value: "*/*", Checked: true})
	*g.request.Headers = append(*g.request.Headers, core.FormType{Key: "User-Agent", Value: "MyAPI/" + "0.0.1", Checked: true})
	*g.request.Headers = append(*g.request.Headers, core.FormType{Key: "Accept-Encoding", Value: "gzip, deflate, br", Checked: true})
	*g.request.Headers = append(*g.request.Headers, core.FormType{Key: "Connection", Value: "keep-alive", Checked: true})
	headerFieldss := g.headerBlock(g.request.Headers)

	headerContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, widget.NewLabel("Request Headers"), widget.NewButton("Add Header", func() {
				*g.request.Headers = append(*g.request.Headers, core.FormType{Checked: true})
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
	bodyF := &bodyFields{content_type: "JSON"}

	jsonHeading := widget.NewLabel("JSON")
	jsonHeading.TextStyle.Bold = true
	jsonTextArea := widget.NewEntry()
	jsonTextArea.MultiLine = true
	jsonTextArea.SetMinRowsVisible(7)
	jsonTextArea.TextStyle.Monospace = true
	jsonTextArea.OnChanged = func(s string) {
		bodyF.content = s
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
		bodyF.content = s
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
		bodyF.content = s
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
		g.request.BodyType = value

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

		bodyF.content_type = value // Settings the value to use it when submitting the request
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

func (g *gui) queryBlock(queries *[]core.FormType) fyne.CanvasObject {

	var list *widget.List
	list = widget.NewList(func() int {
		return len(*queries)
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
		entry.SetChecked((*queries)[lii].Checked)
		entry.OnChanged = func(b bool) {
			(*queries)[lii].Checked = b
		}

		if lii == 0 {
			btn := ctx.Objects[2].(*widget.Button)
			btn.Hide()
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			*queries = append((*queries)[:lii], (*queries)[lii+1:]...)
			list.Refresh()
			g.urlInput.Refresh()
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.SetText((*queries)[lii].Key)
		parameter.OnChanged = func(s string) {
			(*queries)[lii].Key = s

			params := url.Values{}

			for _, q := range *queries {
				if q.Checked {
					params.Add(q.Key, "")
				}
			}

			if g.urlInput.Text == "" {
				return
			}

			parsedURL, err := url.Parse(g.urlInput.Text)

			if err != nil {
				log.Println(err)
				return
			}

			g.urlInput.SetText(parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path + "?" + params.Encode())
		}
		value := entryCtx.Objects[1].(*widget.Entry)
		value.SetText((*queries)[lii].Value)
		value.OnChanged = func(s string) {
			(*queries)[lii].Value = s

			params := url.Values{}

			for _, q := range *queries {
				if q.Checked {
					params.Add(q.Key, q.Value)
				}
			}

			if g.urlInput.Text == "" {
				return
			}

			parsedURL, err := url.Parse(g.urlInput.Text)
			if err != nil {
				log.Println(err)
				return
			}

			g.urlInput.SetText(parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path + "?" + params.Encode())
		}
	})

	return list
}

func (g *gui) headerBlock(headers *[]core.FormType) fyne.CanvasObject {

	var list *widget.List
	list = widget.NewList(func() int {
		return len(*headers)
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
		entry.SetChecked((*headers)[lii].Checked)
		entry.OnChanged = func(b bool) {
			(*headers)[lii].Checked = b
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			(*headers) = append((*headers)[:lii], (*headers)[lii+1:]...)
			list.Refresh()
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.SetText((*headers)[lii].Key)
		parameter.OnChanged = func(s string) {
			(*headers)[lii].Key = s
		}

		value := entryCtx.Objects[1].(*widget.Entry)
		value.SetText((*headers)[lii].Value)
		value.OnChanged = func(s string) {
			(*headers)[lii].Value = s
		}
	})

	return list
}
