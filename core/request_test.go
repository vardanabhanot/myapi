package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testRequest(id, url string) *Request {
	return &Request{ID: id, Method: "GET", URL: url, Headers: &[]FormType{}}
}

func TestSendRequestURLEncodedBody(t *testing.T) {
	var gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer server.Close()

	req := testRequest("urlenctest", server.URL)
	req.Method = "POST"
	req.BodyType = "URL Encoded"
	req.Body = Body{Form: &[]FormType{
		{Checked: true, Key: "a", Value: "1 2"},
		{Checked: true, Key: "b", Value: "&="},
		{Checked: false, Key: "off", Value: "x"},
		{Checked: true, Key: "", Value: "no key"},
	}}
	defer DeleteHistory("urlenctest")

	if _, err := req.SendRequest(context.Background()); err != nil {
		t.Fatal(err)
	}

	if gotContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("content type: %q", gotContentType)
	}
	if gotBody != "a=1+2&b=%26%3D" {
		t.Fatalf("body: %q", gotBody)
	}
}

func TestSettingsRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		w.Write([]byte("landed"))
	}))
	defer server.Close()

	// Default: follows to the 200
	req := testRequest("redirfollow", server.URL+"/redirect")
	defer DeleteHistory("redirfollow")
	res, err := req.SendRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Status, "200") {
		t.Fatalf("default should follow redirect, got %s", res.Status)
	}

	// NoFollowRedirects: returns the 302 itself
	req = testRequest("redirstop", server.URL+"/redirect")
	req.Settings.NoFollowRedirects = true
	defer DeleteHistory("redirstop")
	res, err = req.SendRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Status, "302") {
		t.Fatalf("NoFollowRedirects should return the 302, got %s", res.Status)
	}
}

func TestSettingsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	req := testRequest("timeouttest", server.URL)
	req.Settings.TimeoutSec = 1
	if _, err := req.SendRequest(context.Background()); err == nil {
		t.Fatal("TimeoutSec=1 against a 2s handler should error")
	}
}

func TestSettingsSkipTLSVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Self-signed cert: fails without the skip
	req := testRequest("tlsfail", server.URL)
	if _, err := req.SendRequest(context.Background()); err == nil {
		t.Fatal("self-signed cert should fail without SkipTLSVerify")
	}

	req = testRequest("tlsskip", server.URL)
	req.Settings.SkipTLSVerify = true
	defer DeleteHistory("tlsskip")
	res, err := req.SendRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Status, "200") {
		t.Fatalf("SkipTLSVerify should succeed, got %s", res.Status)
	}
}
