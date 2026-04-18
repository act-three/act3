package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Split(attrs ...attr.Node) func(list, detail html.Node) html.Node {
	return func(list, detail html.Node) html.Node {
		return html.Div(
			Class("u-split"),
			group(attrs...),
		)(
			html.Div(Class("u-split-list"))(list),
			detail,
		)
	}
}
