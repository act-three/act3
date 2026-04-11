package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// SettingsToggle renders an inline-updating switch control.
// It POSTs to action with a form field named name
// carrying the boolean value "true" or "false".
// Children become hidden inputs inside the form (for context like IDs).
func SettingsToggle(action, name string, checked bool, attrs ...attr.Node) html.Element {
	val := "false"
	aria := "false"
	if checked {
		val = "true"
		aria = "true"
	}
	return func(nodes ...html.Node) html.Node {
		return html.Form(
			attr.Class("u-settings-toggle"),
			stimulus.Controller("settings-toggle"),
			stimulus.Value("settings-toggle", "url")(action),
			attr.Group(attrs...),
		)(append(nodes,
			html.Input(
				attr.Type("hidden"),
				attr.Name(name),
				attr.Value(val),
				stimulus.Target("settings-toggle", "input"),
			),
			html.Button(
				attr.Class("u-settings-toggle-track"),
				attr.Type("button"),
				attr.Role("switch"),
				attr.Attr("aria-checked")(aria),
				attr.Attr("data-optimistic")(""),
				stimulus.Target("settings-toggle", "track"),
				stimulus.Action("click->settings-toggle#toggle"),
			)(
				html.Span(attr.Class("u-settings-toggle-thumb")),
			),
		)...)
	}
}
