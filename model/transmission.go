package model

import (
	"database/sql"
	urlpkg "net/url"

	"github.com/hekmon/transmissionrpc/v3"
	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
)

type ConfigTransmission struct {
	Path    string
	BaseURL string
}

func (tx *TxR) loadTransmissionConfig(ctx Context) (err error) {
	defer errorfmt.Handlef("transmission: %w", &err)
	ct, err := tx.q.TransmissionGet(ctx)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	u, err := urlpkg.Parse(ct.BaseURL)
	if err != nil {
		return err
	}
	return tx.m.setTransmissionBaseURL(ctx, u)
}

func (tx *TxR) Transmission(ctx Context) (*ConfigTransmission, error) {
	ct, err := tx.q.TransmissionGet(ctx)
	if err == sql.ErrNoRows {
		return &ConfigTransmission{}, nil
	}
	return (*ConfigTransmission)(&ct), err
}

func (tx *TxRW) TransmissionSet(ctx Context, c ConfigTransmission) error {
	u, err := urlpkg.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	err = tx.q.TransmissionSet(ctx, schema.TransmissionSetParams(c))
	if err != nil {
		return err
	}
	return tx.m.setTransmissionBaseURL(ctx, u)
}

func (m *Model) setTransmissionBaseURL(ctx Context, u *urlpkg.URL) error {
	c, err := transmissionrpc.New(u, nil)
	if err != nil {
		return err
	}
	m.transmission.Store(c)
	return nil
}
