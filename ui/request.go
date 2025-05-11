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

type authOptHolder struct {
	none   fyne.CanvasObject
	basic  fyne.CanvasObject
	bearer fyne.CanvasObject
}

type bodyOptHolder struct {
	json fyne.CanvasObject
	xml  fyne.CanvasObject
	text fyne.CanvasObject
	form fyne.CanvasObject
}

func (g *gui) makeRequestUI(request *core.Request) fyne.CanvasObject {

	var status, size, time string
	bodyResponse := "No response yet"
	bindings := g.tabs[request.ID+".json"].bindings
	bindings.status = binding.BindString(&status)
	bindings.size = binding.BindString(&size)
	bindings.time = binding.BindString(&time)
	bindings.body = binding.BindString(&bodyResponse)
	bindings.headers = binding.NewStringList()

	// Query options
	if request.QueryParams == nil {
		request.QueryParams = &[]core.FormType{}
		*request.QueryParams = append(*request.QueryParams, core.FormType{Checked: true})
	}

	fields := g.queryBlock(request.QueryParams)

	queryContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, widget.NewLabel("Query Parameters"), widget.NewButton("Add Parameter", func() {
				*request.QueryParams = append(*request.QueryParams, core.FormType{})
				fields.Refresh()
			}), nil),
			nil,
			nil,
			nil,
			fields,
		),
	)

	if request.Headers == nil {
		request.Headers = &[]core.FormType{}

		// Default Header Options
		*request.Headers = append(*request.Headers, core.FormType{Key: "Accept", Value: "*/*", Checked: true})
		*request.Headers = append(*request.Headers, core.FormType{Key: "User-Agent", Value: "MyAPI/" + appversion, Checked: true})
		//*request.Headers = append(*request.Headers, core.FormType{Key: "Accept-Encoding", Value: "gzip, deflate, br", Checked: true})
		*request.Headers = append(*request.Headers, core.FormType{Key: "Connection", Value: "keep-alive", Checked: true})
	}

	headerFieldss := g.headerBlock(request.Headers)

	headerContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, widget.NewLabel("Request Headers"), widget.NewButton("Add Header", func() {
				*request.Headers = append(*request.Headers, core.FormType{Checked: true})
				headerFieldss.Refresh()
			}), nil),
			nil,
			nil,
			nil,
			headerFieldss,
		),
	)

	if request.Auth == nil {
		request.Auth = &core.Auth{}
	}

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

	if request.Auth.BasicUser != "" {
		basicUsername.SetText(request.Auth.BasicUser)
	}

	if request.Auth.BasicPass != "" {
		basicPassword.SetText(request.Auth.BasicPass)
	}

	basicUsername.OnChanged = func(s string) {
		request.Auth.BasicUser = s
	}

	basicPassword.OnChanged = func(s string) {
		request.Auth.BasicPass = s
	}

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

	bearerPrefix.OnChanged = func(s string) {
		request.Auth.BearerPrefix = s
	}

	if request.Auth.BearerPrefix != "" {
		bearerPrefix.SetText(request.Auth.BearerPrefix)
	} else {
		bearerPrefix.SetText("Bearer")
	}

	bearerHeading := widget.NewLabel("Bearer Authentication")
	bearerHeading.TextStyle.Bold = true
	bearerTokenArea := widget.NewEntry()
	bearerTokenArea.MultiLine = true
	bearerTokenArea.SetMinRowsVisible(5)
	bearerTokenArea.Scroll = fyne.ScrollVerticalOnly
	bearerTokenArea.Wrapping = fyne.TextWrapBreak

	if request.Auth.BearerAuth != "" {
		bearerTokenArea.SetText(request.Auth.BearerAuth) // Loading the Auth token
	}

	// Updating the Request Bearer token
	bearerTokenArea.OnChanged = func(s string) {
		request.Auth.BearerAuth = s
	}

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

	// TODO:: Will need to implement AWS as well here
	authOptions := widget.NewRadioGroup([]string{"None", "Basic", "Bearer"}, func(value string) {
		request.AuthType = value

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

	// Loading the Request
	if request.AuthType != "" {
		authOptions.SetSelected(request.AuthType)
	} else {
		authOptions.SetSelected("None")
	}

	authContainer := container.NewPadded(
		container.NewBorder(authOptions, nil, nil, nil, authOptionView),
	)

	// Body Options
	bodyOptIns := &bodyOptHolder{}
	if request.BodyType == "" {
		request.BodyType = "JSON"
	}

	jsonHeading := widget.NewLabel("JSON")
	jsonHeading.TextStyle.Bold = true
	jsonTextArea := widget.NewEntry()
	if request.Body.Json != "" && request.BodyType == "JSON" {
		jsonTextArea.SetText(request.Body.Json)
	}

	jsonTextArea.MultiLine = true
	jsonTextArea.SetMinRowsVisible(5)
	jsonTextArea.TextStyle.Monospace = true
	jsonTextArea.OnChanged = func(s string) {
		request.Body.Json = s
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

	if request.Body.Xml != "" && request.BodyType == "XML" {
		jsonTextArea.SetText(request.Body.Xml)
	}

	xmlTextArea.MultiLine = true
	xmlTextArea.SetMinRowsVisible(7)
	xmlTextArea.TextStyle.Monospace = true
	xmlTextArea.OnChanged = func(s string) {
		request.Body.Xml = s
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

	if request.Body.Text != "" && request.BodyType == "Text" {
		jsonTextArea.SetText(request.Body.Text)
	}

	textTextArea.OnChanged = func(s string) {
		request.Body.Text = s
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

	if request.Body.Form == nil {
		request.Body.Form = &[]core.FormType{}
		*request.Body.Form = append(*request.Body.Form, core.FormType{Checked: true})
	}

	formFieldsBlock := g.formBlock(request.Body.Form)
	formHeading := widget.NewLabel("Form Fields")
	formHeading.TextStyle.Bold = true
	formContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, formHeading, widget.NewButton("Add Field", func() {
				*request.Body.Form = append(*request.Body.Form, core.FormType{})
				formFieldsBlock.Refresh()
			}), nil),
			nil,
			nil,
			nil,
			formFieldsBlock,
		),
	)

	bodyOptIns.form = container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		formContainer,
	)

	bodyOptions := widget.NewRadioGroup([]string{"JSON", "Form", "XML", "Text"}, func(value string) {
		request.BodyType = value

		switch value {
		case "JSON":
			bodyOptIns.json.Show()
			bodyOptIns.xml.Hide()
			bodyOptIns.text.Hide()
			bodyOptIns.form.Hide()

		case "Form":
			bodyOptIns.form.Show()
			bodyOptIns.json.Hide()
			bodyOptIns.xml.Hide()
			bodyOptIns.text.Hide()

		case "XML":
			bodyOptIns.xml.Show()
			bodyOptIns.json.Hide()
			bodyOptIns.text.Hide()
			bodyOptIns.form.Hide()

		case "Text":
			bodyOptIns.text.Show()
			bodyOptIns.json.Hide()
			bodyOptIns.xml.Hide()
			bodyOptIns.form.Hide()
		}
	})

	bodyOptions.Horizontal = true

	if request.BodyType == "" {
		bodyOptions.SetSelected("JSON")
	} else {
		bodyOptions.SetSelected(request.BodyType)
	}

	bodyOptIns.json.Show()
	bodyOptIns.form.Hide() // TODO:: Need to implement forms, am lazy to do it now as it will need key value inputs
	bodyOptIns.xml.Hide()
	bodyOptIns.text.Hide()

	bodyOptionView := container.NewStack(
		bodyOptIns.json,
		bodyOptIns.xml,
		bodyOptIns.text,
		bodyOptIns.form,
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

			if g.urlInput.Text == "" {
				return
			}

			g.updateURL(queries)
		}

		if lii == 0 {
			btn := ctx.Objects[2].(*widget.Button)
			btn.Hide()
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			*queries = append((*queries)[:lii], (*queries)[lii+1:]...)
			list.Refresh()

			g.updateURL(queries)
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.SetText((*queries)[lii].Key)
		parameter.OnChanged = func(s string) {
			(*queries)[lii].Key = s

			if !(*queries)[lii].Checked {
				return
			}

			if g.urlInput.Text == "" {
				return
			}

			g.updateURL(queries)
		}
		value := entryCtx.Objects[1].(*widget.Entry)
		value.SetText((*queries)[lii].Value)
		value.OnChanged = func(s string) {
			(*queries)[lii].Value = s

			if !(*queries)[lii].Checked {
				return
			}

			g.updateURL(queries)
		}
	})

	return list
}

