package core

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Request struct {
	ID          string      `json:"ID"`
	Name        string      `json:"Name"` // user-given label; empty → UI falls back to method+path
	Method      string      `json:"Method"`
	URL         string      `json:"URL"`
	QueryParams *[]FormType `json:"QueryParams"`
	Headers     *[]FormType `json:"Headers"`
	BodyType    string      `json:"BodyType"`
	Body        Body        `json:"Body"`
	AuthType    string      `json:"AuthType"`
	Auth        *Auth       `json:"Auth"`
	Settings    Settings    `json:"Settings"`
	MTime       string      `json:"-"`
	IsDirty     bool        `json:"-"`
}

// Settings holds per-request transport options. Fields are named so the Go
// zero value means default behaviour — old saved requests unmarshal to zero.
type Settings struct {
	TimeoutSec        int  `json:"TimeoutSec"` // 0 → 30s default
	NoFollowRedirects bool `json:"NoFollowRedirects"`
	SkipTLSVerify     bool `json:"SkipTLSVerify"`
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
	IsFile  bool   `json:"IsFile,omitempty"` // multipart Form rows only: Value is a file path
}

type Response struct {
	Body     string
	Headers  map[string]string
	Cookies  []*http.Cookie
	Status   string
	Duration time.Duration
	Size     string
	Timings  Timings
}

// Timings holds the phase breakdown of a request. DNS/Connect/TLS are zero
// when the transport reused a connection. TTFB is measured from request
// start, so it contains the earlier phases plus server wait.
type Timings struct {
	DNS      time.Duration
	Connect  time.Duration
	TLS      time.Duration
	TTFB     time.Duration
	Download time.Duration
	Total    time.Duration
}

func (r *Request) SendRequest(ctx context.Context) (*Response, error) {
	// {{var}} substitution happens here at send time so the saved request
	// keeps its placeholders.
	req, err := http.NewRequest(r.Method, ApplyEnv(r.URL), nil)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	req = req.WithContext(ctx)

	for _, header := range *r.Headers {
		if !header.Checked || header.Key == "" || header.Value == "" {
			continue
		}

		req.Header.Set(ApplyEnv(header.Key), ApplyEnv(header.Value))
	}

	// Setting Basic Auth in the request
	if r.AuthType == "Basic" && r.Auth.BasicPass != "" && r.Auth.BasicUser != "" {
		req.SetBasicAuth(ApplyEnv(r.Auth.BasicUser), ApplyEnv(r.Auth.BasicPass))
	}

	// Setting Bearer auth
	if r.AuthType == "Bearer" && r.Auth.BearerAuth != "" && r.Auth.BearerPrefix != "" {
		req.Header.Add("Authorization", r.Auth.BearerPrefix+" "+ApplyEnv(r.Auth.BearerAuth))
	}

	if r.Body.Json != "" || r.Body.Xml != "" || r.Body.Text != "" || r.Body.Form != nil {
		switch r.BodyType {
		case "JSON":
			req.Header.Set("Content-Type", "application/json")
			req.Body = io.NopCloser(bytes.NewBufferString(ApplyEnv(r.Body.Json)))

		case "XML":
			req.Header.Set("Content-Type", "application/xml")
			req.Body = io.NopCloser(bytes.NewBufferString(ApplyEnv(r.Body.Xml)))

		case "Text":
			req.Header.Set("Content-Type", "text/plain")
			req.Body = io.NopCloser(bytes.NewBufferString(ApplyEnv(r.Body.Text)))

		case "Form":
			// ponytail: whole multipart body is buffered in RAM; io.Pipe
			// streaming if huge uploads ever matter.
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			for _, v := range *r.Body.Form {
				if !v.Checked {
					continue
				}

				if v.IsFile {
					path := ApplyEnv(v.Value)
					f, err := os.Open(path)
					if err != nil {
						return nil, err
					}

					part, err := writer.CreateFormFile(ApplyEnv(v.Key), filepath.Base(path))
					if err == nil {
						_, err = io.Copy(part, f)
					}
					f.Close()
					if err != nil {
						return nil, err
					}
					continue
				}

				writer.WriteField(ApplyEnv(v.Key), ApplyEnv(v.Value))
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())
			writer.Close()

			req.Body = io.NopCloser(&b)

		case "URL Encoded":
			values := url.Values{}
			for _, v := range *r.Body.Form {
				if v.Checked && v.Key != "" && !v.IsFile { // file rows can't be urlencoded
					values.Add(ApplyEnv(v.Key), ApplyEnv(v.Value))
				}
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Body = io.NopCloser(strings.NewReader(values.Encode()))
		}

	}

	// Zero-value settings keep the old defaults: 30s timeout, follow
	// redirects, verify TLS. Cancel button still works via ctx.
	timeout := 30 * time.Second
	if r.Settings.TimeoutSec > 0 {
		timeout = time.Duration(r.Settings.TimeoutSec) * time.Second
	}

	client := &http.Client{Timeout: timeout}

	if r.Settings.NoFollowRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	if r.Settings.SkipTLSVerify {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	var timings Timings
	startTime := time.Now()
	var dnsStart, connStart, tlsStart time.Time
	trace := &httptrace.ClientTrace{
		DNSStart:          func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:           func(httptrace.DNSDoneInfo) { timings.DNS = time.Since(dnsStart) },
		ConnectStart:      func(string, string) { connStart = time.Now() },
		ConnectDone:       func(string, string, error) { timings.Connect = time.Since(connStart) },
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone:  func(tls.ConnectionState, error) { timings.TLS = time.Since(tlsStart) },
		GotFirstResponseByte: func() {
			timings.TTFB = time.Since(startTime)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	response, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	res := &Response{}

	res.Headers = make(map[string]string)
	res.Cookies = response.Cookies()

	// Convert response headers to a bindable map
	for key, values := range response.Header {
		res.Headers[key] = values[0] // Get the first value for simplicity
	}

	defer response.Body.Close()

	// Cap the read: io.ReadAll grows unbounded, so a large response buffers
	// entirely into RAM (twice, counting the string copy below), spiking RSS
	// that Go only lazily returns to the OS. The UI keeps ~2MB anyway; read a
	// touch more so truncation is detectable. True size still comes from
	// Content-Length below.
	const maxBodyRead = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodyRead+1))
	if err != nil {
		log.Println("Error reading response body:", err)
		return nil, err
	}
	if len(body) > maxBodyRead {
		body = body[:maxBodyRead]
	}

	// Total now includes the body download, which the old headers-only
	// measurement missed.
	timings.Total = time.Since(startTime)
	if timings.TTFB > 0 {
		timings.Download = timings.Total - timings.TTFB
	}
	res.Duration = timings.Total
	res.Timings = timings

	res.Body = string(body)

	if response != nil && response.Status != "" {
		res.Status = response.Status
	}

	// Prefer the server's Content-Length so the reported size stays honest
	// even when we stopped reading at maxBodyRead.
	size := len(body)
	if response.ContentLength > int64(size) {
		size = int(response.ContentLength)
	}
	res.Size = bytestoHuman(size)

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
