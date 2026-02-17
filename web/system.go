package web

import (
	"net/http"
	"path"
	"path/filepath"

	"ily.dev/act3/model"
	"ily.dev/act3/sys/fsinfo"
	"ily.dev/act3/view"
)

var excludeFSType = map[string]bool{
	"devtmpfs": true,
	"efivarfs": true,
	"tmpfs":    true,
}

func (w *web) systemTransmission(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		config, err := tx.Transmission(ctx)
		if err != nil {
			return nil, err
		}
		return page(view.EditSystemTransmission(config)), nil
	})
}

func (w *web) doUpdateTransmissionSettings(req *http.Request) (http.Handler, error) {
	return w.withTxRW(func(tx *model.TxRW) (http.Handler, error) {
		ctx := req.Context()
		err := tx.TransmissionSet(ctx, model.ConfigTransmission{
			Path:    req.FormValue("path"),
			BaseURL: req.FormValue("url"),
		})
		if err != nil {
			return nil, err
		}
		return http.RedirectHandler("/system/transmission", http.StatusSeeOther), nil
	})
}

func (w *web) systemStorage(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
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

		return page(view.EditSystemStorage(storage, fs)), nil
	})
}

func (w *web) systemTasks(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		tasks, err := tx.TaskList(ctx)
		if err != nil {
			return nil, err
		}
		return page(view.EditSystemTasks(tasks)), nil
	})
}

func (w *web) doRunTask(req *http.Request) (http.Handler, error) {
	ctx := req.Context()
	err := w.model.RunTaskNow(ctx, req.PathValue("id"))
	if err != nil {
		return nil, err
	}
	return http.RedirectHandler("/system/tasks", http.StatusSeeOther), nil
}

func (w *web) doDeleteTask(req *http.Request) (http.Handler, error) {
	return w.withTxRW(func(tx *model.TxRW) (http.Handler, error) {
		ctx := req.Context()
		err := tx.TaskDelete(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		return http.RedirectHandler("/system/tasks", http.StatusSeeOther), nil
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
