package web

import (
	"database/sql"
	"fmt"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
)

func (c *Config) appDownloads(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		dls, err := tx.DownloadHeadList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppDownloads("Downloads", dls, nil)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) appDownloadsDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
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
			return view.AppDownloadsDetailFrame(dl.Title(), dl), nil
		}

		dls, err := tx.DownloadHeadList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppDownloads(dl.Title(), dls, dl)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) dialogDownloadFileAttach(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		infoHash := req.FormValue("infohash")
		filePath := req.FormValue("path")
		dl, err := tx.Download(ctx, infoHash)
		if err != nil {
			return nil, err
		}
		sed := dl.PlanSeriesEdition()
		if sed == nil {
			return nil, fmt.Errorf("download %s is not planned for a series", infoHash)
		}
		linked := map[string]bool{}
		vid, err := tx.VideoGetByName(ctx, infoHash, filePath)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == nil {
			for _, epID := range dl.EpisodeIDsByVideoID(vid.ID()) {
				linked[epID] = true
			}
		}
		triggerID := req.FormValue("popover-trigger")
		return view.AppDownloadFileAttachPopover(triggerID, sed, infoHash, filePath, linked), nil
	})
}

func (c *Config) doEpisodeVideoSet(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		infoHash := req.FormValue("infohash")
		filePath := req.FormValue("path")
		episodeID := req.FormValue("episode-id")
		attach := req.FormValue("attach") == "true"
		err := tx.EpisodeVideoSet(ctx, infoHash, filePath, episodeID, attach)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doDownloadAutoImport(w http.ResponseWriter, req *http.Request) (html.Node, error) {
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

func (c *Config) doDownloadImport(w http.ResponseWriter, req *http.Request) (html.Node, error) {
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

func (c *Config) doTorrentAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
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
			dl, err = tx.DownloadCreatePlanSeries(ctx, dl.InfoHash(), sedID)
			edID = sedID
		} else if medID := req.FormValue("med-id"); medID != "" {
			dl, err = tx.DownloadCreatePlanMovie(ctx, dl.InfoHash(), medID)
			edID = medID
		}
		if err != nil {
			return nil, err
		}
		dls := []*model.DownloadHead{&dl.DownloadHead}

		return view.AppDownloadsStream(dls, edID), nil
	})
}
