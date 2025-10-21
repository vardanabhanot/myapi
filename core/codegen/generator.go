package codegen

import (
	"fmt"

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
	return langs
}

func GenerateCode(language string, request *core.Request) (string, error) {
	gen, ok := registry[language]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", language)
	}
	return gen.Generate(request), nil
}
