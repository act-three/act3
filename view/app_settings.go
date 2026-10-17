package view

import (
	"ily.dev/domi"

	. "ily.dev/act3/hi"
)

// settingsPage lays out a system settings page:
// a title above a settings form.
func settingsPage(title string, form domi.Node) View {
	return ScrollView(Vertical,
		VStack(
			Text(title).Title(title),
			HTML(form).Class("v-html-block"),
		).
			Alignment(Leading).
			Gap(16i).
			Padding(Edges(16i)),
	)
}
