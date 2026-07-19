package codegen

import (
	"path/filepath"
	"strings"

	"github.com/vardanabhanot/myapi/core"
)

type JSGenerator struct{}

func (g JSGenerator) Name() string {
	return "JavaScript (fetch)"
}

func (g JSGenerator) Generate(request *core.Request) string {
	var pre []string  // setup lines before the fetch call (FormData)
	var opts []string // lines inside the fetch options object

	if request.Method != "GET" {
		opts = append(opts, "\tmethod: "+scriptQuote(request.Method)+",")
	}

	var headerLines []string
	hasContentType := false
	if request.Headers != nil {
		for _, h := range *request.Headers {
			if !h.Checked || h.Key == "" {
				continue
			}
			if strings.EqualFold(h.Key, "content-type") {
				hasContentType = true
			}
			headerLines = append(headerLines, "\t\t"+scriptQuote(h.Key)+": "+scriptQuote(h.Value)+",")
		}
	}

	if request.Auth != nil {
		switch request.AuthType {
		case "Basic":
			if request.Auth.BasicUser != "" {
				headerLines = append(headerLines,
					"\t\t'Authorization': 'Basic ' + btoa("+scriptQuote(request.Auth.BasicUser+":"+request.Auth.BasicPass)+"),")
			}
		case "Bearer":
			if request.Auth.BearerAuth != "" {
				prefix := request.Auth.BearerPrefix
				if prefix == "" {
					prefix = "Bearer"
				}
				headerLines = append(headerLines, "\t\t'Authorization': "+scriptQuote(prefix+" "+request.Auth.BearerAuth)+",")
			}
		}
	}

	// contentType mirrors what SendRequest sets for each body type
	var bodyLine string
	rawBody := func(contentType, data string) {
		if !hasContentType {
			headerLines = append(headerLines, "\t\t'Content-Type': "+scriptQuote(contentType)+",")
		}
		bodyLine = "\tbody: " + scriptQuote(data) + ","
	}

	switch request.BodyType {
	case "JSON":
		if request.Body.Json != "" {
			rawBody("application/json", request.Body.Json)
		}
	case "XML":
		if request.Body.Xml != "" {
			rawBody("application/xml", request.Body.Xml)
		}
	case "Text":
		if request.Body.Text != "" {
			rawBody("text/plain", request.Body.Text)
		}
	case "Form":
		if request.Body.Form != nil {
			var appends []string
			hasFile := false
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" {
					if f.IsFile {
						hasFile = true
						appends = append(appends, "form.append("+scriptQuote(f.Key)+", new Blob([fs.readFileSync("+scriptQuote(f.Value)+")]), "+scriptQuote(filepath.Base(f.Value))+");")
						continue
					}
					appends = append(appends, "form.append("+scriptQuote(f.Key)+", "+scriptQuote(f.Value)+");")
				}
			}
			if len(appends) > 0 {
				if hasFile {
					pre = append(pre, "const fs = require('node:fs'); // file fields need Node")
				}
				pre = append(pre, "const form = new FormData();")
				pre = append(pre, appends...)
				bodyLine = "\tbody: form," // fetch sets the multipart Content-Type itself
			}
		}
	case "URL Encoded":
		if request.Body.Form != nil {
			var fields []string
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" && !f.IsFile {
					fields = append(fields, "\t\t"+scriptQuote(f.Key)+": "+scriptQuote(f.Value)+",")
				}
			}
			if len(fields) > 0 {
				bodyLine = "\tbody: new URLSearchParams({\n" + strings.Join(fields, "\n") + "\n\t}),"
			}
		}
	}

	if len(headerLines) > 0 {
		opts = append(opts, "\theaders: {\n"+strings.Join(headerLines, "\n")+"\n\t},")
	}
	if bodyLine != "" {
		opts = append(opts, bodyLine)
	}

	var out []string
	if request.Settings.SkipTLSVerify {
		out = append(out, "// note: fetch cannot skip TLS verification (curl -k)")
	}
	out = append(out, pre...)

	if len(opts) == 0 {
		out = append(out, "const response = await fetch("+scriptQuote(request.URL)+");")
	} else {
		out = append(out, "const response = await fetch("+scriptQuote(request.URL)+", {\n"+strings.Join(opts, "\n")+"\n});")
	}

	out = append(out, "console.log(await response.text());")

	return strings.Join(out, "\n")
}

func init() {
	Register(JSGenerator{})
}
