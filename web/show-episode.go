package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
)

func (c *Config) showEpisode(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		_, epID, _ := xstrings.LastCut(req.PathValue("id"), "-")
		ep, err := tx.Episode(ctx, epID)
		if err != nil {
			return nil, err
		}

		videos, err := tx.VideoListByEpisodeID(ctx, epID)
		if err != nil {
			return nil, err
		}

		dls, err := tx.RenditionForDownloadList(ctx, epID)
		if err != nil {
			return nil, err
		}

		return view.MediaEpisode(ep, videos, dls), nil
	})
}
