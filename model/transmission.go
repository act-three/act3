package model

import (
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"time"

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
	// Transmission RPCs are quick metadata exchanges, but their
	// callers carry no deadline (task ctxs and the poll loop), so
	// without a client timeout a connected-but-unresponsive daemon
	// would wedge its caller indefinitely. tmdb and tvmaze bound
	// their requests the same way, per call inside the service.
	c, err := transmissionrpc.New(u, &transmissionrpc.Config{
		CustomClient: &http.Client{Timeout: 30 * time.Second},
	})
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
