package view

import "ily.dev/act3/html"

func AppSecurity() (string, html.Node) {
	return "Security", html.Div()(html.Text("Change Password"))
}
