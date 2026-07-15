package codegen

import (
	"fmt"
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
	return gen.Generate(request.ResolveEnv()), nil
}

// scriptQuote single-quotes s for JavaScript and Python — the two share
// the same escapes for single-quoted string literals.
func scriptQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return "'" + r.Replace(s) + "'"
}
