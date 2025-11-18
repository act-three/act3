package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
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

		return media(ep.Title(),
			html.Div(
				attr.Class("p-4 font-bold"),
			)(
				html.Text(ep.Title()),
			),
			html.Div(
				attr.Class("p-4"),
			)(
				html.Text("Streams:"),
				html.Range(streams, func(r *model.RenditionForStreaming) html.Node {
					return html.Div()(
						html.Video(
							attr.Controls,
							attr.Class("w-sm"),
						)(
							html.Source(
								attr.Src(r.URL()),
								attr.Type("video/mp4"),
							),
						),
					)
				}),
			),
			html.Div(
				attr.Class("p-4"),
			)(
				html.Text("Downloads:"),
				html.Range(dls, func(r *model.RenditionForDownload) html.Node {
					return html.Div()(
						html.A(
							attr.Href(r.URL()),
							attr.Download(r.Filename()),
						)(
							html.Text(r.Label()),
						),
					)
				}),
			),
		), nil
	})
}
