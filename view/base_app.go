package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view/sidebar"
)

type AppConfig struct {
	TaskCount      int
	TaskCountError int
}

func App(title string, body html.Node, cfg AppConfig) html.Node {
	return base(title)()(
		turbo.StreamSource("/-/events"),
		html.Div(
			attr.Attr("data-slot")("sidebar-wrapper"),
			attr.Class("v-app"),
			attr.Style("--sidebar-width: 200px; --sidebar-width-mobile: 20rem;"),
		)(
			sidebar.Sidebar(sidebar.Config{
				TaskCount:      cfg.TaskCount,
				TaskCountError: cfg.TaskCountError,
			}),
			html.Div(
				attr.Role("main"),
				attr.Attr("data-slot")("sidebar-inset"),
				attr.Class("v-app-main"),
			)(
				turbo.Frame("main",
					turbo.Advance(),
				)(
					body,
				),
			),
		),
		Port(),
		NotePort(),
	)
}

func PageFrame(title, id string, child ...html.Node) html.Node {
	return Group(
		html.Title()(html.Text(title)),
		turbo.Frame(id)(child...),
	)
}
