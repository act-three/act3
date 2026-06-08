package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"
	"ily.dev/domi/html"
)

// SettingsToggle renders a switch control reflecting checked.
// Clicking delivers commit, which should carry the opposite of
// checked; the switch re-renders from server state.
func SettingsToggle[Msg any](checked bool, commit Msg, attrs ...domi.Attr) domi.Node {
	aria := "false"
	if checked {
		aria = "true"
	}
	return html.Div(group(attrs...))(
		html.Button(
			Class("u-settings-toggle-track"),
			attr.Type("button"),
			attr.Role("switch"),
			Attr("aria-checked")(aria),
			event.Click(commit),
		)(
			html.Span(Class("u-settings-toggle-thumb")),
		),
	)
}
