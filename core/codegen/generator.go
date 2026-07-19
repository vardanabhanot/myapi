package codegen

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/vardanabhanot/myapi/core"
)

type CodeGenerator interface {
	Generate(*core.Request) string
	Name() string
}

var registry = map[string]CodeGenerator{}

func Register(generator CodeGenerator) {
	registry[generator.Name()] = generator
}

func GetSupportedLanguages() []string {
	langs := make([]string, 0, len(registry))
	for name := range registry {
		langs = append(langs, name)
	}
	sort.Strings(langs) // map order is random; keep the dropdown stable
	return langs
}

func GenerateCode(language string, request *core.Request) (string, error) {
	gen, ok := registry[language]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", language)
	}
	// Resolve {{var}} placeholders once here so every generator emits
	// runnable snippets instead of raw placeholders.
	resolved := request.ResolveEnv()
	normalizeAuth(resolved)
	return gen.Generate(resolved), nil
}

// normalizeAuth folds the auth types generators don't know about into plain
// headers/query on the resolved copy, so generators only ever see
// Basic/Bearer and never need per-type cases.
func normalizeAuth(r *core.Request) {
	if r.Auth == nil {
		return
	}
	if r.Headers == nil {
		r.Headers = &[]core.FormType{}
	}

	switch r.AuthType {
	case "API Key":
		if r.Auth.APIKeyName == "" {
			return
		}
		if r.Auth.APIKeyIn == "Query" {
			if u, err := url.Parse(r.URL); err == nil {
				q := u.Query()
				q.Set(r.Auth.APIKeyName, r.Auth.APIKeyValue)
				u.RawQuery = q.Encode()
				r.URL = u.String()
			}
		} else {
			*r.Headers = append(*r.Headers, core.FormType{Checked: true, Key: r.Auth.APIKeyName, Value: r.Auth.APIKeyValue})
		}
	case "OAuth2":
		// The real token is fetched at send time; snippets get a placeholder.
		*r.Headers = append(*r.Headers, core.FormType{Checked: true, Key: "Authorization", Value: "Bearer YOUR_ACCESS_TOKEN"})
	}
}

// scriptQuote single-quotes s for JavaScript and Python — the two share
// the same escapes for single-quoted string literals.
func scriptQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return "'" + r.Replace(s) + "'"
}
