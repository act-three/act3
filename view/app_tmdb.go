package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func AppTMDB(settings model.Settings) (string, html.Node) {
	return "TMDB", html.Div(Class("v-system"))(
		html.Div()(html.Text("TMDB")),
		html.Form(
			attr.Method("post"),
			attr.Action("/-/do/tmdb-settings-update"),
		)(
			html.Div(Class("v-system-field"))(
				html.Div()(
					html.Text("API Read Access Token"),
				),
				InputText(
					attr.Name("token"),
					Class("v-system-input"),
					attr.Value(settings[model.SettingKeyTMDBAccessToken].String()),
				),
			),
			html.Div(Class("v-system-field"))(
				InputSubmit(
					attr.Value("Save"),
				),
			),
		),
	)
}
