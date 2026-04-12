package ui

import (
	"encoding/json/v2"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// SettingsToggle renders an inline-updating switch control.
// It POSTs to action with a form field named name
// carrying the boolean value "true" or "false",
// alongside any additional fields in params.
func SettingsToggle(action, name string, checked bool, params map[string]string, attrs ...attr.Node) html.Node {
	aria := "false"
	if checked {
		aria = "true"
	}
	paramsJSON, _ := json.Marshal(params)
	return html.Div(
		stimulus.Controller("settings-toggle"),
		stimulus.Value("settings-toggle", "url")(action),
		stimulus.Value("settings-toggle", "name")(name),
		stimulus.Value("settings-toggle", "params")(string(paramsJSON)),
		attr.Group(attrs...),
	)(
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
	)
}