func (g *gui) updateURL(queries *[]core.FormType) {
	params := url.Values{}

	for _, q := range *queries {
		if q.Checked {
			params.Add(q.Key, q.Value)
		}
	}

	parsedURL, err := url.Parse(g.urlInput.Text)

	if err != nil {
		log.Println(err)
		return
	}

	if len(params) < 1 {
		g.urlInput.SetText(parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path)
		//g.urlInput.Refresh()
		return
	}

	g.urlInput.SetText(parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path + "?" + params.Encode())
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

func (g *gui) formBlock(fields *[]core.FormType) fyne.CanvasObject {

	var list *widget.List
	list = widget.NewList(func() int {
		return len(*fields)
	}, func() fyne.CanvasObject {
		parameterEntry := widget.NewEntry()
		parameterEntry.SetPlaceHolder("Form")
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
		entry.SetChecked((*fields)[lii].Checked)
		entry.OnChanged = func(b bool) {
			(*fields)[lii].Checked = b
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			(*fields) = append((*fields)[:lii], (*fields)[lii+1:]...)
			list.Refresh()
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.SetText((*fields)[lii].Key)
		parameter.OnChanged = func(s string) {
			(*fields)[lii].Key = s
		}

		value := entryCtx.Objects[1].(*widget.Entry)
		value.SetText((*fields)[lii].Value)
		value.OnChanged = func(s string) {
			(*fields)[lii].Value = s
		}
	})

	return list
}
