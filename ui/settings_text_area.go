package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// SettingsTextArea renders an inline-updating textarea.
// It POSTs to action with a form body containing all
// child hidden inputs plus the textarea's value on blur.
// Children become hidden inputs (for context like IDs).
//
// When addr is non-empty, the form element gets data-live
// and data-addr0, data-addr1, etc. attributes.
// A LiveTextUpdate with matching addr dispatches a live:update
// event that the Stimulus controller picks up
// to update the textarea and its saved original value.
func SettingsTextArea(action, name, value string, attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(
			Class("u-settings-text-area"),
			stimulus.Controller("settings-text-area"),
			stimulus.Value("settings-text-area", "url")(action),
			group(attrs...),
		)(append(nodes,
			html.Textarea(
				Class("u-settings-text-area-input"),
				attr.Name(name),
				Attr("data-optimistic")(""),
				stimulus.Target("settings-text-area", "input"),
				stimulus.Action("blur->settings-text-area#save"),
				stimulus.Action("keydown->settings-text-area#keydown"),
			)(html.Text(value)),
		)...)
	}
}
