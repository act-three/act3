package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/view/sidebar"
)

type AppConfig struct {
	Path           string
	TaskCount      int
	TaskCountError int
	Uploads        []model.Upload
}

func Editor(body domi.Node, cfg AppConfig) domi.Node {
	return domi.Fragment(
		html.Div(
			Attr("data-slot")("sidebar-wrapper"),
			Class("v-app"),
			Style("--sidebar-width: 200px; --sidebar-width-mobile: 20rem;"),
		)(
			sidebar.Sidebar(sidebar.Config{
				Path:           cfg.Path,
				TaskCount:      cfg.TaskCount,
				TaskCountError: cfg.TaskCountError,
				Uploads:        cfg.Uploads,
			}),
			html.Div(
				attr.Role("main"),
				Attr("data-slot")("sidebar-inset"),
				Class("v-app-main"),
			)(
				body,
			),
		),
		Port(),
	)
}
