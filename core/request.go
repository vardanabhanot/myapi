package core

import (
	"bytes"
	"fmt"
	"io"
	"log"
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
	Body        string      `json:"Body"`
	AuthType    string      `json:"AuthType"`
}

type FormType struct {
	Checked bool   `json:"Checked"`
	Key     string `json:"Key"`
	Value   string `json:"Value"`
}

type Response struct {
	Body     string
	Headers  map[string]string
	Status   string
	Duration time.Duration
	Size     string
}

func (r *Request) SendRequest() (*Response, error) {
	req, err := http.NewRequest(r.Method, r.URL, nil)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, header := range *r.Headers {
		if !header.Checked || header.Key == "" || header.Value == "" {
			continue
		}

		req.Header.Set(header.Key, header.Value)
	}

	if r.Body != "" {
		switch r.BodyType {
		case "JSON":
			req.Header.Set("Content-Type", "application/json")

		case "XML":
			req.Header.Set("Content-Type", "application/xml")

		case "Text":
			req.Header.Set("Content-Type", "text/plain")

		}

		req.Body = io.NopCloser(bytes.NewBuffer([]byte(r.Body)))
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

	res.Size = fmt.Sprint(len(body))

	if _, err = saveRequestData(r); err != nil {
		return nil, err
	}

	return res, nil
}
