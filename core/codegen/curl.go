package codegen

import (
	"fmt"

	"github.com/vardanabhanot/myapi/core"
)

type CurlGenerator struct{}

func (g CurlGenerator) Name() string {
	return "cURL"
}

func (g CurlGenerator) Generate(request *core.Request) string {
	return fmt.Sprintf(`curl -X %s "%s"`, request.Method, request.URL)
}

func init() {
	Register(CurlGenerator{})
}
