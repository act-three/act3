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

func (tx *TxR) loadTransmissionConfig() (err error) {
	defer errorfmt.Handlef("transmission: %w", &err)
	settings := tx.SettingGetByGroup("transmission")
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

// torrentRelPath returns the path of relPath relative to the torrent's
// download directory. For single-file torrents torrentName == relPath,
// so the file sits directly in the download dir. Both inputs must have
// already been accepted by checkTorrentPaths.
func torrentRelPath(torrentName, relPath string) string {
	if torrentName == relPath {
		return relPath
	}
	return filepath.Join(torrentName, relPath)
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
