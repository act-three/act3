package web

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tvmaze"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
	"ily.dev/act3/web/input"
	"ily.dev/act3/xstrings"
)

func (w *web) editSeries(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		all, err := tx.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return page(view.EditMediaSeries("Edit Series", all)), nil
	})
}

func (w *web) editSeriesDetail(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		_, selID, _ := xstrings.LastCut(req.PathValue("id"), "-")

		sr, err := tx.Series(ctx, selID)
		if err == sql.ErrNoRows {
			return http.RedirectHandler("/edit/series", http.StatusSeeOther), nil
		} else if err != nil {
			return nil, err
		}

		orderBy := req.FormValue("orderBy")
		if orderBy == "" {
			orderBy = model.AirDate
		}
		sed := sr.EditionByTitle(orderBy)
		if sed == nil {
			return nil, fmt.Errorf("unknown edition name: %s", orderBy)
		}

		dls, err := tx.DownloadHeadListBySeriesEditionID(ctx, sed.ID())
		if err != nil {
			return nil, err
		}

		detail := view.EditMediaSeriesDetail(sr, sed, dls)
		if req.Header.Get("turbo-frame") == "detail" {
			return page(view.PageFrame(sr.Title(), "detail", detail)), nil
		}

		all, err := tx.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return page(view.EditMediaSeries(sr.Title(), all, detail)), nil
	})
}

func (w *web) seriesAddDialogReq(req *http.Request) (http.Handler, error) {
	return page(view.Dialog(
		html.Div(
			attr.Attr("data-controller")("add-series"),
			attr.Class(`
				w-2xl
				h-full
				flex
				flex-col
				gap-2
			`),
		)(
			html.Div(
				attr.Class("flex-none"),
			)(
				html.Text("Add Series"),
			),
			html.Form(
				attr.Action("/search-series"),
				attr.Attr("data-turbo-frame")("results"),
			)(
				input.Text(
					attr.Attr("autofocus"),
					attr.Attr("data-action")("add-series#search"),
					attr.Class("flex-none"),
					attr.Name("q"),
				),
			),
			html.Div(
				attr.Class(`
					flex-initial
					overflow-auto
					overscroll-contain
					h-dvh
					max-h-full
					border
					rounded-sm
				`),
			)(
				turbo.Frame("results"),
			),
		),
	)), nil
}

func (w *web) dialogEditEpisode(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tr *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		_, id, _ := xstrings.LastCut(req.PathValue("id"), "-")
		ep, err := tr.Episode(ctx, id)
		if err != nil {
			return nil, err
		}

		videos, err := tr.VideoListByEpisodeID(ctx, id)
		if err != nil {
			return nil, err
		}

		renditions, err := tr.RenditionForStreamingListByEpisodeID(ctx, id)
		if err != nil {
			return nil, err
		}

		return page(view.Dialog(
			ScrollArea()(
				html.Div()(
					html.Text(ep.SeriesHead().Title()),
				),
				html.Div()(
					html.Text(ep.SeasonHead().Name()),
				),
				html.Div()(
					html.Text(ep.Label()),
				),

				html.Div(
					attr.Class("mt-4 font-bold"),
				)(html.Text("Videos")),
				expr.IfElse(len(videos) == 0,
					func() html.Node {
						return html.Div(
							attr.Class("text-gray-500"),
						)(html.Text("No videos found"))
					},
					func() html.Node { return html.Group() },
				),
				html.Range(videos, func(v schema.Video) html.Node {
					return html.Div(
						attr.Class("ml-4 mt-2"),
					)(
						html.Div()(
							html.Text("ID: "),
							html.Text(v.ID),
						),
						html.Div()(
							html.Text("Release Path: "),
							html.Text(v.ReleasePath),
						),
						html.Div()(
							html.Text("Original Hash: "),
							html.Text(v.OriginalHash),
						),
						expr.IfElse(v.MVPlaylist != "",
							func() html.Node {
								return html.Div()(
									html.Text("Playlist: "),
									html.Text(v.MVPlaylist),
								)
							},
							func() html.Node { return html.Group() },
						),
					)
				}),

				html.Div(
					attr.Class("mt-4 font-bold"),
				)(html.Text("Renditions for Streaming")),
				expr.IfElse(len(renditions) == 0,
					func() html.Node {
						return html.Div(
							attr.Class("text-gray-500"),
						)(html.Text("No renditions found"))
					},
					func() html.Node { return html.Group() },
				),
				html.Range(renditions, func(r schema.RenditionForStreaming) html.Node {
					return html.Div(
						attr.Class("ml-4 mt-2"),
					)(
						html.Div()(
							html.Text("ID: "),
							html.Text(r.ID),
						),
						html.Div()(
							html.Text("Video ID: "),
							html.Text(r.VideoID),
						),
						html.Div()(
							html.Text("Codec: "),
							html.Text(r.Codec),
						),
						html.Div()(
							html.Textf("Target Bitrate: %d kbit/s", r.TargetBitrate),
						),
						html.Div()(
							html.Textf("Remux: %v", r.Remux != 0),
						),
						html.Div()(
							html.Textf("Copy Audio: %v", r.CopyAudio != 0),
						),
						expr.IfElse(r.MaxHeight != 0,
							func() html.Node {
								return html.Div()(
									html.Textf("Max Height: %d", r.MaxHeight),
								)
							},
							func() html.Node { return html.Group() },
						),
						expr.IfElse(r.MaxFPS != 0,
							func() html.Node {
								return html.Div()(
									html.Textf("Max FPS: %d", r.MaxFPS),
								)
							},
							func() html.Node { return html.Group() },
						),
						expr.IfElse(r.Hash != "",
							func() html.Node {
								return html.Div()(
									html.Text("Hash: "),
									html.Text(r.Hash),
								)
							},
							func() html.Node { return html.Group() },
						),
					)
				}),

				html.Div(
					attr.Class("mt-4 font-bold"),
				)(html.Text("Metadata")),
				html.Div()(html.Text("Title")),
				html.Div()(html.Text("Sort Title")),
				html.Div()(html.Text("Season Number")),
				html.Div()(html.Text("Episode Number")),
				html.Div()(html.Text("Overview (plot summary)")),
				html.Div()(html.Text("Release Date")),
				html.Div()(html.Text("Special Episode Info")),
				html.Div()(html.Text("Path")),
				html.Div()(html.Text("Filesize")),
				html.Div()(html.Text("Video Details (codec, framerate, etc)")),
				html.Div()(html.Text("Audio Details (codec, etc)")),
				html.Div()(html.Text("Subtitle Details (format, etc)")),
			),
		)), nil
	})
}

