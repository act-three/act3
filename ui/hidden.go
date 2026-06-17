package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"
)

func Hidden(name, value string) domi.Node {
	return html.Input(attr.Type("hidden"), attr.Name(name), attr.Value(value))
}
