package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/ui/stimulus"
)

// SettingsTextArea renders an inline-editing textarea showing value.
// When the user commits an edit (blur after a change), commit is
// called with the new value and the resulting message is delivered.
// Escape reverts the textarea to value.
func SettingsTextArea[Msg any](value string, commit func(value string) Msg, attrs ...domi.Attr) domi.Node {
	return html.Div(
		Class("u-settings-text-area"),
		stimulus.Controller("settings-text-area"),
		group(attrs...),
	)(
		html.Textarea(
			Class("u-settings-text-area-input"),
			onChangeValue(commit),
			stimulus.Target("settings-text-area", "input"),
			stimulus.Action("keydown->settings-text-area#keydown"),
		)(domi.Text(value)),
	)
}
