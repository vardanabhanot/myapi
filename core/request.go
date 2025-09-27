package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"
)

type Request struct {
	ID          string      `json:"ID"`
	Method      string      `json:"Method"`
	URL         string      `json:"URL"`
	QueryParams *[]FormType `json:"QueryParams"`
	Headers     *[]FormType `json:"Headers"`
	BodyType    string      `json:"BodyType"`
	Body        Body        `json:"Body"`
	AuthType    string      `json:"AuthType"`
	Auth        *Auth       `json:"Auth"`
	MTime       string      `json:"-"`
	IsDirty     bool        `json:"-"`
}

type Body struct {
	Json string      `json:"Json"`
	Text string      `json:"Text"`
	Xml  string      `json:"Xml"`
	Form *[]FormType `json:"Form"`
}

type Auth struct {
	BasicUser    string `json:"BasicUser"`
	BasicPass    string `json:"BasicPass"`
	BearerAuth   string `json:"BearerAuth"`
	BearerPrefix string `json:"BearerPrefix"`
}

type FormType struct {
	Checked bool   `json:"Checked"`
	Key     string `json:"Key"`
	Value   string `json:"Value"`
}

type Response struct {
	Body     string
	Headers  map[string]string
	Cookies  []*http.Cookie
	Status   string
	Duration time.Duration
	Size     string
}

func (r *Request) SendRequest(ctx context.Context) (*Response, error) {
	req, err := http.NewRequest(r.Method, r.URL, nil)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	req = req.WithContext(ctx)

	for _, header := range *r.Headers {
		if !header.Checked || header.Key == "" || header.Value == "" {
			continue
		}

		req.Header.Set(header.Key, header.Value)
	}

	// Setting Basic Auth in the request
	if r.AuthType == "Basic" && r.Auth.BasicPass != "" && r.Auth.BasicUser != "" {
		req.SetBasicAuth(r.Auth.BasicUser, r.Auth.BasicPass)
	}

	// Setting Bearer auth
	if r.AuthType == "Bearer" && r.Auth.BearerAuth != "" && r.Auth.BearerPrefix != "" {
		req.Header.Add("Authorization", r.Auth.BearerPrefix+" "+r.Auth.BearerAuth)
	}

	if r.Body.Json != "" || r.Body.Xml != "" || r.Body.Text != "" || r.Body.Form != nil {
		switch r.BodyType {
		case "JSON":
			req.Header.Set("Content-Type", "application/json")
			req.Body = io.NopCloser(bytes.NewBuffer([]byte(r.Body.Json)))

		case "XML":
			req.Header.Set("Content-Type", "application/xml")
			req.Body = io.NopCloser(bytes.NewBuffer([]byte(r.Body.Xml)))

		case "Text":
			req.Header.Set("Content-Type", "text/plain")
			req.Body = io.NopCloser(bytes.NewBuffer([]byte(r.Body.Text)))

		case "Form":
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			for _, v := range *r.Body.Form {
				if v.Checked {
					writer.WriteField(v.Key, v.Value)
				}
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())
			writer.Close()

			req.Body = io.NopCloser(&b)
		}

	}

	client := &http.Client{}

	startTime := time.Now()
	response, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	res := &Response{}

	endTime := time.Now()
	res.Duration = endTime.Sub(startTime)

	res.Headers = make(map[string]string)
	res.Cookies = response.Cookies()

	// Convert response headers to a bindable map
	for key, values := range response.Header {
		res.Headers[key] = values[0] // Get the first value for simplicity
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return nil, err
	}

	res.Body = string(body)

	if response != nil && response.Status != "" {
		res.Status = response.Status
	}

	res.Size = bytestoHuman(len(body))

	if _, err = saveRequestData(r); err != nil {
		return nil, err
	}

	return res, nil
}

func bytestoHuman(byteLen int) string {
	var kb_in_bytes = 1024
	var mb_in_bytes int = 1024 * kb_in_bytes
	var gb_in_bytes int = 1024 * mb_in_bytes

	if byteLen >= gb_in_bytes {
		return fmt.Sprintf("%d GB", byteLen/gb_in_bytes)
	} else if byteLen >= mb_in_bytes {
		return fmt.Sprintf("%d MB", byteLen/mb_in_bytes)
	} else if byteLen >= kb_in_bytes {
		return fmt.Sprintf("%d KB", byteLen/kb_in_bytes)
	}

	return fmt.Sprintf("%d bytes", byteLen)
}
