package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

const dialogController = "dialog"

// DialogButton renders a button that GETs url as a turbo stream.
// The server responds with a turbo stream that appends
// the dialog to the [Port],
// keeping it outside any replaceable content regions.
func DialogButton(url string, attrs ...attr.Node) html.Element {
	return func(children ...html.Node) html.Node {
		return Button(
			stimulus.Controller("dialog-trigger"),
			stimulus.Value("dialog-trigger", "url")(url),
			stimulus.Action("click->dialog-trigger#open"),
			group(attrs...),
		)(children...)
	}
}

// DialogStream renders a dialog and wraps it in a turbo stream
// append to the [Port].
func DialogStream(children ...html.Node) html.Node {
	return dialogStream(html.Div(Class("u-dialog-panel")), children)
}

// ImageDialogStream renders a dialog whose panel is sized to the
// largest box of the given aspect ratio that fits the viewport
// (minus a 3rem gutter on each side). Use for dialogs whose contents
// are a single image of known intrinsic aspect.
func ImageDialogStream(aspectW, aspectH int) html.Element {
	panel := html.Div(
		Class("u-dialog-panel-image"),
		Stylef("--aspect-w: %d; --aspect-h: %d", aspectW, aspectH),
	)
	return func(children ...html.Node) html.Node {
		return dialogStream(panel, children)
	}
}

func dialogStream(panel html.Element, children []html.Node) html.Node {
	return turbo.Append("port",
		html.Dialog(
			Class("u-dialog"),
			stimulus.Controller(dialogController),
			stimulus.Action("click->dialog#close:self"),
			stimulus.Action("keydown.esc@document->dialog#close"),
			stimulus.Action("turbo:before-visit@document->dialog#close"),
		)(
			html.Div(Class("u-dialog-positioner"))(
				panel(
					Group(children...),
					html.Div(
						Class("u-dialog-close"),
						stimulus.Action("click->dialog#close"),
					)(
						Icon("line/x-close"),
					),
				),
			),
		),
	)
}
