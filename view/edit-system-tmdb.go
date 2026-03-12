package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func EditSystemTMDB(settings model.Settings) html.Node {
	return app("TMDB",
		html.Div(
			attr.Class("h-full w-full p-4"),
		)(
			html.Div()(html.Text("TMDB")),
			html.Form(
				attr.Method("post"),
				attr.Action("/-/do/update-tmdb-settings"),
			)(
				html.Div(attr.Class("py-4"))(
					html.Div()(
						html.Text("API Read Access Token"),
					),
					InputText(
						attr.Name("token"),
						attr.Class("max-w-xs"),
						attr.Value(settings[model.SettingKeyTMDBAccessToken].String()),
					),
				),
				html.Div(attr.Class("py-4"))(
					InputSubmit(
						attr.Value("Save"),
					),
				),
			),
		),
	)
}
