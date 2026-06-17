package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/event"
	"ily.dev/domi/html"

	"ily.dev/act3/ui/stimulus"
)

const dialogController = "dialog"

// Dialog renders a modal dialog. A dialog is open exactly while the
// view renders it: close is delivered by the dialog's X button and
// by the Escape key, and should remove the dialog from the app
// state.
func Dialog[Msg any](close Msg, attrs ...domi.Attr) domi.Element {
	return dialog(close, html.Div(Class("u-dialog-panel")), attrs)
}

// ImageDialog is [Dialog] with its panel sized to the largest box of
// the given aspect ratio that fits the viewport (minus a 3rem gutter
// on each side). Use for dialogs whose contents are a single image
// of known intrinsic aspect.
func ImageDialog[Msg any](close Msg, a Aspect, attrs ...domi.Attr) domi.Element {
	panel := html.Div(
		Class("u-dialog-panel-image"),
		Stylef("--aspect-w: %d; --aspect-h: %d", a.W, a.H),
	)
	return dialog(close, panel, attrs)
}

func dialog[Msg any](close Msg, panel domi.Element, attrs []domi.Attr) domi.Element {
	return func(children ...domi.Node) domi.Node {
		return html.Dialog(
			Class("u-dialog"),
			stimulus.Controller(dialogController),
			onEscape(close),
			group(attrs...),
		)(
			html.Div(Class("u-dialog-positioner"))(
				panel(
					Group(children...),
					html.Div(
						Class("u-dialog-close"),
						event.Click(close),
					)(
						Icon("line/x-close"),
					),
				),
			),
		)
	}
}
