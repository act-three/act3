package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Grip() domi.Node {
	return html.Div(Class("u-sortable-handle"))(Icon("line/dots-grid"))
}
