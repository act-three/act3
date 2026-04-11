package ui

import (
	"crypto/rand"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

// PopoverButton returns a form that GETs url as a turbo stream,
// storing the trigger button's position for the popover to anchor to.
// The label appears on the button;
// children are placed in the form alongside it (e.g. hidden inputs).
func PopoverButton(url string, label html.Node, attrs ...attr.Node) html.Element {
	id := "pt-" + rand.Text()[:8]
	return func(children ...html.Node) html.Node {
		return html.Form(
			attr.Method("get"),
			attr.Action(url),
			stimulus.Controller("popover-trigger"),
			stimulus.Action("submit->popover-trigger#open"),
		)(
			append(children,
				Hidden("popover-trigger", id),
				Button(
					attr.ID(id),
					stimulus.Action("mousedown->popover-trigger#activate"),
					stimulus.Action("mouseleave->popover-trigger#deactivate"),
					attr.Group(attrs...),
				)(label),
			)...,
		)
	}
}

// PopoverStream renders a popover panel and wraps it in a turbo stream
// append to the [Port].
// The panel is positioned below the trigger button
// identified by triggerID.
func PopoverStream(triggerID string, children ...html.Node) html.Node {
	return turbo.Append("port",
		html.Div(
			attr.Class("u-popover"),
			stimulus.Controller("popover"),
			stimulus.Value("popover", "trigger")(triggerID),
			stimulus.Action("click->popover#close:self"),
			stimulus.Action("keydown.esc@document->popover#close"),
			stimulus.Action("turbo:before-visit@document->popover#close"),
		)(
			html.Div(attr.Class("u-popover-panel"))(
				children...,
			),
		),
	)
}
