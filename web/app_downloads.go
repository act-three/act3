package web

import (
	"net/http"

	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
)

func (c *Config) doTorrentAdd(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxRW(ctx, func(tx *model.TxRW) (node, error) {
		file, _, err := req.FormFile("torrent")
		if err != nil {
			return nil, err
		}

		k, err := kind.ParseTorrentTarget(req.FormValue("kind"))
		if err != nil {
			return nil, &model.ValidationError{Op: "parse", Err: err}
		}
		id := req.FormValue("id")
		dl, err := tx.DownloadCreate(file, k, id)
		if err != nil {
			return nil, err
		}
		// Plan for the targeted edition.
		switch k.(type) {
		case kind.SeriesEdition:
			_, err = tx.DownloadCreatePlanSeries(dl.InfoHash(), id)
		case kind.MovieEdition:
			_, err = tx.DownloadCreatePlanMovie(dl.InfoHash(), id)
		}
		if err != nil {
			return nil, err
		}
		// The commit announces the queued Transmission task, and the
		// re-render shows the new download on the edition's page.
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}
