package ui

import "ily.dev/act3/html"

func Grip() html.Node {
	return html.Div(Class("u-sortable-handle"))(Icon("line/dots-grid"))
}
