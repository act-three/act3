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

func (c *Config) doAutoImportDownload(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		auto := req.FormValue("auto-import") == "true"
		err := tx.DownloadAutoImportSet(ctx, id, auto)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doImportDownload(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		err := tx.DownloadImport(ctx, id)
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, "/app/downloads/"+id, http.StatusSeeOther)
		return nil, nil
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

		// Plan for either a series edition or a movie edition,
		// depending on which hidden field is present.
		var edID string
		if sedID := req.FormValue("sed-id"); sedID != "" {
			dl, err = tx.DownloadCreatePlanSeries(ctx, dl.ID(), sedID)
			edID = sedID
		} else if medID := req.FormValue("med-id"); medID != "" {
			dl, err = tx.DownloadCreatePlanMovie(ctx, dl.ID(), medID)
			edID = medID
		}
		if err != nil {
			return nil, err
		}
		dls := []*model.DownloadHead{&dl.DownloadHead}

		return view.EditMediaDownloadsStream(dls, edID), nil
	})
}
