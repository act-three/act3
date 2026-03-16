package model

import (
	urlpkg "net/url"
	"path/filepath"

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

// transmissionDiskPath returns the local disk path for a file within
// a Transmission torrent.
// It resolves the download directory using the transmission.path
// setting (falling back to the dir reported by Transmission),
// and handles the single-file vs multi-file torrent distinction.
func (tx *TxR) transmissionDiskPath(ctx Context, t *transmissionrpc.Torrent, relPath string) (string, error) {
	dir := *t.DownloadDir
	settings, err := tx.SettingGetByGroup(ctx, "transmission")
	if err != nil {
		return "", err
	}
	if p := settings[SettingKeyTransmissionPath].String(); p != "" {
		dir = p
	}
	// For single-file torrents the torrent name is the filename
	// itself, so the path is just dir/name.
	// For multi-file torrents it's dir/name/relPath.
	if *t.Name == relPath {
		return filepath.Join(dir, relPath), nil
	}
	return filepath.Join(dir, *t.Name, relPath), nil
}
