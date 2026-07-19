package ui

import (
	"image/color"
	"log"
	"net/url"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
	"github.com/vardanabhanot/myapi/core/codegen"
)

type bodyOptHolder struct {
	json fyne.CanvasObject
	xml  fyne.CanvasObject
	text fyne.CanvasObject
	form fyne.CanvasObject
}

func (g *gui) makeRequestUI(request *core.Request) fyne.CanvasObject {
	var status, size, resTime, bodyResponse string
	bindings := g.tabs[request.ID].bindings
	bindings.status = binding.BindString(&status)
	bindings.size = binding.BindString(&size)
	bindings.time = binding.BindString(&resTime)
	bindings.body = binding.BindString(&bodyResponse)
	bindings.headers = binding.NewStringList()
	bindings.cookies = binding.NewStringList()
	bindings.timings = binding.NewUntyped()

	// Query options
	if request.QueryParams == nil {
		request.QueryParams = &[]core.FormType{}
		*request.QueryParams = append(*request.QueryParams, core.FormType{Checked: true})
	}

	fields := g.queryBlock(request.QueryParams)

	queryHeading := sectionHeader("Query Parameters")

	addQueryBtn := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		*request.QueryParams = append(*request.QueryParams, core.FormType{})
		fields.Refresh()
	})
	addQueryBtn.Importance = widget.LowImportance

	queryContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, queryHeading, addQueryBtn, nil),
			nil, nil, nil,
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

	headerFields := g.headerBlock(request.Headers)

	headerHeading := sectionHeader("Request Headers")

	addHeaderBtn := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		*request.Headers = append(*request.Headers, core.FormType{Checked: true})
		headerFields.Refresh()
	})
	addHeaderBtn.Importance = widget.LowImportance

	headerContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, headerHeading, addHeaderBtn, nil),
			nil, nil, nil,
			headerFields,
		),
	)

	if request.Auth == nil {
		request.Auth = &core.Auth{}
	}

	// Auth Options; authViews maps radio value → form, shown/hidden in the
	// radio callback so adding an auth type is one map entry.
	authViews := map[string]fyne.CanvasObject{}
	authViews["None"] = container.NewVBox(widget.NewLabel("No Authentication Selected"))

	basicUsername := widget.NewEntry()
	basicUsername.SetPlaceHolder("Username")
	basicPassword := widget.NewEntry()
	basicPassword.SetPlaceHolder("Password")
	basicPassword.Password = true
	basicHeading := sectionHeader("Basic Authentication")

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

	authViews["Basic"] = container.NewBorder(
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

	bearerHeading := sectionHeader("Bearer Authentication")
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

	authViews["Bearer"] = container.NewBorder(
		bearerHeading,
		nil,
		nil,
		nil,
		container.NewVBox(
			bearerTokenArea,
			container.NewBorder(nil, nil, widget.NewLabel("Token Prefix"), nil, bearerPrefix),
		),
	)

	// API Key
	apiKeyName := widget.NewEntry()
	apiKeyName.SetPlaceHolder("X-API-Key")
	apiKeyName.SetText(request.Auth.APIKeyName)
	apiKeyName.OnChanged = func(s string) {
		request.Auth.APIKeyName = s
	}

	apiKeyValue := widget.NewEntry()
	apiKeyValue.SetPlaceHolder("Value")
	apiKeyValue.SetText(request.Auth.APIKeyValue)
	apiKeyValue.OnChanged = func(s string) {
		request.Auth.APIKeyValue = s
	}

	apiKeyIn := widget.NewSelect([]string{"Header", "Query"}, func(s string) {
		request.Auth.APIKeyIn = s
	})
	if request.Auth.APIKeyIn != "" {
		apiKeyIn.SetSelected(request.Auth.APIKeyIn)
	} else {
		apiKeyIn.SetSelected("Header")
	}

	authViews["API Key"] = container.NewBorder(
		sectionHeader("API Key"),
		nil,
		nil,
		nil,
		container.NewVBox(
			container.NewBorder(nil, nil, widget.NewLabel("Key"), nil, apiKeyName),
			container.NewBorder(nil, nil, widget.NewLabel("Value"), nil, apiKeyValue),
			container.NewBorder(nil, nil, widget.NewLabel("Add to"), nil, apiKeyIn),
		),
	)

	// OAuth2 (client credentials); token fetched and cached at send time
	oauthTokenURL := widget.NewEntry()
	oauthTokenURL.SetPlaceHolder("https://auth.example.com/oauth/token")
	oauthTokenURL.SetText(request.Auth.OAuthTokenURL)
	oauthTokenURL.OnChanged = func(s string) {
		request.Auth.OAuthTokenURL = s
	}

	oauthClientID := widget.NewEntry()
	oauthClientID.SetText(request.Auth.OAuthClientID)
	oauthClientID.OnChanged = func(s string) {
		request.Auth.OAuthClientID = s
	}

	oauthClientSecret := widget.NewEntry()
	oauthClientSecret.Password = true
	oauthClientSecret.SetText(request.Auth.OAuthClientSecret)
	oauthClientSecret.OnChanged = func(s string) {
		request.Auth.OAuthClientSecret = s
	}

	oauthScope := widget.NewEntry()
	oauthScope.SetPlaceHolder("Optional, space separated")
	oauthScope.SetText(request.Auth.OAuthScope)
	oauthScope.OnChanged = func(s string) {
		request.Auth.OAuthScope = s
	}

	authViews["OAuth2"] = container.NewBorder(
		sectionHeader("OAuth 2.0 — Client Credentials"),
		nil,
		nil,
		nil,
		container.NewVBox(
			container.NewBorder(nil, nil, widget.NewLabel("Token URL"), nil, oauthTokenURL),
			container.NewBorder(nil, nil, widget.NewLabel("Client ID"), nil, oauthClientID),
			container.NewBorder(nil, nil, widget.NewLabel("Client Secret"), nil, oauthClientSecret),
			container.NewBorder(nil, nil, widget.NewLabel("Scope"), nil, oauthScope),
		),
	)

	authOptionView := container.NewStack()
	for _, view := range authViews {
		view.Hide()
		authOptionView.Add(view)
	}

	// TODO:: Will need to implement AWS as well here
	authOptions := widget.NewRadioGroup([]string{"None", "Basic", "Bearer", "API Key", "OAuth2"}, func(value string) {
		request.AuthType = value

		for name, view := range authViews {
			if name == value {
				view.Show()
			} else {
				view.Hide()
			}
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

	jsonTextArea := g.newAppEntry()
	jsonTextArea.SetText(request.Body.Json)

	jsonTextArea.MultiLine = true
	jsonTextArea.SetMinRowsVisible(5)
	jsonTextArea.TextStyle.Monospace = true
	jsonTextArea.OnChanged = func(s string) {
		request.Body.Json = s
	}

	bodyOptIns.json = jsonTextArea

	xmlTextArea := g.newAppEntry()
	xmlTextArea.SetText(request.Body.Xml)

	xmlTextArea.MultiLine = true
	xmlTextArea.SetMinRowsVisible(7)
	xmlTextArea.TextStyle.Monospace = true
	xmlTextArea.OnChanged = func(s string) {
		request.Body.Xml = s
	}

	bodyOptIns.xml = xmlTextArea

	textTextArea := g.newAppEntry()
	textTextArea.SetText(request.Body.Text)

	textTextArea.OnChanged = func(s string) {
		request.Body.Text = s
	}
	textTextArea.MultiLine = true
	textTextArea.SetMinRowsVisible(7)
	textTextArea.TextStyle.Monospace = true

	bodyOptIns.text = textTextArea

	if request.Body.Form == nil {
		request.Body.Form = &[]core.FormType{}
		*request.Body.Form = append(*request.Body.Form, core.FormType{Checked: true})
	}

	formFieldsBlock := g.formBlock(request.Body.Form)

	formHeading := sectionHeader("Form Fields")

	addFormBtn := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		*request.Body.Form = append(*request.Body.Form, core.FormType{})
		formFieldsBlock.Refresh()
	})
	addFormBtn.Importance = widget.LowImportance

	formContainer := container.NewPadded(
		container.NewBorder(
			container.NewBorder(nil, nil, formHeading, addFormBtn, nil),
			nil, nil, nil,
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

	// "Form" is multipart; "URL Encoded" shares the same key/value rows and
	// only changes how the body is encoded at send time.
	bodyOptions := widget.NewRadioGroup([]string{"JSON", "Form", "URL Encoded", "XML", "Text"}, func(value string) {
		request.BodyType = value

		switch value {
		case "JSON":
			bodyOptIns.json.Show()
			bodyOptIns.xml.Hide()
			bodyOptIns.text.Hide()
			bodyOptIns.form.Hide()

		case "Form", "URL Encoded":
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

	// BodyType is never empty here (defaulted above); the radio callback
	// shows the matching editor and hides the rest.
	bodyOptions.SetSelected(request.BodyType)

	bodyOptionView := container.NewStack(
		bodyOptIns.json,
		bodyOptIns.xml,
		bodyOptIns.text,
		bodyOptIns.form,
	)

	bodyContainer := container.NewPadded(
		container.NewBorder(bodyOptions, nil, nil, nil, bodyOptionView),
	)

	// Per-request settings; zero values = default behaviour (30s, follow
	// redirects, verify TLS), so old saved requests load unchanged.
	timeoutEntry := widget.NewEntry()
	timeoutEntry.SetPlaceHolder("30")
	if request.Settings.TimeoutSec > 0 {
		timeoutEntry.SetText(strconv.Itoa(request.Settings.TimeoutSec))
	}
	timeoutEntry.OnChanged = func(s string) {
		request.Settings.TimeoutSec, _ = strconv.Atoi(s) // invalid/empty → 0 → default
		request.IsDirty = true
	}

	redirectCheck := widget.NewCheck("Don't follow redirects", nil)
	redirectCheck.SetChecked(request.Settings.NoFollowRedirects)
	redirectCheck.OnChanged = func(b bool) {
		request.Settings.NoFollowRedirects = b
		request.IsDirty = true
	}

	tlsCheck := widget.NewCheck("Skip TLS certificate verification", nil)
	tlsCheck.SetChecked(request.Settings.SkipTLSVerify)
	tlsCheck.OnChanged = func(b bool) {
		request.Settings.SkipTLSVerify = b
		request.IsDirty = true
	}

	settingsContainer := container.NewPadded(container.NewVBox(
		sectionHeader("Request Settings"),
		container.NewBorder(nil, nil, widget.NewLabel("Timeout (seconds)"), nil, timeoutEntry),
		redirectCheck,
		tlsCheck,
	))

	// Code Gen drawer
	var codePreviewContainer *fyne.Container

	languageOption := codegen.GetSupportedLanguages()
	var codePreview *widget.TextGrid
	codeBackGround := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	codeBackGround.CornerRadius = 6
	codePreview = widget.NewTextGrid()
	codePreview.ShowLineNumbers = true
	codePreview.ShowWhitespace = true
	codePreview.Scroll = fyne.ScrollHorizontalOnly

	languageSelect := widget.NewSelect(languageOption, func(s string) {
		code, err := codegen.GenerateCode(s, request)

		if err != nil {
			return
		}

		codePreview.SetText(code)
	})

	languageSelect.SetSelectedIndex(0)
	copyCode := copyFeedbackButton(func() string { return codePreview.Text() })

	codePreviewContainer = container.NewPadded(container.NewVBox(
		container.NewBorder(nil, nil, nil, copyCode, languageSelect),
		container.NewStack(
			codeBackGround,
			container.NewPadded(codePreview)),
	))

	codeGenTitle := sectionHeader("Code Generator")

	// Drawer on the right edge, hidden by default; the spacer fixes its width
	drawerSpacer := canvas.NewRectangle(color.Transparent)
	drawerSpacer.SetMinSize(fyne.NewSize(380, 0))
	codeContainer := container.NewStack(
		drawerSpacer,
		container.NewBorder(nil, nil, widget.NewSeparator(), nil,
			container.NewBorder(container.NewPadded(codeGenTitle), nil, nil, nil, codePreviewContainer),
		),
	)
	codeContainer.Hide()

	var requestArea *fyne.Container
	var codeIconTappable *widget.Button
	codeIconTappable = widget.NewButtonWithIcon("", theme.NewThemedResource(resourceCodeSvg), func() {
		if codeContainer.Visible() {
			codeContainer.Hide()
			codeIconTappable.Importance = widget.LowImportance
		} else {
			// Regenerate so the drawer reflects the current request state
			languageSelect.OnChanged(languageSelect.Selected)
			codeContainer.Show()
			codeIconTappable.Importance = widget.MediumImportance
		}
		codeIconTappable.Refresh()
		requestArea.Refresh()
	})
	codeIconTappable.Importance = widget.LowImportance

	requestArea = container.NewBorder(nil, nil, nil, codeContainer, container.NewStack(
		container.NewAppTabs(
			container.NewTabItem("Query", queryContainer),
			container.NewTabItem("Headers", headerContainer),
			container.NewTabItem("Auth", authContainer),
			container.NewTabItem("Body", bodyContainer),
			container.NewTabItem("Settings", settingsContainer),
		),
		container.NewBorder(container.NewBorder(nil, nil, nil, codeIconTappable), nil, nil, nil),
	))

	return requestArea
}

func (g *gui) queryBlock(queries *[]core.FormType) fyne.CanvasObject {

	var list *widget.List
	list = widget.NewList(func() int {
		return len(*queries)
	}, func() fyne.CanvasObject {
		parameterEntry := widget.NewEntry()
		parameterEntry.SetPlaceHolder("Parameter")
		valueEntry := widget.NewEntry()
		valueEntry.SetPlaceHolder("Value")

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

		// Detach callbacks before Set*: recycled rows still hold the
		// previous row's callback, which would fire with stale indexes.
		entry := ctx.Objects[1].(*widget.Check)
		entry.OnChanged = nil
		entry.SetChecked((*queries)[lii].Checked)
		entry.OnChanged = func(b bool) {
			(*queries)[lii].Checked = b

			if g.urlInput.Text == "" {
				return
			}

			g.updateURL(queries)
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			*queries = append((*queries)[:lii], (*queries)[lii+1:]...)
			list.Refresh()

			g.updateURL(queries)
		}

		// Typing in the last row appends a fresh empty row
		autoAppend := func(lii int) {
			if lii == len(*queries)-1 {
				*queries = append(*queries, core.FormType{Checked: true})
				list.Refresh()
			}
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.OnChanged = nil
		parameter.SetText((*queries)[lii].Key)
		parameter.OnChanged = func(s string) {
			(*queries)[lii].Key = s
			if s != "" {
				autoAppend(lii)
			}

			if !(*queries)[lii].Checked {
				return
			}

			if g.urlInput.Text == "" {
				return
			}

			g.updateURL(queries)
		}
		value := entryCtx.Objects[1].(*widget.Entry)
		value.OnChanged = nil
		value.SetText((*queries)[lii].Value)
		value.OnChanged = func(s string) {
			(*queries)[lii].Value = s
			if s != "" {
				autoAppend(lii)
			}

			if !(*queries)[lii].Checked {
				return
			}

			g.updateURL(queries)
		}
	})

	g.queryList = list // urlInput.OnChanged refreshes it after syncing rows
	return list
}

func (g *gui) updateURL(queries *[]core.FormType) {
	if g.urlInput.Text == "" {
		return
	}

	params := url.Values{}

	for _, q := range *queries {
		if q.Checked && q.Key != "" {
			params.Add(q.Key, q.Value)
		}
	}

	parsedURL, err := url.Parse(g.urlInput.Text)

	if err != nil {
		log.Println(err)
		return
	}

	base := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	if len(params) > 0 {
		base += "?" + params.Encode()
	}

	// guard: the query tab is the source of truth here, don't let the
	// URL OnChanged sync mutate the rows being edited
	g.syncingQuery = true
	g.urlInput.SetText(base)
	g.syncingQuery = false
}

// syncQueryParams merges the query string of a hand-edited URL into the
// query tab rows: updates values for existing keys, re-checks rows the URL
// re-added, drops checked rows no longer in the URL, keeps unchecked rows
// (they only live in the tab), and keeps a trailing empty row for typing.
func syncQueryParams(rows *[]core.FormType, values url.Values) {
	kept := (*rows)[:0]
	for _, r := range *rows {
		if r.Key == "" {
			continue // empty rows are re-added below
		}
		if v, inURL := values[r.Key]; inURL {
			r.Value = v[0]
			r.Checked = true
			kept = append(kept, r)
			delete(values, r.Key)
		} else if !r.Checked {
			kept = append(kept, r)
		}
	}

	for key, v := range values {
		kept = append(kept, core.FormType{Checked: true, Key: key, Value: v[0]})
	}

	*rows = append(kept, core.FormType{Checked: true})
}

func (g *gui) headerBlock(headers *[]core.FormType) fyne.CanvasObject {

	var list *widget.List
	list = widget.NewList(func() int {
		return len(*headers)
	}, func() fyne.CanvasObject {
		parameterEntry := widget.NewEntry()
		parameterEntry.SetPlaceHolder("Header")
		valueEntry := widget.NewEntry()
		valueEntry.SetPlaceHolder("Value")

		return container.NewBorder(nil, nil,
			widget.NewCheck("", func(b bool) {}),
			widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
			container.NewGridWithColumns(2, parameterEntry, valueEntry),
		)
	}, func(lii widget.ListItemID, co fyne.CanvasObject) {
		ctx, _ := co.(*fyne.Container)

		// Detach callbacks before Set*: recycled rows still hold the
		// previous row's callback, which would fire with stale indexes.
		entry := ctx.Objects[1].(*widget.Check)
		entry.OnChanged = nil
		entry.SetChecked((*headers)[lii].Checked)
		entry.OnChanged = func(b bool) {
			(*headers)[lii].Checked = b
		}

		btn := ctx.Objects[2].(*widget.Button)
		btn.OnTapped = func() {
			(*headers) = append((*headers)[:lii], (*headers)[lii+1:]...)
			list.Refresh()
		}

		// Typing in the last row appends a fresh empty row
		autoAppend := func(lii int) {
			if lii == len(*headers)-1 {
				*headers = append(*headers, core.FormType{Checked: true})
				list.Refresh()
			}
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.OnChanged = nil
		parameter.SetText((*headers)[lii].Key)
		parameter.OnChanged = func(s string) {
			(*headers)[lii].Key = s
			if s != "" {
				autoAppend(lii)
			}
		}

		value := entryCtx.Objects[1].(*widget.Entry)
		value.OnChanged = nil
		value.SetText((*headers)[lii].Value)
		value.OnChanged = func(s string) {
			(*headers)[lii].Value = s
			if s != "" {
				autoAppend(lii)
			}
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
		parameterEntry.SetPlaceHolder("Key")
		valueEntry := widget.NewEntry()
		valueEntry.SetPlaceHolder("Value")

		return container.NewBorder(nil, nil,
			widget.NewCheck("", func(b bool) {}),
			container.NewHBox(
				widget.NewButtonWithIcon("", theme.FileIcon(), func() {}),
				widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
			),
			container.NewGridWithColumns(2, parameterEntry, valueEntry),
		)
	}, func(lii widget.ListItemID, co fyne.CanvasObject) {
		ctx, _ := co.(*fyne.Container)

		// Detach callbacks before Set*: recycled rows still hold the
		// previous row's callback, which would fire with stale indexes.
		entry := ctx.Objects[1].(*widget.Check)
		entry.OnChanged = nil
		entry.SetChecked((*fields)[lii].Checked)
		entry.OnChanged = func(b bool) {
			(*fields)[lii].Checked = b
		}

		btns := ctx.Objects[2].(*fyne.Container)

		// File toggle: pick a file to make this a file row (value = path),
		// tap again to clear back to a text field. Only "Form" (multipart)
		// sends file rows; URL Encoded skips them at send time.
		fileBtn := btns.Objects[0].(*widget.Button)
		if (*fields)[lii].IsFile {
			fileBtn.SetIcon(theme.CancelIcon())
		} else {
			fileBtn.SetIcon(theme.FileIcon())
		}
		fileBtn.OnTapped = func() {
			if (*fields)[lii].IsFile {
				(*fields)[lii].IsFile = false
				(*fields)[lii].Value = ""
				list.Refresh()
				return
			}

			dialog.ShowFileOpen(func(rc fyne.URIReadCloser, err error) {
				if err != nil || rc == nil {
					return
				}
				rc.Close() // only the path is needed; SendRequest opens it fresh
				(*fields)[lii].IsFile = true
				(*fields)[lii].Value = rc.URI().Path()
				list.Refresh()
			}, *g.Window)
		}

		btn := btns.Objects[1].(*widget.Button)
		btn.OnTapped = func() {
			(*fields) = append((*fields)[:lii], (*fields)[lii+1:]...)
			list.Refresh()
		}

		// Typing in the last row appends a fresh empty row
		autoAppend := func(lii int) {
			if lii == len(*fields)-1 {
				*fields = append(*fields, core.FormType{Checked: true})
				list.Refresh()
			}
		}

		entryCtx, _ := ctx.Objects[0].(*fyne.Container)

		parameter := entryCtx.Objects[0].(*widget.Entry)
		parameter.OnChanged = nil
		parameter.SetText((*fields)[lii].Key)
		parameter.OnChanged = func(s string) {
			(*fields)[lii].Key = s
			if s != "" {
				autoAppend(lii)
			}
		}

		value := entryCtx.Objects[1].(*widget.Entry)
		value.OnChanged = nil
		value.SetText((*fields)[lii].Value)
		// File rows show the picked path read-only; recycled rows must be
		// re-enabled explicitly.
		if (*fields)[lii].IsFile {
			value.Disable()
		} else {
			value.Enable()
		}
		value.OnChanged = func(s string) {
			(*fields)[lii].Value = s
			if s != "" {
				autoAppend(lii)
			}
		}
	})

	return list
}
