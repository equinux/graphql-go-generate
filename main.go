package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	var (
		tag string
	)
	flag.StringVar(&tag, "tag", "graphql",
		"The struct tag name of the fields that should be included.",
	)
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}
	src, err := GenerateSchemaDefinitions(args[0], tag)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(src))
}
