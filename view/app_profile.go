package view

import "ily.dev/act3/html"

func AppProfile() (string, html.Node) {
	return "Profile", html.Div()(html.Text("Change Name"))
}
