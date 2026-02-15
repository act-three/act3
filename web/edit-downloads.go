package web

import (
	"database/sql"
	"net/http"

	"ily.dev/act3/model"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
)

func (w *web) editDownloads(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		dls, err := tx.DownloadHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return page(view.EditMediaDownloads("Downloads", dls, nil)), nil
	})
}

func (w *web) editDownloadsDetail(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		_, id, _ := xstrings.LastCut(req.PathValue("id"), "-")

		dl, err := tx.Download(ctx, id)
		if err == sql.ErrNoRows {
			return http.RedirectHandler("/edit/downloads", http.StatusSeeOther), nil
		} else if err != nil {
			return nil, err
		}

		if req.Header.Get("turbo-frame") == "detail" {
			return page(view.EditMediaDownloadsDetailFrame(dl.Title(), dl)), nil
		}

		dls, err := tx.DownloadHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return page(view.EditMediaDownloads(dl.Title(), dls, dl)), nil
	})
}

func (w *web) doAddTorrent(req *http.Request) (h http.Handler, err error) {
	defer decorateErrorFrame("add-torrent-errors", &err)
	return w.withTxRW(func(tx *model.TxRW) (http.Handler, error) {
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

		return stream(view.EditMediaDownloadsStream(dls, req.FormValue("sed-id"))), nil
	})
}
