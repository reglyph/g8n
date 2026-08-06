package main

import (
	"fmt"

	"github.com/whoqmi/g8n/internal/parser"
)

func main() {
	s, err := parser.ParseFile(".env.schema")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", s)

	for i, f := range s.Fields {
		fmt.Printf("Fields[%d]: %+v\n", i, *f)
	}

}
