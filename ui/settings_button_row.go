package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// SettingsButtonRow renders a row of buttons where one is selected.
// It POSTs to action with a form field named name
// carrying the value of the clicked button.
// The selected button is determined by matching value
// against each button's data-settings-button-row-value-param.
// Children become hidden inputs inside the form (for context like IDs).
//
// When addr is non-empty, the component receives live:update events
// and updates the selection accordingly.
func SettingsButtonRow(action, name, value string, attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Form(
			attr.Class("u-settings-button-row"),
			stimulus.Controller("settings-button-row"),
			stimulus.Value("settings-button-row", "url")(action),
			stimulus.Value("settings-button-row", "name")(name),
			stimulus.Value("settings-button-row", "selected")(value),
			attr.Group(attrs...),
		)(nodes...)
	}
}

// SettingsButtonRowItem renders a single button within a SettingsButtonRow.
func SettingsButtonRowItem(value string, nodes ...html.Node) html.Node {
	return Button(
		ButtonSurface,
		ButtonSize2,
		attr.Type("button"),
		stimulus.Target("settings-button-row", "button"),
		stimulus.Action("click->settings-button-row#select"),
		attr.Attr("data-value")(value),
	)(nodes...)
}
