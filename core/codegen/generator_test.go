package codegen

import (
	"strings"
	"testing"

	"github.com/vardanabhanot/myapi/core"
)

func TestJSGenerate(t *testing.T) {
	req := &core.Request{
		Method:   "POST",
		URL:      "https://api.example.com/items",
		Headers:  &[]core.FormType{{Checked: true, Key: "X-Token", Value: "abc"}},
		BodyType: "JSON",
		Body:     core.Body{Json: `{"a":"b"}`},
		AuthType: "Basic",
		Auth:     &core.Auth{BasicUser: "alice", BasicPass: "pw"},
	}

	out := JSGenerator{}.Generate(req)
	for _, want := range []string{
		"await fetch('https://api.example.com/items', {",
		"method: 'POST',",
		"'X-Token': 'abc',",
		"'Authorization': 'Basic ' + btoa('alice:pw'),",
		"'Content-Type': 'application/json',",
		`body: '{"a":"b"}',`,
		"console.log(await response.text());",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// bare GET stays a one-liner
	bare := JSGenerator{}.Generate(&core.Request{Method: "GET", URL: "https://x.test/"})
	if !strings.Contains(bare, "await fetch('https://x.test/');") {
		t.Errorf("bare GET:\n%s", bare)
	}
}

func TestPythonGenerate(t *testing.T) {
	req := &core.Request{
		Method:   "POST",
		URL:      "https://x.test/login",
		BodyType: "URL Encoded",
		Body:     core.Body{Form: &[]core.FormType{{Checked: true, Key: "user", Value: "bob"}}},
		AuthType: "Bearer",
		Auth:     &core.Auth{BearerAuth: "tok"},
		Settings: core.Settings{SkipTLSVerify: true, TimeoutSec: 10},
	}

	out := PythonGenerator{}.Generate(req)
	for _, want := range []string{
		"import requests",
		"requests.post(",
		"'https://x.test/login',",
		"'Authorization': 'Bearer tok',",
		"'user': 'bob',",
		"verify=False,",
		"timeout=10,",
		"print(response.text)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// GenerateCode must substitute {{var}} placeholders so snippets run as-is.
func TestGenerateCodeResolvesEnv(t *testing.T) {
	core.SetActiveVars(map[string]string{"base": "https://real.example.com", "key": "s3cret"})
	defer core.SetActiveVars(nil)

	req := &core.Request{
		Method:  "GET",
		URL:     "{{base}}/things",
		Headers: &[]core.FormType{{Checked: true, Key: "X-Api-Key", Value: "{{key}}"}},
	}

	for _, lang := range GetSupportedLanguages() {
		out, err := GenerateCode(lang, req)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "{{") {
			t.Errorf("%s: unresolved placeholder in:\n%s", lang, out)
		}
		if !strings.Contains(out, "https://real.example.com/things") {
			t.Errorf("%s: URL not substituted:\n%s", lang, out)
		}
	}

	// the source request must keep its placeholders
	if req.URL != "{{base}}/things" || (*req.Headers)[0].Value != "{{key}}" {
		t.Fatalf("GenerateCode mutated the request: %+v", req)
	}
}

func TestGoGenerate(t *testing.T) {
	req := &core.Request{
		Method:   "POST",
		URL:      "https://api.example.com/items",
		Headers:  &[]core.FormType{{Checked: true, Key: "X-Token", Value: "abc"}},
		BodyType: "JSON",
		Body:     core.Body{Json: `{"a":"b"}`},
		AuthType: "Basic",
		Auth:     &core.Auth{BasicUser: "alice", BasicPass: "pw"},
		Settings: core.Settings{SkipTLSVerify: true, TimeoutSec: 10},
	}

	out := GoGenerator{}.Generate(req)
	for _, want := range []string{
		"package main",
		`"net/http"`,
		`"crypto/tls"`,
		"body := strings.NewReader(\"{\\\"a\\\":\\\"b\\\"}\")",
		`client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}`,
		`req, err := http.NewRequest("POST", "https://api.example.com/items", body)`,
		`req.Header.Set("X-Token", "abc")`,
		`req.SetBasicAuth("alice", "pw")`,
		`req.Header.Set("Content-Type", "application/json")`,
		"resp, err := client.Do(req)",
		"data, _ := io.ReadAll(resp.Body)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// bare GET: nil body, default client
	bare := GoGenerator{}.Generate(&core.Request{Method: "GET", URL: "https://x.test/"})
	for _, want := range []string{
		`req, err := http.NewRequest("GET", "https://x.test/", nil)`,
		"resp, err := http.DefaultClient.Do(req)",
	} {
		if !strings.Contains(bare, want) {
			t.Errorf("bare GET missing %q in:\n%s", want, bare)
		}
	}
}

// API Key and OAuth2 are folded into plain headers/query before generation,
// so every generator emits them without knowing the auth types exist.
func TestGenerateCodeNormalizesAuth(t *testing.T) {
	apiHeader := &core.Request{
		Method:   "GET",
		URL:      "https://x.test/a",
		AuthType: "API Key",
		Auth:     &core.Auth{APIKeyName: "X-API-Key", APIKeyValue: "k1"},
	}
	out, err := GenerateCode("cURL", apiHeader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "X-API-Key: k1") {
		t.Errorf("header API key missing:\n%s", out)
	}

	apiQuery := &core.Request{
		Method:   "GET",
		URL:      "https://x.test/a?p=1",
		AuthType: "API Key",
		Auth:     &core.Auth{APIKeyName: "api_key", APIKeyValue: "k2", APIKeyIn: "Query"},
	}
	out, err = GenerateCode("cURL", apiQuery)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "api_key=k2") || !strings.Contains(out, "p=1") {
		t.Errorf("query API key missing or existing params dropped:\n%s", out)
	}
	if apiQuery.URL != "https://x.test/a?p=1" {
		t.Fatalf("GenerateCode mutated the request URL: %s", apiQuery.URL)
	}

	oauth := &core.Request{
		Method:   "GET",
		URL:      "https://x.test/a",
		AuthType: "OAuth2",
		Auth:     &core.Auth{OAuthTokenURL: "https://x.test/token", OAuthClientID: "id"},
	}
	out, err = GenerateCode("cURL", oauth)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Authorization: Bearer YOUR_ACCESS_TOKEN") {
		t.Errorf("OAuth2 placeholder missing:\n%s", out)
	}
}

func TestGetSupportedLanguagesSorted(t *testing.T) {
	langs := GetSupportedLanguages()
	if len(langs) != 5 {
		t.Fatalf("expected 5 generators, got %v", langs)
	}
	for i := 1; i < len(langs); i++ {
		if langs[i-1] > langs[i] {
			t.Fatalf("not sorted: %v", langs)
		}
	}
}
