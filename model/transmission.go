package model

import (
	urlpkg "net/url"

	"github.com/hekmon/transmissionrpc/v3"
	"kr.dev/errorfmt"
)

func (m *Model) registerTransmissionSettingHooks() {
	SettingHook(SettingKeyTransmissionBaseURL, func(s *Setting) {
		u, err := urlpkg.Parse(s.String())
		if err != nil {
			return
		}
		m.setTransmissionBaseURL(u)
	})
}

func (tx *TxR) loadTransmissionConfig(ctx Context) (err error) {
	defer errorfmt.Handlef("transmission: %w", &err)
	settings, err := tx.SettingGetByGroup(ctx, "transmission")
	if err != nil {
		return err
	}
	baseURL := settings[SettingKeyTransmissionBaseURL].String()
	if baseURL == "" {
		return nil
	}
	u, err := urlpkg.Parse(baseURL)
	if err != nil {
		return err
	}
	tx.m.setTransmissionBaseURL(u)
	return nil
}

func (m *Model) setTransmissionBaseURL(u *urlpkg.URL) {
	c, err := transmissionrpc.New(u, nil)
	if err != nil {
		return
	}
	m.transmission.Store(c)
}

// transmissionDownloadDir returns the local directory for a torrent's
// download dir.
// If the transmission.path setting is configured,
// it is used instead of the path reported by Transmission,
// which may differ when Transmission runs on a different host.
func (tx *TxR) transmissionDownloadDir(ctx Context, remoteDir string) (string, error) {
	settings, err := tx.SettingGetByGroup(ctx, "transmission")
	if err != nil {
		return "", err
	}
	if p := settings[SettingKeyTransmissionPath].String(); p != "" {
		return p, nil
	}
	return remoteDir, nil
}
