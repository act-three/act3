package ui

import (
	"crypto/rand"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

const dialogController = "dialog"

// DialogButton returns an element that, when called with children,
// produces a button paired with an inline turbo frame.
// Clicking the button loads the URL into the frame,
// which renders the dialog.
func DialogButton(url string, attrs ...attr.Node) html.Element {
	id := "dialog-" + rand.Text()[:8]
	return func(children ...html.Node) html.Node {
		return html.Group(
			Button(
				attr.Href(url),
				attr.Attr("data-turbo-frame")(id),
				group(attrs...),
			)(children...),
			turbo.Frame(id),
		)
	}
}

// Dialog wraps children in a native <dialog> element
// with Stimulus controller for showModal/close behavior.
func Dialog(frameID string, children ...html.Node) html.Node {
	return turbo.Frame(frameID)(
		html.Dialog(
			attr.Class("u-dialog"),
			stimulus.Controller(dialogController),
			stimulus.Action("click->dialog#backdropClose"),
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
