package model

import (
	"bytes"
	"errors"
	"testing"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

func TestCheckTorrentPaths(t *testing.T) {
	tests := []struct {
		name    string
		info    metainfo.Info
		wantErr bool
	}{
		{
			name: "single-file benign",
			info: metainfo.Info{Name: "movie.mkv", Length: 1},
		},
		{
			name:    "single-file dotdot prefix",
			info:    metainfo.Info{Name: "../../etc/passwd.mkv", Length: 1},
			wantErr: true,
		},
		{
			name:    "single-file absolute",
			info:    metainfo.Info{Name: "/etc/passwd.mkv", Length: 1},
			wantErr: true,
		},
		{
			name:    "single-file empty",
			info:    metainfo.Info{Name: "", Length: 1},
			wantErr: true,
		},
		{
			// "." is accepted by filepath.IsLocal (it's the
			// current dir). Not an escape; a downstream os.Open
			// of a directory would fail cleanly at read time.
			name: "single-file dot",
			info: metainfo.Info{Name: ".", Length: 1},
		},
		{
			name:    "single-file dotdot",
			info:    metainfo.Info{Name: "..", Length: 1},
			wantErr: true,
		},
		{
			name: "multi-file benign",
			info: metainfo.Info{
				Name: "MyShow",
				Files: []metainfo.FileInfo{
					{Length: 1, Path: []string{"S01", "E01.mkv"}},
					{Length: 1, Path: []string{"S01", "E02.mkv"}},
				},
			},
		},
		{
			name: "multi-file dotdot prefix in one entry",
			info: metainfo.Info{
				Name: "MyShow",
				Files: []metainfo.FileInfo{
					{Length: 1, Path: []string{"S01", "E01.mkv"}},
					{Length: 1, Path: []string{"..", "..", "etc", "shadow.mkv"}},
				},
			},
			wantErr: true,
		},
		{
			name: "multi-file absolute component",
			info: metainfo.Info{
				Name: "MyShow",
				Files: []metainfo.FileInfo{
					{Length: 1, Path: []string{"/etc/passwd.mkv"}},
				},
			},
			wantErr: true,
		},
		{
			name: "multi-file escaping with intermediate dotdots",
			info: metainfo.Info{
				Name: "MyShow",
				Files: []metainfo.FileInfo{
					{Length: 1, Path: []string{"a", "..", "..", "etc", "passwd.mkv"}},
				},
			},
			wantErr: true,
		},
		{
			name: "multi-file empty path",
			info: metainfo.Info{
				Name:  "MyShow",
				Files: []metainfo.FileInfo{{Length: 1, Path: nil}},
			},
			wantErr: true,
		},
		{
			// PathUtf8 diverges maliciously from Path; we must
			// consult Path only, so this should pass (the safe
			// Path wins) and a later read will use Path.
			name: "PathUtf8 malicious but Path safe",
			info: metainfo.Info{
				Name: "MyShow",
				Files: []metainfo.FileInfo{{
					Length:   1,
					Path:     []string{"ok", "movie.mkv"},
					PathUtf8: []string{"..", "..", "evil.mkv"},
				}},
			},
		},
		{
			// Conversely, a malicious Path must be rejected even
			// when PathUtf8 looks benign — Path is what drives
			// filesystem layout.
			name: "Path malicious but PathUtf8 safe",
			info: metainfo.Info{
				Name: "MyShow",
				Files: []metainfo.FileInfo{{
					Length:   1,
					Path:     []string{"..", "..", "evil.mkv"},
					PathUtf8: []string{"ok", "movie.mkv"},
				}},
			},
			wantErr: true,
		},
		{
			name:    "info.Name escapes via dotdots before files",
			info:    metainfo.Info{Name: "../../etc", Files: []metainfo.FileInfo{{Length: 1, Path: []string{"ok.mkv"}}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTorrentPaths(&tt.info)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkTorrentPaths: err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseTorrentRejectsTraversal(t *testing.T) {
	build := func(info metainfo.Info) []byte {
		t.Helper()
		infoBytes, err := bencode.Marshal(info)
		if err != nil {
			t.Fatal(err)
		}
		mi := metainfo.MetaInfo{InfoBytes: infoBytes}
		var buf bytes.Buffer
		if err := mi.Write(&buf); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	maliciousSingle := build(metainfo.Info{
		Name:        "../../../etc/passwd.mkv",
		Length:      1,
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
	})
	if _, _, err := parseTorrent(maliciousSingle); err == nil {
		t.Fatal("parseTorrent accepted single-file traversal")
	}

	maliciousMulti := build(metainfo.Info{
		Name:        "MyShow",
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
		Files: []metainfo.FileInfo{
			{Length: 1, Path: []string{"..", "..", "evil.mkv"}},
		},
	})
	if _, _, err := parseTorrent(maliciousMulti); err == nil {
		t.Fatal("parseTorrent accepted multi-file traversal")
	}

	benign := build(metainfo.Info{
		Name:        "movie.mkv",
		Length:      1,
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
	})
	if _, _, err := parseTorrent(benign); err != nil {
		t.Fatalf("parseTorrent rejected benign torrent: %v", err)
	}
}

func TestDownloadCreateRejectsTraversal(t *testing.T) {
	m := newTestModel(t)

	infoBytes, err := bencode.Marshal(metainfo.Info{
		Name:        "../../../etc/passwd.mkv",
		Length:      1,
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
	})
	if err != nil {
		t.Fatal(err)
	}
	mi := metainfo.MetaInfo{InfoBytes: infoBytes}
	var buf bytes.Buffer
	if err := mi.Write(&buf); err != nil {
		t.Fatal(err)
	}

	err = m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.DownloadCreate(t.Context(), bytes.NewReader(buf.Bytes()), nil, nil)
		return err
	})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("DownloadCreate: err = %v (%T), want *ValidationError", err, err)
	}

	// No Download row should have been persisted.
	if err := m.WithTxR(func(tx *TxR) error {
		dls, err := tx.q.DownloadList(t.Context())
		if err != nil {
			return err
		}
		if len(dls) != 0 {
			t.Fatalf("DownloadList: got %d rows, want 0", len(dls))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
