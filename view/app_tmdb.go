package view

import (
	"ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
)

func AppTMDB(tx *model.TxR) hi.View {
	settings := tx.SettingGetByGroup("tmdb")
	return settingsPage("TMDB",
		settingsTextField("API Read Access Token",
			settings[model.SettingKeyTMDBAccessToken].String(),
			func(v string) msg.Msg { return &msg.TMDBSetToken{Token: v} },
		),
	)
}
