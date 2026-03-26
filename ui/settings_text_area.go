package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

// SettingsTextArea renders an inline-updating textarea.
// It POSTs to action with a form body containing all
// child hidden inputs plus the textarea's value on blur.
// Children become hidden inputs (for context like IDs).
//
// When target is non-empty, the form element gets that CSS class
// and a data-settings-text-area-text-value attribute.
// A custom "set" Turbo Stream action can update that attribute,
// which triggers the Stimulus controller to update the textarea.
// See SettingsTextAreaSetValue.
func SettingsTextArea(action, name, value, target string, attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		targetAttrs := attr.Group()
		if target != "" {
			targetAttrs = attr.Group(
				Class(target),
				stimulus.Value("settings-text-area", "text")(value),
			)
		}
		return html.Form(
			attr.Class("u-settings-text-area"),
			stimulus.Controller("settings-text-area"),
			stimulus.Value("settings-text-area", "url")(action),
			targetAttrs,
			attr.Group(attrs...),
		)(append(nodes,
			html.Textarea(
				attr.Class("u-settings-text-area-input"),
				attr.Name(name),
				stimulus.Target("settings-text-area", "input"),
				stimulus.Action("blur->settings-text-area#save"),
				stimulus.Action("keydown->settings-text-area#keydown"),
			)(html.Text(value)),
		)...)
	}
}

// SettingsTextAreaSetValue emits a custom "set" Turbo Stream
// that updates the text value on matching SettingsTextArea forms.
func SettingsTextAreaSetValue(selector, value string) html.Node {
	return turbo.SetTargets(selector,
		html.Div(attr.Attr("data-settings-text-area-text-value")(value))(),
	)
}
