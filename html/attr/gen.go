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

//go:embed attr.txt
var attrs string

var out = flag.String("output", "all.go", "`file` to write")

func main() {
	flag.Parse()
	err := gen()
	if err != nil {
		log.Fatal(err)
	}
}

func gen() error {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "package attr")
	fmt.Fprintln(b, "var (")
	s := bufio.NewScanner(strings.NewReader(attrs))
	for s.Scan() {
		name, amb, _ := strings.Cut(s.Text(), " ")
		ident := strings.Title(name)
		if name == "id" {
			ident = "ID"
		}
		fmt.Fprintf(b, "%s%s=Attr(%q)\n", ident, amb, name)
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
