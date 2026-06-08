package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/ui/stimulus"
)

// Popover renders a popover panel positioned below the trigger
// element identified by triggerID. A popover is open exactly while
// the view renders it: close is delivered by the Escape key and by
// clicking the backdrop outside the panel, and should remove the
// popover from the app state.
func Popover[Msg any](close Msg, triggerID string, attrs ...domi.Attr) domi.Element {
	backdropID := triggerID + "-popover"
	return func(children ...domi.Node) domi.Node {
		return html.Div(
			attr.ID(backdropID),
			Class("u-popover"),
			stimulus.Controller("popover"),
			stimulus.Value("popover", "trigger")(triggerID),
			onEscape(close),
			onClickSelf(close),
		)(
			html.Div(
				Class("u-popover-panel"),
				group(attrs...),
			)(children...),
		)
	}
}
