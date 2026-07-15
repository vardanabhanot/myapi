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

func TestGetSupportedLanguagesSorted(t *testing.T) {
	langs := GetSupportedLanguages()
	if len(langs) != 4 {
		t.Fatalf("expected 4 generators, got %v", langs)
	}
	for i := 1; i < len(langs); i++ {
		if langs[i-1] > langs[i] {
			t.Fatalf("not sorted: %v", langs)
		}
	}
}
