package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Link(url string, attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return Button(
			attr.Href(url),
			Class("hover:underline"),
			group(attrs...),
		)(nodes...).With(ButtonBorderless)
	}
}
