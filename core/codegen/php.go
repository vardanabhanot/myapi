package codegen

import (
	"fmt"
	"strings"

	"github.com/vardanabhanot/myapi/core"
)

type PHPGenerator struct{}

func (g PHPGenerator) Name() string {
	return "PHP"
}

func (g PHPGenerator) Generate(request *core.Request) string {
	var parts []string

	parts = append(parts, fmt.Sprintf(`<?php
$ch = curl_init("%s");
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);`, request.URL))

	// CURLOPT_HTTPHEADER takes a list of "Key: Value" strings, not a map
	if request.Headers != nil {
		var lines []string
		for _, v := range *request.Headers {
			if v.Checked {
				lines = append(lines, fmt.Sprintf("\t'%s: %s',", v.Key, v.Value))
			}
		}

		if len(lines) > 0 {
			parts = append(parts, "$headers = [\n"+strings.Join(lines, "\n")+`
];
curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);`)
		}
	}

	parts = append(parts, `$response = curl_exec($ch);
curl_close($ch);
echo $response;
?>`)

	return strings.Join(parts, "\n")
}

func init() {
	Register(PHPGenerator{})
}
