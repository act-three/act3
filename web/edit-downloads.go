package web

import (
	"database/sql"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
)

func (c *Config) editDownloads(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		dls, err := tx.DownloadHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaDownloads("Downloads", dls, nil), nil
	})
}

func (c *Config) editDownloadsDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		_, id, _ := xstrings.LastCut(req.PathValue("id"), "-")

		dl, err := tx.Download(ctx, id)
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/downloads", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		if req.Header.Get("turbo-frame") == "detail" {
			return view.EditMediaDownloadsDetailFrame(dl.Title(), dl), nil
		}

		dls, err := tx.DownloadHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaDownloads(dl.Title(), dls, dl), nil
	})
}

func (c *Config) doAddTorrent(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		file, _, err := req.FormFile("torrent")
		if err != nil {
			return nil, err
		}
		dl, err := tx.DownloadCreate(ctx, file)
		if err != nil {
			return nil, err
		}
		dl, err = tx.DownloadCreatePlanSeries(ctx, dl.ID(), req.FormValue("sed-id"))
		if err != nil {
			return nil, err
		}
		dls := []*model.DownloadHead{&dl.DownloadHead}

		return view.EditMediaDownloadsStream(dls, req.FormValue("sed-id")), nil
	})
}
