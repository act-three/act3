package view

import "ily.dev/act3/html"

func AppSecurity() html.Node {
	return app("Security",
		html.Div()(html.Text("Change Password")),
	)
}
