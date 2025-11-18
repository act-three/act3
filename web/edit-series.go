package web

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tvmaze"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/app"
	"ily.dev/act3/web/editseries"
	"ily.dev/act3/web/input"
	"ily.dev/act3/web/turbo"
	"ily.dev/act3/xstrings"
)

func (w *web) editSeries(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		all, err := tx.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return editseries.Editor("Edit Series", all), nil
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

		dls, err := tx.DownloadHeadListBySeriesEditionID(ctx, sed.ID())
		if err != nil {
			return nil, err
		}

		detail := editseries.Detail(sr, sed, dls)
		if req.Header.Get("turbo-frame") == "detail" {
			return app.PageFrame(sr.Title(), "detail", detail), nil
		}

		all, err := tx.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return editseries.Editor(sr.Title(), all, detail), nil
	})
}

func (w *web) seriesAddDialogReq(req *http.Request) (http.Handler, error) {
	return app.Dialog(
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
	), nil
}

func (w *web) dialogEditEpisode(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tr *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		_, id, _ := xstrings.LastCut(req.PathValue("id"), "-")
		ep, err := tr.Episode(ctx, id)
		if err != nil {
			return nil, err
		}

		return app.Dialog(
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

				html.Div()(html.Text("Metadata")),
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
		), nil
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
		return app.PartFrame("results"), nil
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

		return app.PartFrame("results",
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
												turbo.TurboFrame(frameID),
											)(
												html.Input(
													attr.Type("hidden"),
													attr.Name("id"),
													attr.Value(strconv.Itoa(t.TVmaze.ID)),
												),
												Button()(html.Text("Add")).
													With(ButtonBordered),
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
		), nil
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
