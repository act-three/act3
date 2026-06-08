package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Split(attrs ...domi.Attr) func(list, detail domi.Node) domi.Node {
	return func(list, detail domi.Node) domi.Node {
		return html.Div(
			Class("u-split"),
			group(attrs...),
		)(
			html.Div(Class("u-split-list"))(list),
			detail,
		)
	}
}
