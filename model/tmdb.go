package model

import "kr.dev/errorfmt"

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
