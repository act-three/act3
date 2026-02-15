package web

import (
	"net/http"

	"ily.dev/act3/model"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
)

func (w *web) showEpisode(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		_, epID, _ := xstrings.LastCut(req.PathValue("id"), "-")
		ep, err := tx.Episode(ctx, epID)
		if err != nil {
			return nil, err
		}

		streams, err := tx.RenditionForStreamingList(ctx, epID)
		if err != nil {
			return nil, err
		}

		dls, err := tx.RenditionForDownloadList(ctx, epID)
		if err != nil {
			return nil, err
		}

		return page(view.MediaEpisode(ep, streams, dls)), nil
	})
}
