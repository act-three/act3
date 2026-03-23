package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// SettingsTextFieldSuffix sets a non-editable suffix
// (e.g. "min") that tracks the end of the input text.
var SettingsTextFieldSuffix = stimulus.Value("settings-text-field", "suffix")

// SettingsTextField renders an inline-updating text field.
// It POSTs to action with a form body containing all
// child hidden inputs plus the text input's value on blur.
// Children become hidden inputs (for context like IDs).
func SettingsTextField(action, name, value string, attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Form(
			attr.Class("u-settings-text-field"),
			stimulus.Controller("settings-text-field"),
			stimulus.Value("settings-text-field", "url")(action),
			attr.Group(attrs...),
		)(append(nodes,
			html.Div(attr.Class("u-settings-text-field-inner"))(
				InputText(
					attr.Name(name),
					attr.Value(value),
					stimulus.Target("settings-text-field", "input"),
					stimulus.Action("blur->settings-text-field#save"),
					stimulus.Action("keydown->settings-text-field#keydown"),
					stimulus.Action("input->settings-text-field#sync"),
				),
				html.Div(attr.Class("u-settings-text-field-overlay"))(
					InputText(
						attr.Tabindex("-1"),
						attr.Disabled,
						stimulus.Target("settings-text-field", "mirror"),
					),
				),
			),
		)...)
	}
}
