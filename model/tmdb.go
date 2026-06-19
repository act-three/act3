package model

import (
	"kr.dev/errorfmt"
)

func (m *Model) registerTMDBSettingHooks() {
	SettingHook(SettingKeyTMDBAccessToken, func(s *Setting) {
		m.tmdb.SetToken(s.String())
	})
}

func (tx *TxR) loadTMDBConfig() (err error) {
	defer errorfmt.Handlef("tmdb: %w", &err)
	settings, err := tx.SettingGetByGroup("tmdb")
	if err != nil {
		return err
	}
	token := settings[SettingKeyTMDBAccessToken].String()
	if token != "" {
		tx.m.tmdb.SetToken(token)
	}
	return nil
}

func (tx *TxR) taskFetchMoviePoster(args []string) error {
	medID := args[0]
	url := args[1]
	posterID, err := tx.m.imageFetch(tx.ctx, url, ImagePoster)
	if err != nil {
		return err
	}
	return tx.m.WithTxRW(tx.ctx, func(tx *TxRW) error {
		return tx.MovieEditionPosterIDSet(medID, posterID)
	})
}
