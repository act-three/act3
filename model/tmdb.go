package model

import (
	"context"
	"fmt"
	"net/http"

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
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("bad status %d", resp.StatusCode)
	} else if t := resp.Header.Get("Content-Type"); t != "image/jpeg" {
		return fmt.Errorf("bad content type %s", t)
	}
	body := http.MaxBytesReader(nil, resp.Body, maxImageBytes)
	posterID, err := tx.m.store.Copy(body)
	if err != nil {
		return err
	}
	return tx.m.WithTxRW(func(tx *TxRW) error {
		return tx.MovieEditionPosterKeySet(ctx, medID, posterID)
	})
}
