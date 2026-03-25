package view

import "ily.dev/act3/html"

func AppProfile() html.Node {
	return app("Profile",
		html.Div()(html.Text("Change Name")),
	)
}
