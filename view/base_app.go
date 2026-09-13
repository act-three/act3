package view

import (
	"ily.dev/domi/attr"

	"ily.dev/act3/hi"
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

func Editor(body hi.View, cfg AppConfig) hi.View {
	return hi.Group(
		hi.HStack(
			hi.HTML(sidebar.Sidebar(sidebar.Config{
				Path:           cfg.Path,
				TaskCount:      cfg.TaskCount,
				TaskCountError: cfg.TaskCountError,
				Uploads:        cfg.Uploads,
			})).
				FixedSize(),
			hi.ZStack(body).
				Attr(attr.Role("main"), Attr("data-slot")("sidebar-inset")).
				Class("v-app-main").
				BorderClipped().
				Padding(hi.EdgesLetterbox(8i), hi.EdgeTrailing(8i)),
		).
			Gap(0).
			Class("v-app").
			Attr(Attr("data-slot")("sidebar-wrapper")),
		hi.HTML(Port()).FixedSize(),
	)
}
