package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/ui/stimulus"
)

// SettingsTextFieldPrefix sets a non-editable prefix
// (e.g. "/the-matrix/") displayed before the input text.
var SettingsTextFieldPrefix = stimulus.Value("settings-text-field", "prefix")

// SettingsTextFieldSuffix sets a non-editable suffix
// (e.g. " min") displayed after the input text.
var SettingsTextFieldSuffix = stimulus.Value("settings-text-field", "suffix")

// SettingsTextField renders an inline-editing text field showing
// value. When the user commits an edit (blur or Enter after a
// change), commit is called with the new value and the resulting
// message is delivered. Escape reverts the field to value.
func SettingsTextField[Msg any](value string, commit func(value string) Msg, attrs ...domi.Attr) domi.Node {
	return html.Div(
		Class("u-settings-text-field"),
		stimulus.Controller("settings-text-field"),
		group(attrs...),
	)(
		html.Div(Class("u-settings-text-field-inner"))(
			InputText(
				attr.Value(value),
				onChangeValue(commit),
				stimulus.Target("settings-text-field", "input"),
				stimulus.Action("keydown->settings-text-field#keydown"),
				stimulus.Action("input->settings-text-field#sync"),
			),
			html.Div(Class("u-settings-text-field-overlay"))(
				InputText(
					attr.TabIndex("-1"),
					attr.Disabled(true),
					stimulus.Target("settings-text-field", "mirror"),
				),
			),
		),
	)
}