func (w *web) seriesSearch(req *http.Request) (http.Handler, error) {
	type result struct {
		TVmaze tvmaze.Show
		Local  *model.SeriesHead
	}
	ctx := req.Context()
	query := req.FormValue("q")
	slog.InfoContext(ctx, "search", "q", query)
	if strings.TrimSpace(query) == "" {
		return page(turbo.Frame("results")), nil
	}
	series, err := w.tvmaze.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "results", "len", len(series))
	results := make([]result, len(series))
	ids := make([]*int64, len(series))
	for i := range series {
		id := int64(series[i].Show.ID)
		ids[i] = &id
	}
	m := make(map[int64]*model.SeriesHead)

	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		summaries, err := tx.SeriesHeadListByTVmazeID(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, s := range summaries {
			m[*s.TVmazeID()] = s
		}
		for i, res := range series {
			results[i] = result{TVmaze: res.Show, Local: m[int64(res.Show.ID)]}
		}

		return page(turbo.Frame("results")(
			FlexCol(Gap4, Class("p-4"))(
				html.Range(results, func(t result) html.Node {
					frameID := "tvmaze-" + strconv.Itoa(t.TVmaze.ID)
					return Card(Size3, Class("h-[200px]"))(
						FlexRow(Gap4, Class("h-full"))(
							Inset(SideLeft, Class("flex-none"))(
								html.Img(
									Class("block h-full aspect-2/3 object-cover"),
									attr.Src(t.TVmaze.Image.Medium()),
								),
							),
							FlexCol(Gap2)(
								html.Text(t.TVmaze.Name),
								expr.IfElse(t.Local == nil,
									func() html.Node {
										return turbo.Frame(frameID)(
											html.Form(
												attr.Method("post"),
												attr.Action("/do/add-series"),
												turbo.DataFrame(frameID),
											)(
												html.Input(
													attr.Type("hidden"),
													attr.Name("id"),
													attr.Value(strconv.Itoa(t.TVmaze.ID)),
												),
												Button()(html.Text("Add")).
													With(ButtonSurface),
											),
										)
									},
									func() html.Node {
										return seriesResultLink(t.Local)
									},
								),
								Text(t.TVmaze.Summary).
									With(LineClamp3),
							),
						),
					)
				}),
			),
		)), nil
	})
}

func seriesResultLink(ss *model.SeriesHead) html.Node {
	return FlexRow(Gap2)(
		Label("circle-check", "In Library"),
		Button(
			Href(ss.EditURL()),
			Attr("data-turbo-frame")("detail"),
			Attr("data-action")("click->dialog#dismiss"),
		)(
			Text("Edit"),
		),
	)
}
