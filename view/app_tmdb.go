package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

func AppTMDB(settings model.Settings) (string, domi.Node) {
	return "TMDB", html.Div(Class("v-system"))(
		html.Div()(domi.Text("TMDB")),
		html.Form(
			onSubmit("token", func(value string) msg.Msg {
				return &msg.TMDBSetToken{Token: value}
			}),
		)(
			html.Div(Class("v-system-field"))(
				html.Div()(
					domi.Text("API Read Access Token"),
				),
				InputText(
					attr.Name("token"),
					Class("v-system-input"),
					attr.Value(settings[model.SettingKeyTMDBAccessToken].String()),
					OnChangeValue(func(v string) msg.Msg {
						return &msg.TMDBSetToken{Token: v}
					}),
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
