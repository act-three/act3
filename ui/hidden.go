package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Hidden(name, value string) html.Node {
	return html.Input(attr.Type("hidden"), attr.Name(name), attr.Value(value))
}
