package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/web/turbo"
)

func Split(attrs ...attr.Node) func(list, detail html.Node) html.Node {
	return func(list, detail html.Node) html.Node {
		return html.Div(
			attr.Class("flex flex-1 h-1"),
			attr.Group(attrs...),
		)(
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
			turbo.Frame("detail", turbo.TurboAction("advance"))(detail),
		)
	}
}
