package web

import (
	"net/http"

	"ily.dev/act3/model"
)

func (c *Config) doTorrentAdd(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxRW(ctx, func(tx *model.TxRW) (node, error) {
		file, _, err := req.FormFile("torrent")
		if err != nil {
			return nil, err
		}

		// Plan for either a series edition or a movie edition,
		// depending on which hidden field is present.
		var sedID, medID *string
		if v := req.FormValue("sed-id"); v != "" {
			sedID = &v
		} else if v := req.FormValue("med-id"); v != "" {
			medID = &v
		}
		dl, err := tx.DownloadCreate(file, sedID, medID)
		if err != nil {
			return nil, err
		}
		if sedID != nil {
			_, err = tx.DownloadCreatePlanSeries(dl.InfoHash(), *sedID)
		} else if medID != nil {
			_, err = tx.DownloadCreatePlanMovie(dl.InfoHash(), *medID)
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
