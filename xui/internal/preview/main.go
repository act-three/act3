// Command preview renders the ui package's components to a standalone HTML
// page on stdout, for eyeballing layout in a browser:
//
//	go run ./xui/internal/preview > /tmp/ui-preview.html
package main

import (
	"os"

	"ily.dev/act3/xui/internal/fixture"
)

func main() {
	html, err := fixture.Document()
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.WriteString(html); err != nil {
		panic(err)
	}
}
