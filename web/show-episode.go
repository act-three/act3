package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) showEpisode(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		slug := req.PathValue("slug") + "/" + req.PathValue("epSlug")
		ep, err := tx.EpisodeBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}

		videos, err := tx.VideoListByEpisodeID(ctx, ep.ID())
		if err != nil {
			return nil, err
		}

		dls, err := tx.RenditionForDownloadList(ctx, ep.ID())
		if err != nil {
			return nil, err
		}

		return view.MediaEpisode(ep, videos, dls), nil
	})
}
