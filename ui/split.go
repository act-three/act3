package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/turbo"
)

func Split(attrs ...attr.Node) func(list, detail html.Node) html.Node {
	return func(list, detail html.Node) html.Node {
		return html.Div(
			attr.Class("flex flex-1 h-1"),
			attr.Group(attrs...),
		)(
			// TODO(april): refactor list stuff and frame target stuff;
			// pull it out of ui and put it in view
			turbo.Frame("item-list",
				attr.Target("detail"),
				attr.Class(`
					w-90
					h-cvh
					flex-none
					border-e
					border-gray-6
					grid
					place-items-center
				`),
			)(
				list,
			),
			turbo.Frame("detail", turbo.DataAction("advance"))(detail),
		)
	}
}
