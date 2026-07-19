package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestSendRequestMultipartFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "upload.txt")
	if err := os.WriteFile(filePath, []byte("file-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotField, gotFileName, gotFileBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return
		}
		gotField = r.FormValue("name")
		f, fh, err := r.FormFile("doc")
		if err != nil {
			t.Errorf("form file: %v", err)
			return
		}
		defer f.Close()
		gotFileName = fh.Filename
		b, _ := io.ReadAll(f)
		gotFileBody = string(b)
	}))
	defer server.Close()

	req := testRequest("multipartfile", server.URL)
	req.Method = "POST"
	req.BodyType = "Form"
	req.Body = Body{Form: &[]FormType{
		{Checked: true, Key: "name", Value: "bob"},
		{Checked: true, Key: "doc", Value: filePath, IsFile: true},
		{Checked: false, Key: "off", Value: filePath, IsFile: true},
	}}
	defer DeleteHistory("multipartfile")

	if _, err := req.SendRequest(context.Background()); err != nil {
		t.Fatal(err)
	}

	if gotField != "bob" || gotFileName != "upload.txt" || gotFileBody != "file-bytes" {
		t.Fatalf("field=%q filename=%q body=%q", gotField, gotFileName, gotFileBody)
	}

	// Missing file should surface as an error, not a silent empty part
	req2 := testRequest("multipartmissing", server.URL)
	req2.Method = "POST"
	req2.BodyType = "Form"
	req2.Body = Body{Form: &[]FormType{{Checked: true, Key: "doc", Value: filepath.Join(dir, "gone.txt"), IsFile: true}}}
	if _, err := req2.SendRequest(context.Background()); err == nil {
		t.Fatal("missing upload file should error")
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

func TestSendRequestAPIKey(t *testing.T) {
	var gotHeader, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-API-Key")
		gotQuery = r.URL.Query().Get("api_key")
	}))
	defer server.Close()

	req := testRequest("apikeyheader", server.URL)
	req.AuthType = "API Key"
	req.Auth = &Auth{APIKeyName: "X-API-Key", APIKeyValue: "sekrit"}
	defer DeleteHistory("apikeyheader")
	if _, err := req.SendRequest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotHeader != "sekrit" {
		t.Fatalf("header api key: %q", gotHeader)
	}

	req = testRequest("apikeyquery", server.URL+"/?existing=1")
	req.AuthType = "API Key"
	req.Auth = &Auth{APIKeyName: "api_key", APIKeyValue: "qsekrit", APIKeyIn: "Query"}
	defer DeleteHistory("apikeyquery")
	if _, err := req.SendRequest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "qsekrit" {
		t.Fatalf("query api key: %q", gotQuery)
	}
}

func TestSendRequestOAuth2(t *testing.T) {
	tokenHits := 0
	var gotAuthz, gotGrant, gotBasicUser string
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		tokenHits++
		gotBasicUser, _, _ = r.BasicAuth()
		r.ParseForm()
		gotGrant = r.PostForm.Get("grant_type")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"tok123","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		gotAuthz = r.Header.Get("Authorization")
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	req := testRequest("oauthtest", server.URL+"/api")
	req.AuthType = "OAuth2"
	req.Auth = &Auth{OAuthTokenURL: server.URL + "/token", OAuthClientID: "cid", OAuthClientSecret: "csec"}
	defer DeleteHistory("oauthtest")

	for i := 0; i < 2; i++ { // second send must reuse the cached token
		if _, err := req.SendRequest(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if gotAuthz != "Bearer tok123" {
		t.Fatalf("authorization: %q", gotAuthz)
	}
	if gotGrant != "client_credentials" {
		t.Fatalf("grant_type: %q", gotGrant)
	}
	if gotBasicUser != "cid" {
		t.Fatalf("basic user: %q", gotBasicUser)
	}
	if tokenHits != 1 {
		t.Fatalf("token endpoint hit %d times, want 1 (cache)", tokenHits)
	}
}
