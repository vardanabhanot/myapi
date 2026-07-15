package codegen

import (
	"fmt"
	"strings"

	"github.com/vardanabhanot/myapi/core"
)

type PythonGenerator struct{}

func (g PythonGenerator) Name() string {
	return "Python (requests)"
}

func (g PythonGenerator) Generate(request *core.Request) string {
	// every method the app offers exists as a requests.<method> function
	args := []string{"\t" + scriptQuote(request.URL) + ","}

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

	var authArg string
	if request.Auth != nil {
		switch request.AuthType {
		case "Basic":
			if request.Auth.BasicUser != "" {
				authArg = "\tauth=(" + scriptQuote(request.Auth.BasicUser) + ", " + scriptQuote(request.Auth.BasicPass) + "),"
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
	var bodyArg string
	rawBody := func(contentType, data string) {
		if !hasContentType {
			headerLines = append(headerLines, "\t\t'Content-Type': "+scriptQuote(contentType)+",")
		}
		bodyArg = "\tdata=" + scriptQuote(data) + ","
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
			var fields []string
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" {
					// (None, value) is the requests idiom for a plain
					// multipart field without a filename
					fields = append(fields, "\t\t"+scriptQuote(f.Key)+": (None, "+scriptQuote(f.Value)+"),")
				}
			}
			if len(fields) > 0 {
				bodyArg = "\tfiles={\n" + strings.Join(fields, "\n") + "\n\t},"
			}
		}
	case "URL Encoded":
		if request.Body.Form != nil {
			var fields []string
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" {
					fields = append(fields, "\t\t"+scriptQuote(f.Key)+": "+scriptQuote(f.Value)+",")
				}
			}
			if len(fields) > 0 {
				bodyArg = "\tdata={\n" + strings.Join(fields, "\n") + "\n\t},"
			}
		}
	}

	if len(headerLines) > 0 {
		args = append(args, "\theaders={\n"+strings.Join(headerLines, "\n")+"\n\t},")
	}
	if bodyArg != "" {
		args = append(args, bodyArg)
	}
	if authArg != "" {
		args = append(args, authArg)
	}
	if request.Settings.SkipTLSVerify {
		args = append(args, "\tverify=False,")
	}
	if request.Settings.TimeoutSec > 0 {
		args = append(args, fmt.Sprintf("\ttimeout=%d,", request.Settings.TimeoutSec))
	}

	return "import requests\n\nresponse = requests." + strings.ToLower(request.Method) + "(\n" +
		strings.Join(args, "\n") + "\n)\nprint(response.text)"
}

func init() {
	Register(PythonGenerator{})
}
