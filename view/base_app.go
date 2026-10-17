package view

import (
	"ily.dev/domi/attr"

	. "ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/ui"
	"ily.dev/act3/view/sidebar"
)

type AppConfig struct {
	Path           string
	TaskCount      int
	TaskCountError int
	Uploads        []model.Upload
}

func Editor(body View, cfg AppConfig) View {
	return HStack(
		HTML(sidebar.Sidebar(sidebar.Config{
			Path:           cfg.Path,
			TaskCount:      cfg.TaskCount,
			TaskCountError: cfg.TaskCountError,
			Uploads:        cfg.Uploads,
		})).
			FixedSize(),
		ZStack(body).
			Attr(attr.Role("main"), ui.Attr("data-slot")("sidebar-inset")).
			Class("v-app-main").
			BorderClipped().
			Padding(EdgesLetterbox(8i), EdgeTrailing(8i)),
	).
		Gap(0).
		Class("v-app").
		Attr(ui.Attr("data-slot")("sidebar-wrapper")).
		Overlay(Center, HTML(ui.Port()).FixedSize())
}
