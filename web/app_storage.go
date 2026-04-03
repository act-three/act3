package web

import (
	"net/http"
	"path"
	"path/filepath"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/sys/fsinfo"
	"ily.dev/act3/view"
)

var excludeFSType = map[string]bool{
	"devtmpfs": true,
	"efivarfs": true,
	"tmpfs":    true,
}

func (c *Config) appStorage(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		storage, err := tx.StorageList(ctx)
		if err != nil {
			return nil, err
		}
		fsList, err := fsinfo.GetInfo()
		if err != nil {
			return nil, err
		}
		var fs []*view.Filesystem
		for _, f := range fsList {
			if excludeFSType[f.Type] {
				continue
			}
			if f.Size == 0 {
				continue
			}
			path := []string{f.Path[0]}
		outer:
			for _, p := range f.Path[1:] {
				for _, s := range path {
					if filepathHasPrefix(p, s) {
						continue outer
					}
				}
				path = append(path, p)
			}
			fs = append(fs, &view.Filesystem{
				Type: f.Type,
				Path: path,
				Size: int64(f.Size),
				Used: int64(f.Size) - int64(f.Avail),
				Free: int64(f.Avail),
			})
		}

		title, body := view.AppStorage(storage, fs)
		return c.app(ctx, tx, title, body)
	})
}

func filepathHasPrefix(p, prefix string) bool {
	prefix = path.Clean(prefix)
	for {
		p = path.Clean(p)
		if p == prefix {
			return true
		}
		if p == "/" {
			return false
		}
		p, _ = filepath.Split(p)
	}
}
