package view

import (
	"ily.dev/domi"

	"ily.dev/act3/hi"
)

// settingsPage lays out a system settings page:
// a title above a settings form.
func settingsPage(title string, form domi.Node) hi.View {
	return hi.ScrollView(hi.Vertical,
		hi.VStack(
			hi.Text(title).Title(title),
			hi.HTML(form).Class("v-html-block"),
		).
			Alignment(hi.Leading).
			Gap(16i).
			Padding(hi.Edges(16i)),
	)
}
