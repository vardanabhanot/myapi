package codegen

import (
	"strings"

	"github.com/vardanabhanot/myapi/core"
)

type CurlGenerator struct{}

func (g CurlGenerator) Name() string {
	return "cURL"
}

func (g CurlGenerator) Generate(request *core.Request) string {
	var parts []string
	parts = append(parts, "curl -X "+request.Method+" "+shellQuote(request.URL))

	if request.Settings.SkipTLSVerify {
		parts = append(parts, "-k")
	}

	if request.Auth != nil {
		switch request.AuthType {
		case "Basic":
			if request.Auth.BasicUser != "" {
				parts = append(parts, "-u "+shellQuote(request.Auth.BasicUser+":"+request.Auth.BasicPass))
			}
		case "Bearer":
			if request.Auth.BearerAuth != "" {
				prefix := request.Auth.BearerPrefix
				if prefix == "" {
					prefix = "Bearer"
				}
				parts = append(parts, "-H "+shellQuote("Authorization: "+prefix+" "+request.Auth.BearerAuth))
			}
		}
	}

	hasContentType := false
	if request.Headers != nil {
		for _, h := range *request.Headers {
			if !h.Checked || h.Key == "" {
				continue
			}
			if strings.EqualFold(h.Key, "content-type") {
				hasContentType = true
			}
			parts = append(parts, "-H "+shellQuote(h.Key+": "+h.Value))
		}
	}

	// contentType mirrors what SendRequest sets for each body type
	addBody := func(contentType, data string) {
		if !hasContentType {
			parts = append(parts, "-H "+shellQuote("Content-Type: "+contentType))
		}
		parts = append(parts, "--data-raw "+shellQuote(data))
	}

	switch request.BodyType {
	case "JSON":
		if request.Body.Json != "" {
			addBody("application/json", request.Body.Json)
		}
	case "XML":
		if request.Body.Xml != "" {
			addBody("application/xml", request.Body.Xml)
		}
	case "Text":
		if request.Body.Text != "" {
			addBody("text/plain", request.Body.Text)
		}
	case "Form":
		if request.Body.Form != nil {
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" {
					value := f.Value
					if f.IsFile {
						value = "@" + value
					}
					parts = append(parts, "-F "+shellQuote(f.Key+"="+value))
				}
			}
		}
	case "URL Encoded":
		if request.Body.Form != nil {
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" && !f.IsFile {
					parts = append(parts, "--data-urlencode "+shellQuote(f.Key+"="+f.Value))
				}
			}
		}
	}

	return strings.Join(parts, " \\\n  ")
}

// shellQuote single-quotes s for a POSIX shell; embedded ' becomes '\''.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func init() {
	Register(CurlGenerator{})
}
