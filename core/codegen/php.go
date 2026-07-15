package codegen

import (
	"strings"

	"github.com/vardanabhanot/myapi/core"
)

type PHPGenerator struct{}

func (g PHPGenerator) Name() string {
	return "PHP"
}

func (g PHPGenerator) Generate(request *core.Request) string {
	var parts []string

	parts = append(parts, `<?php
$ch = curl_init(`+phpQuote(request.URL)+`);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);`)

	if request.Method != "GET" {
		parts = append(parts, "curl_setopt($ch, CURLOPT_CUSTOMREQUEST, "+phpQuote(request.Method)+");")
	}

	if request.Settings.SkipTLSVerify {
		parts = append(parts, `curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, 0);`)
	}

	if request.Auth != nil && request.AuthType == "Basic" && request.Auth.BasicUser != "" {
		parts = append(parts, "curl_setopt($ch, CURLOPT_USERPWD, "+phpQuote(request.Auth.BasicUser+":"+request.Auth.BasicPass)+");")
	}

	// CURLOPT_HTTPHEADER takes a list of "Key: Value" strings, not a map
	var headerLines []string
	hasContentType := false
	if request.Headers != nil {
		for _, v := range *request.Headers {
			if !v.Checked || v.Key == "" {
				continue
			}
			if strings.EqualFold(v.Key, "content-type") {
				hasContentType = true
			}
			headerLines = append(headerLines, v.Key+": "+v.Value)
		}
	}

	if request.Auth != nil && request.AuthType == "Bearer" && request.Auth.BearerAuth != "" {
		prefix := request.Auth.BearerPrefix
		if prefix == "" {
			prefix = "Bearer"
		}
		headerLines = append(headerLines, "Authorization: "+prefix+" "+request.Auth.BearerAuth)
	}

	// contentType mirrors what SendRequest sets for each body type
	var bodyPart string
	rawBody := func(contentType, data string) {
		if !hasContentType {
			headerLines = append(headerLines, "Content-Type: "+contentType)
		}
		bodyPart = "curl_setopt($ch, CURLOPT_POSTFIELDS, " + phpQuote(data) + ");"
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
	case "Form", "URL Encoded":
		if request.Body.Form != nil {
			var fields []string
			for _, f := range *request.Body.Form {
				if f.Checked && f.Key != "" {
					fields = append(fields, "\t"+phpQuote(f.Key)+" => "+phpQuote(f.Value)+",")
				}
			}
			if len(fields) > 0 {
				// array POSTFIELDS → multipart; http_build_query → urlencoded.
				// PHP curl sets the matching Content-Type itself either way.
				value := "$fields"
				if request.BodyType == "URL Encoded" {
					value = "http_build_query($fields)"
				}
				bodyPart = "$fields = [\n" + strings.Join(fields, "\n") + "\n];\ncurl_setopt($ch, CURLOPT_POSTFIELDS, " + value + ");"
			}
		}
	}

	if len(headerLines) > 0 {
		for i, h := range headerLines {
			headerLines[i] = "\t" + phpQuote(h) + ","
		}
		parts = append(parts, "$headers = [\n"+strings.Join(headerLines, "\n")+`
];
curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);`)
	}

	if bodyPart != "" {
		parts = append(parts, bodyPart)
	}

	parts = append(parts, `$response = curl_exec($ch);
curl_close($ch);
echo $response;
?>`)

	return strings.Join(parts, "\n")
}

// phpQuote single-quotes s as a PHP string literal.
func phpQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}

func init() {
	Register(PHPGenerator{})
}
