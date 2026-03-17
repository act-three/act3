package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func EditSystemTransmission(settings model.Settings) html.Node {
	return app("Transmission",
		html.Div(attr.Class("v-system"))(
			html.Div()(html.Text("Transmission")),
			html.Form(
				attr.Method("post"),
				attr.Action("/-/do/update-transmission-settings"),
			)(
				html.Div(attr.Class("v-system-field"))(
					html.Div()(html.Text("RPC URL")),
					InputText(
						attr.Name("url"),
						attr.Class("v-system-input"),
						attr.Value(settings[model.SettingKeyTransmissionBaseURL].String()),
					),
				),
				html.Div(attr.Class("v-system-field"))(
					html.Div()(html.Text("Download Folder")),
					InputText(
						attr.Name("path"),
						attr.Class("v-system-input"),
						attr.Value(settings[model.SettingKeyTransmissionPath].String()),
					),
				),
				html.Div(attr.Class("v-system-field"))(
					InputSubmit(
						attr.Value("Save"),
					),
				),
			),
		),
	)
}
