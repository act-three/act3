//go:build ignore

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
)

//go:embed tag.txt
var tags string

var out = flag.String("output", "tag.go", "`file` to write")

func main() {
	flag.Parse()
	err := gen()
	if err != nil {
		log.Fatal(err)
	}
}

func gen() error {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "package html")
	fmt.Fprintln(b, "var (")
	fmt.Fprintf(b, "Doctype =Raw(%q)\n", "<!doctype html>")
	s := bufio.NewScanner(strings.NewReader(tags))
	for s.Scan() {
		name, amb, _ := strings.Cut(s.Text(), " ")
		fmt.Fprintf(b, "%s%s=Tag(%q)\n", strings.Title(name), amb, name)
	}
	if s.Err() != nil {
		return s.Err()
	}
	fmt.Fprintln(b, ")")

	code, err := format.Source(b.Bytes())
	if err != nil {
		return err
	}

	return os.WriteFile(*out, code, 0666)
}
