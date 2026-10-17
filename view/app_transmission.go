package view

import (
	"ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
)

func AppTransmission(tx *model.TxR) hi.View {
	settings := tx.SettingGetByGroup("transmission")
	return settingsPage("Transmission",
		settingsTextField("RPC URL",
			settings[model.SettingKeyTransmissionBaseURL].String(),
			func(v string) msg.Msg { return &msg.TransmissionSetURL{URL: v} },
		),
	)
}
