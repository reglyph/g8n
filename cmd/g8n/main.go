package main

import (
	"fmt"

	"github.com/whoqmi/g8n/internal/parser"
)

func main() {
	src := `KEY=VALUE`
	s, err := parser.ParseString("test.schema", src)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", s)
}
