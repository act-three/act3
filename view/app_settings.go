package view

import (
	. "ily.dev/act3/hi"
	"ily.dev/domi/attr"

	"ily.dev/act3/msg"
	"ily.dev/act3/ui"
)

// settingsPage lays out a system settings page:
// a title above its fields.
func settingsPage(title string, fields ...View) View {
	return ScrollView(Vertical,
		VStack(
			Text(title).Title(title),
			Group(fields...),
		).
			Alignment(Leading).
			Gap(16i).
			Padding(Edges(16i)),
	)
}

// settingsTextField lays out a labeled text input showing value.
// Edits are dispatched through set.
func settingsTextField(label, value string, set func(string) msg.Msg) View {
	return VStack(
		Text(label),
		HTML(ui.InputText(
			ui.Class("v-system-input"),
			attr.Value(value),
			ui.OnChangeValue(set),
		)).Class("v-html-block"),
	).
		Alignment(Leading).
		Gap(0).
		Padding(EdgesLetterbox(16i))
}
