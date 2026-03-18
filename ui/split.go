package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/turbo"
)

func Split(attrs ...attr.Node) func(list, detail html.Node) html.Node {
	return func(list, detail html.Node) html.Node {
		return html.Div(
			attr.Class("u-split"),
			attr.Group(attrs...),
		)(
			// TODO(april): refactor list stuff and frame target stuff;
			// pull it out of ui and put it in view
			html.Div(attr.Class("u-split-list"))(
				turbo.Frame("item-list",
					turbo.Target("detail"),
				)(list),
			),
			turbo.Frame("detail", turbo.Advance())(detail),
		)
	}
}
