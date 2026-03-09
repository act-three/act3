package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/ui/icon"
)

func Icon(name string) html.Node {
	return html.Raw(icon.SVG(name))
}
