package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"
	"ily.dev/domi/html"
)

// SettingsButtonRow renders a row of buttons where one is selected.
// Build the buttons with [SettingsButtonRowItem].
func SettingsButtonRow(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-settings-button-row"),
		group(attrs...),
	)
}

// SettingsButtonRowItem renders a single button within a
// SettingsButtonRow. Clicking delivers commit, which should carry
// the item's value; the selection re-renders from server state.
func SettingsButtonRowItem[Msg any](selected bool, commit Msg, nodes ...domi.Node) domi.Node {
	return Button(
		ButtonSurface,
		ButtonSize2,
		attr.Type("button"),
		domi.Bool("data-selected")(selected),
		event.Click(commit),
	)(nodes...)
}
