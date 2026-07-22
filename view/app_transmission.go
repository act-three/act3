package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

func AppTransmission(settings model.Settings) (string, domi.Node) {
	return "Transmission", html.Div(Class("v-system"))(
		html.Div()(domi.Text("Transmission")),
		html.Form(
			onSubmit("url", func(value string) msg.Msg {
				return &msg.TransmissionSetURL{URL: value}
			}),
		)(
			html.Div(Class("v-system-field"))(
				html.Div()(domi.Text("RPC URL")),
				InputText(
					attr.Name("url"),
					Class("v-system-input"),
					attr.Value(settings[model.SettingKeyTransmissionBaseURL].String()),
					OnChangeValue(func(v string) msg.Msg {
						return &msg.TransmissionSetURL{URL: v}
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
