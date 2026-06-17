package view

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func AppProfile() (string, domi.Node) {
	return "Profile", html.Div()(domi.Text("Change Name"))
}
