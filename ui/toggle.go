package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// Toggle renders an inline-updating switch control.
// It POSTs to action with a form field named name
// carrying the boolean value "true" or "false".
// Children become hidden inputs inside the form (for context like IDs).
func Toggle(action, name string, checked bool, attrs ...attr.Node) html.Element {
	val := "false"
	aria := "false"
	if checked {
		val = "true"
		aria = "true"
	}
	return func(nodes ...html.Node) html.Node {
		return html.Form(
			attr.Class("u-toggle"),
			stimulus.Controller("toggle"),
			stimulus.Value("toggle", "url")(action),
			attr.Group(attrs...),
		)(append(nodes,
			html.Input(
				attr.Type("hidden"),
				attr.Name(name),
				attr.Value(val),
				stimulus.Target("toggle", "input"),
			),
			html.Button(
				attr.Class("u-toggle-track"),
				attr.Type("button"),
				attr.Role("switch"),
				attr.Attr("aria-checked")(aria),
				stimulus.Target("toggle", "track"),
				stimulus.Action("click->toggle#toggle"),
			)(
				html.Span(attr.Class("u-toggle-thumb")),
			),
		)...)
	}
}
