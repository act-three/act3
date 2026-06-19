package model

import (
	"context"

	"kr.dev/errorfmt"
)

func (m *Model) registerTMDBSettingHooks() {
	SettingHook(SettingKeyTMDBAccessToken, func(s *Setting) {
		m.tmdb.SetToken(s.String())
	})
}

func (tx *TxR) loadTMDBConfig(ctx Context) (err error) {
	defer errorfmt.Handlef("tmdb: %w", &err)
	settings, err := tx.SettingGetByGroup(ctx, "tmdb")
	if err != nil {
		return err
	}
	token := settings[SettingKeyTMDBAccessToken].String()
	if token != "" {
		tx.m.tmdb.SetToken(token)
	}
	return nil
}

func (tx *TxR) taskFetchMoviePoster(ctx context.Context, args []string) error {
	medID := args[0]
	url := args[1]
	posterID, err := tx.m.imageFetch(ctx, url, ImagePoster)
	if err != nil {
		return err
	}
	return tx.m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.MovieEditionPosterIDSet(ctx, medID, posterID)
	})
}
