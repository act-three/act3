package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Home() html.Node {
	return media("Act Three")(
		html.Div()(html.A(attr.Href("/movies"))(html.Text("All Movies"))),
		html.Div()(html.A(attr.Href("/series"))(html.Text("All Series"))),
	)
}
