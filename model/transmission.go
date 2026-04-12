package model

import (
	urlpkg "net/url"
	"path/filepath"

	"github.com/hekmon/transmissionrpc/v3"
	"ily.dev/act3/sys/fsinfo"
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

// transmissionName returns the path of relPath relative to the torrent's
// download directory.
// For single-file torrents the torrent name is the filename itself.
// For multi-file torrents, files are in a subdirectory named after the torrent.
func transmissionName(t *transmissionrpc.Torrent, relPath string) string {
	if *t.Name == relPath {
		return relPath
	}
	return filepath.Join(*t.Name, relPath)
}

func (m *Model) resolveDownloadDir(remoteDir, name string) (string, error) {
	m.downloadDirMu.Lock()
	defer m.downloadDirMu.Unlock()
	if local, ok := m.downloadDir[remoteDir]; ok {
		return local, nil
	}
	local, err := fsinfo.Probe(remoteDir, name)
	if err != nil {
		return "", err
	}
	m.downloadDir[remoteDir] = local
	return local, nil
}
