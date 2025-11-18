package ui

import (
	"embed"

	"ily.dev/act3/html"
)

//go:embed icon
var staticFS embed.FS

var missing = mustIcon("icon/square-dashed.svg")

func Icon(name string) html.Node {
	b, err := staticFS.ReadFile("icon/" + name + ".svg")
	if err != nil {
		return missing
	}
	return html.Raw(string(b))
}

func mustIcon(path string) html.Node {
	b, err := staticFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return html.Raw(string(b))
}
