package codegen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vardanabhanot/myapi/core"
)

type GoGenerator struct{}

func (g GoGenerator) Name() string {
	return "Go (net/http)"
}

func (g GoGenerator) Generate(request *core.Request) string {
	imports := map[string]bool{"fmt": true, "io": true, "net/http": true}
	var setup []string // body construction, before http.NewRequest
	var after []string // header/auth lines, after http.NewRequest
	bodyExpr := "nil"

	hasContentType := false
	if request.Headers != nil {
		for _, h := range *request.Headers {
			if !h.Checked || h.Key == "" {
				continue
			}
			if strings.EqualFold(h.Key, "content-type") {
				hasContentType = true
			}
			after = append(after, "req.Header.Set("+strconv.Quote(h.Key)+", "+strconv.Quote(h.Value)+")")
		}
	}

	if request.Auth != nil {
		switch request.AuthType {
		case "Basic":
			if request.Auth.BasicUser != "" {
				after = append(after, "req.SetBasicAuth("+strconv.Quote(request.Auth.BasicUser)+", "+strconv.Quote(request.Auth.BasicPass)+")")
			}
		case "Bearer":
			if request.Auth.BearerAuth != "" {
				prefix := request.Auth.BearerPrefix
				if prefix == "" {
					prefix = "Bearer"
				}
				after = append(after, "req.Header.Set(\"Authorization\", "+strconv.Quote(prefix+" "+request.Auth.BearerAuth)+")")
			}
		}
	}

	// contentType mirrors what SendRequest sets for each body type
	rawBody := func(contentType, data string) {
		imports["strings"] = true
		setup = append(setup, "body := strings.NewReader("+strconv.Quote(data)+")")
		bodyExpr = "body"
		if !hasContentType {
			after = append(after, "req.Header.Set(\"Content-Type\", "+strconv.Quote(contentType)+")")
		}
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
			fileN := 0
			for _, f := range *request.Body.Form {
				if !f.Checked || f.Key == "" {
					continue
				}
				if f.IsFile {
					imports["os"] = true
					n := strconv.Itoa(fileN)
					fileN++
					fields = append(fields,
						"part"+n+", _ := form.CreateFormFile("+strconv.Quote(f.Key)+", "+strconv.Quote(filepath.Base(f.Value))+")",
						"file"+n+", _ := os.Open("+strconv.Quote(f.Value)+")",
						"io.Copy(part"+n+", file"+n+")",
						"file"+n+".Close()")
					continue
				}
				fields = append(fields, "form.WriteField("+strconv.Quote(f.Key)+", "+strconv.Quote(f.Value)+")")
			}
			if len(fields) > 0 {
				imports["bytes"] = true
				imports["mime/multipart"] = true
				setup = append(setup, "var buf bytes.Buffer", "form := multipart.NewWriter(&buf)")
				setup = append(setup, fields...)
				setup = append(setup, "form.Close()")
				bodyExpr = "&buf"
				after = append(after, "req.Header.Set(\"Content-Type\", form.FormDataContentType())")
			}
		}
	case "URL Encoded":
		if request.Body.Form != nil {
			var fields []string
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" && !f.IsFile {
					fields = append(fields, "form.Set("+strconv.Quote(f.Key)+", "+strconv.Quote(f.Value)+")")
				}
			}
			if len(fields) > 0 {
				imports["net/url"] = true
				imports["strings"] = true
				setup = append(setup, "form := url.Values{}")
				setup = append(setup, fields...)
				setup = append(setup, "body := strings.NewReader(form.Encode())")
				bodyExpr = "body"
				if !hasContentType {
					after = append(after, "req.Header.Set(\"Content-Type\", \"application/x-www-form-urlencoded\")")
				}
			}
		}
	}

	clientExpr := "http.DefaultClient"
	var clientFields []string
	if request.Settings.TimeoutSec > 0 {
		imports["time"] = true
		clientFields = append(clientFields, fmt.Sprintf("Timeout: %d * time.Second", request.Settings.TimeoutSec))
	}
	if request.Settings.SkipTLSVerify {
		imports["crypto/tls"] = true
		clientFields = append(clientFields, "Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}")
	}
	if len(clientFields) > 0 {
		setup = append(setup, "client := &http.Client{"+strings.Join(clientFields, ", ")+"}")
		clientExpr = "client"
	}

	paths := make([]string, 0, len(imports))
	for p := range imports {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	b.WriteString("package main\n\nimport (\n")
	for _, p := range paths {
		b.WriteString("\t" + strconv.Quote(p) + "\n")
	}
	b.WriteString(")\n\nfunc main() {\n")

	body := append([]string{}, setup...)
	body = append(body,
		"req, err := http.NewRequest("+strconv.Quote(request.Method)+", "+strconv.Quote(request.URL)+", "+bodyExpr+")",
		"if err != nil {",
		"\tpanic(err)",
		"}")
	body = append(body, after...)
	body = append(body,
		"",
		"resp, err := "+clientExpr+".Do(req)",
		"if err != nil {",
		"\tpanic(err)",
		"}",
		"defer resp.Body.Close()",
		"",
		"data, _ := io.ReadAll(resp.Body)",
		"fmt.Println(string(data))")

	for _, line := range body {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString("\t" + line + "\n")
	}
	b.WriteString("}")

	return b.String()
}

func init() {
	Register(GoGenerator{})
}
