package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func AppTransmission(settings model.Settings) (string, html.Node) {
	return "Transmission", html.Div(Class("v-system"))(
		html.Div()(html.Text("Transmission")),
		html.Form(
			attr.Method("post"),
			attr.Action("/-/do/transmission-settings-update"),
		)(
			html.Div(Class("v-system-field"))(
				html.Div()(html.Text("RPC URL")),
				InputText(
					attr.Name("url"),
					Class("v-system-input"),
					attr.Value(settings[model.SettingKeyTransmissionBaseURL].String()),
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
