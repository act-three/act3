package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

const dialogController = "dialog"

// DialogButton returns a form that GETs url as a turbo stream.
// The server responds with a turbo stream that appends
// the dialog to the [Port],
// keeping it outside any replaceable content regions.
func DialogButton(url string, attrs ...attr.Node) html.Element {
	return func(children ...html.Node) html.Node {
		return html.Form(attr.Method("get"), attr.Action(url))(
			Button(attrs...)(children...),
		)
	}
}

// DialogStream renders a dialog and wraps it in a turbo stream
// append to the [Port].
func DialogStream(children ...html.Node) html.Node {
	return turbo.Append("port",
		html.Dialog(
			attr.Class("u-dialog"),
			stimulus.Controller(dialogController),
			stimulus.Action("click->dialog#backdropClose"),
			stimulus.Action("turbo:before-visit@document->dialog#close"),
		)(
			html.Div(attr.Class("u-dialog-positioner"))(
				html.Div(attr.Class("u-dialog-panel"))(
					html.Group(children...),
					html.Div(
						attr.Class("u-dialog-close"),
						stimulus.Action("click->dialog#close"),
					)(
						Icon("line/x-close"),
					),
				),
			),
		),
	)
}
