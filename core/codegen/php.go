package codegen

import (
	"fmt"

	"github.com/vardanabhanot/myapi/core"
)

type PHPGenerator struct{}

func (g PHPGenerator) Name() string {
	return "PHP"
}

func (g PHPGenerator) Generate(request *core.Request) string {
	return fmt.Sprintf(`<?php
$ch = curl_init("%s");
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
$response = curl_exec($ch);
curl_close($ch);
echo $response;
?>`, request.URL)
}

func init() {
	Register(PHPGenerator{})
}
