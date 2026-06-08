package view

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func AppSecurity() (string, domi.Node) {
	return "Security", html.Div()(domi.Text("Change Password"))
}
