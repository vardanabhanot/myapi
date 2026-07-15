package codegen

import (
	"testing"

	"github.com/vardanabhanot/myapi/core"
)

// Export → import round-trip: what the generator emits, ParseCurl must
// read back to the same request essentials.
func TestCurlGenerateRoundTrip(t *testing.T) {
	req := &core.Request{
		Method:   "POST",
		URL:      "https://api.example.com/items?x=1",
		Headers:  &[]core.FormType{{Checked: true, Key: "X-Token", Value: "it's secret"}},
		BodyType: "JSON",
		Body:     core.Body{Json: `{"a":"b"}`},
		AuthType: "Basic",
		Auth:     &core.Auth{BasicUser: "alice", BasicPass: "pw"},
		Settings: core.Settings{SkipTLSVerify: true},
	}

	out := CurlGenerator{}.Generate(req)

	parsed, err := core.ParseCurl(out)
	if err != nil {
		t.Fatalf("generated curl does not parse: %v\n%s", err, out)
	}

	if parsed.Method != "POST" || parsed.URL != req.URL {
		t.Fatalf("method=%q url=%q", parsed.Method, parsed.URL)
	}
	if parsed.BodyType != "JSON" || parsed.Body.Json != req.Body.Json {
		t.Fatalf("bodytype=%q json=%q", parsed.BodyType, parsed.Body.Json)
	}
	if parsed.AuthType != "Basic" || parsed.Auth.BasicUser != "alice" || parsed.Auth.BasicPass != "pw" {
		t.Fatalf("auth: %+v", parsed.Auth)
	}
	if !parsed.Settings.SkipTLSVerify {
		t.Fatal("lost -k")
	}

	var token string
	for _, h := range *parsed.Headers {
		if h.Key == "X-Token" {
			token = h.Value
		}
	}
	if token != "it's secret" {
		t.Fatalf("header with quote lost: %q", token)
	}
}
