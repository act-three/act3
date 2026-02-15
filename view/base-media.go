package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func media(title string, child ...html.Node) html.Node {
	return base(title)(
		html.Group(child...),
		html.Div(
			attr.Class("fixed inset-x-0 bottom-0 flex flex-col items-center"),
		)(
			html.Div(
				attr.Class("flex flex-col items-center gap-6 p-7 md:flex-row md:gap-8 rounded-2xl"),
			)(
				html.Div()(html.Text("Home")),
				html.Div()(html.Text("Search")),
				html.A(attr.Href("/account/profile"))(html.Text("Settings")),
			),
		),
	)
}
