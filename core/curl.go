package core

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ParseCurl turns a pasted "Copy as cURL" command (bash or Windows cmd
// flavour) into a Request ready for makeTab. The returned request has a
// fresh ID and IsDirty set.
// ponytail: no $'...' escape decoding and no @file reading; the literal
// text lands in the body instead. Upgrade if real captures need it.
func ParseCurl(cmd string) (*Request, error) {
	tokens, err := tokenizeCurl(cmd)
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 || tokens[0] != "curl" {
		return nil, errors.New("not a curl command")
	}
	tokens = tokens[1:]

	// flags that take a value but mean nothing to us — the value must
	// still be consumed or it would be mistaken for the URL
	ignoredWithArg := map[string]bool{
		"-o": true, "--output": true,
		"-A": true, "--user-agent": true,
		"-e": true, "--referer": true,
		"-m": true, "--max-time": true,
		"--connect-timeout": true, "--retry": true,
		"--cacert": true, "--capath": true,
	}

	req := &Request{ID: NewRequestID(), Method: "GET", IsDirty: true}
	var headers []FormType
	var formRows []FormType
	var cookies []string
	var dataParts []string
	var reqURL, contentType string

	next := func(i *int, flag string) (string, error) {
		*i++
		if *i >= len(tokens) {
			return "", fmt.Errorf("%s needs a value", flag)
		}
		return tokens[*i], nil
	}

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t {
		case "-X", "--request":
			v, err := next(&i, t)
			if err != nil {
				return nil, err
			}
			req.Method = strings.ToUpper(v)

		case "-H", "--header":
			v, err := next(&i, t)
			if err != nil {
				return nil, err
			}
			key, val, _ := strings.Cut(v, ":")
			key, val = strings.TrimSpace(key), strings.TrimSpace(val)
			if strings.EqualFold(key, "content-type") {
				contentType = strings.ToLower(val)
			}
			headers = append(headers, FormType{Checked: true, Key: key, Value: val})

		case "-d", "--data", "--data-raw", "--data-binary", "--data-urlencode":
			v, err := next(&i, t)
			if err != nil {
				return nil, err
			}
			dataParts = append(dataParts, v)

		case "-F", "--form":
			v, err := next(&i, t)
			if err != nil {
				return nil, err
			}
			key, val, _ := strings.Cut(v, "=")
			// curl's -F key=@path means "upload the file at path"
			isFile := strings.HasPrefix(val, "@")
			formRows = append(formRows, FormType{Checked: true, Key: key, Value: strings.TrimPrefix(val, "@"), IsFile: isFile})

		case "-u", "--user":
			v, err := next(&i, t)
			if err != nil {
				return nil, err
			}
			user, pass, _ := strings.Cut(v, ":")
			req.AuthType = "Basic"
			req.Auth = &Auth{BasicUser: user, BasicPass: pass}

		case "-b", "--cookie":
			v, err := next(&i, t)
			if err != nil {
				return nil, err
			}
			cookies = append(cookies, v)

		case "-k", "--insecure":
			req.Settings.SkipTLSVerify = true

		case "--url":
			v, err := next(&i, t)
			if err != nil {
				return nil, err
			}
			reqURL = v

		default:
			if ignoredWithArg[t] {
				i++ // skip the value too
				continue
			}
			if strings.HasPrefix(t, "-") {
				continue // --compressed, -s, -L and friends
			}
			if reqURL == "" {
				reqURL = t
			}
		}
	}

	if reqURL == "" {
		return nil, errors.New("no URL in curl command")
	}
	req.URL = reqURL

	if len(cookies) > 0 {
		headers = append(headers, FormType{Checked: true, Key: "Cookie", Value: strings.Join(cookies, "; ")})
	}
	if len(headers) > 0 {
		req.Headers = &headers
	}

	// Mirror the URL's query string into the query tab, like the UI does
	// as you type.
	if u, err := url.Parse(reqURL); err == nil && u.RawQuery != "" {
		var rows []FormType
		for _, kv := range strings.Split(u.RawQuery, "&") {
			k, v, _ := strings.Cut(kv, "=")
			uk, errK := url.QueryUnescape(k)
			uv, errV := url.QueryUnescape(v)
			if errK != nil || errV != nil {
				uk, uv = k, v
			}
			rows = append(rows, FormType{Checked: true, Key: uk, Value: uv})
		}
		req.QueryParams = &rows
	}

	data := strings.Join(dataParts, "&") // curl joins multiple -d with &

	switch {
	case len(formRows) > 0:
		req.BodyType = "Form"
		req.Body.Form = &formRows

	case data != "":
		switch {
		case strings.Contains(contentType, "json"):
			req.BodyType = "JSON"
			req.Body.Json = data
		case strings.Contains(contentType, "xml"):
			req.BodyType = "XML"
			req.Body.Xml = data
		case contentType == "" || strings.Contains(contentType, "x-www-form-urlencoded"):
			// -d with no content type is urlencoded in curl semantics
			if rows, ok := parseURLEncodedBody(data); ok {
				req.BodyType = "URL Encoded"
				req.Body.Form = &rows
			} else {
				req.BodyType = "Text"
				req.Body.Text = data
			}
		default:
			req.BodyType = "Text"
			req.Body.Text = data
		}
	}

	// -d/-F without an explicit -X means POST, like curl itself
	if req.Method == "GET" && (data != "" || len(formRows) > 0) {
		req.Method = "POST"
	}

	return req, nil
}

// parseURLEncodedBody splits "a=1&b=2" into form rows; ok is false when the
// data doesn't look like a form body.
func parseURLEncodedBody(data string) ([]FormType, bool) {
	var rows []FormType
	for _, kv := range strings.Split(data, "&") {
		k, v, found := strings.Cut(kv, "=")
		if !found || k == "" {
			return nil, false
		}
		uk, errK := url.QueryUnescape(k)
		uv, errV := url.QueryUnescape(v)
		if errK != nil || errV != nil {
			return nil, false
		}
		rows = append(rows, FormType{Checked: true, Key: uk, Value: uv})
	}
	return rows, len(rows) > 0
}

// tokenizeCurl is a shell-ish splitter: single/double quotes, $'...'
// (treated like single quotes), backslash escapes, and both bash (\<newline>)
// and cmd (^<newline>) line continuations.
func tokenizeCurl(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inToken := false

	flush := func() {
		if inToken {
			tokens = append(tokens, cur.String())
			cur.Reset()
			inToken = false
		}
	}

	for i := 0; i < len(s); i++ {
		c := s[i]

		switch {
		case c == '\\' || c == '^':
			// line continuation: swallow the newline
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
				continue
			}
			if i+2 < len(s) && s[i+1] == '\r' && s[i+2] == '\n' {
				i += 2
				continue
			}
			if c == '^' { // ^ is only special before a newline
				cur.WriteByte(c)
				inToken = true
				continue
			}
			if i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i++
			}
			inToken = true

		case c == '\'', c == '$' && i+1 < len(s) && s[i+1] == '\'':
			if c == '$' {
				i++ // $'...' — skip the $, then read like single quotes
			}
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return nil, errors.New("unterminated quote")
			}
			cur.WriteString(s[i+1 : i+1+end])
			i += end + 1
			inToken = true

		case c == '"':
			i++
			for ; i < len(s); i++ {
				if s[i] == '"' {
					break
				}
				if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
					i++
				}
				cur.WriteByte(s[i])
			}
			if i >= len(s) {
				return nil, errors.New("unterminated quote")
			}
			inToken = true

		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()

		default:
			cur.WriteByte(c)
			inToken = true
		}
	}
	flush()

	return tokens, nil
}
