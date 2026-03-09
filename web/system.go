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

func (c *Config) systemTMDB(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		config, err := tx.TMDB(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditSystemTMDB(config), nil
	})
}

func (c *Config) doUpdateTMDBSettings(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		err := tx.TMDBSet(ctx, model.ConfigTMDB{
			AccessToken: req.FormValue("token"),
		})
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, "/app/tmdb", http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) systemTransmission(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		config, err := tx.Transmission(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditSystemTransmission(config), nil
	})
}

func (c *Config) doUpdateTransmissionSettings(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		err := tx.TransmissionSet(ctx, model.ConfigTransmission{
			Path:    req.FormValue("path"),
			BaseURL: req.FormValue("url"),
		})
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, "/app/transmission", http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) systemStorage(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
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

		return view.EditSystemStorage(storage, fs), nil
	})
}

func (c *Config) systemTasks(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	running := c.Model.RunningTasks()
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		queued, err := tx.TaskList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditSystemTasks(running, queued), nil
	})
}

func (c *Config) doRunTask(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	err := c.Model.RunTaskNow(ctx, req.PathValue("id"))
	if err != nil {
		return nil, err
	}
	http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
	return nil, nil
}

func (c *Config) doKillTask(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	c.Model.KillTask(req.PathValue("id"))
	http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
	return nil, nil
}

func (c *Config) doDeleteTask(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		err := tx.TaskDelete(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
		return nil, nil
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
