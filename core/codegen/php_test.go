package codegen

import (
	"strings"
	"testing"

	"github.com/vardanabhanot/myapi/core"
)

func TestPHPGenerate(t *testing.T) {
	req := &core.Request{
		Method:   "POST",
		URL:      "https://api.example.com/items",
		Headers:  &[]core.FormType{{Checked: true, Key: "X-Token", Value: "it's secret"}},
		BodyType: "JSON",
		Body:     core.Body{Json: `{"a":"b"}`},
		AuthType: "Bearer",
		Auth:     &core.Auth{BearerAuth: "tok123"},
		Settings: core.Settings{SkipTLSVerify: true},
	}

	out := PHPGenerator{}.Generate(req)

	for _, want := range []string{
		"curl_init('https://api.example.com/items')",
		"CURLOPT_CUSTOMREQUEST, 'POST'",
		"CURLOPT_SSL_VERIFYPEER, false",
		`'X-Token: it\'s secret',`,
		"'Authorization: Bearer tok123',",
		"'Content-Type: application/json',",
		`CURLOPT_POSTFIELDS, '{"a":"b"}'`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPHPGenerateURLEncoded(t *testing.T) {
	req := &core.Request{
		Method:   "POST",
		URL:      "https://x.test/login",
		BodyType: "URL Encoded",
		Body:     core.Body{Form: &[]core.FormType{{Checked: true, Key: "user", Value: "bob"}}},
	}

	out := PHPGenerator{}.Generate(req)
	if !strings.Contains(out, "'user' => 'bob',") || !strings.Contains(out, "http_build_query($fields)") {
		t.Errorf("urlencoded body wrong:\n%s", out)
	}
}
