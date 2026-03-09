package model

import (
	"database/sql"

	"kr.dev/errorfmt"
)

type ConfigTMDB struct {
	AccessToken string
}

func (tx *TxR) loadTMDBConfig(ctx Context) (err error) {
	defer errorfmt.Handlef("tmdb: %w", &err)
	ct, err := tx.q.TMDBGet(ctx)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	tx.m.tmdb.SetToken(ct)
	return nil
}

func (tx *TxR) TMDB(ctx Context) (*ConfigTMDB, error) {
	token, err := tx.q.TMDBGet(ctx)
	if err == sql.ErrNoRows {
		return &ConfigTMDB{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &ConfigTMDB{AccessToken: token}, nil
}

func (tx *TxRW) TMDBSet(
	ctx Context, c ConfigTMDB,
) error {
	err := tx.q.TMDBSet(ctx, c.AccessToken)
	if err != nil {
		return err
	}
	tx.m.tmdb.SetToken(c.AccessToken)
	return nil
}
