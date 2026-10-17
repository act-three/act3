package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

func AppTransmission(tx *model.TxR) hi.View {
	settings := tx.SettingGetByGroup("transmission")
	return settingsPage("Transmission",
		html.Div(Class("v-system-field"))(
			html.Div()(domi.Text("RPC URL")),
			InputText(
				Class("v-system-input"),
				attr.Value(settings[model.SettingKeyTransmissionBaseURL].String()),
				OnChangeValue(func(v string) msg.Msg {
					return &msg.TransmissionSetURL{URL: v}
				}),
			),
		),
	)
}
