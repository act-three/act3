package web

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
)

func (c *Config) appSeries(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		all, err := tx.SeriesWorkList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppSeries("Edit Series", all)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) appSeriesDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		slug2 := req.PathValue("slug2")
		sed, err := tx.SeriesEditionBySlug(ctx, req.PathValue("slug"), slug2)
		if err == sql.ErrNoRows {
			// slug2 might be an episode in the default edition.
			req.SetPathValue("edslug", "")
			req.SetPathValue("epslug", slug2)
			return c.appEpisodeDetail(w, req)
		} else if err != nil {
			return nil, err
		}

		editions, err := tx.SeriesEditionList(ctx, sed.SeriesHead())
		if err != nil {
			return nil, err
		}

		dls, err := tx.DownloadHeadListBySeriesEditionID(ctx, sed.ID())
		if err != nil {
			return nil, err
		}

		sr := sed.SeriesHead()
		detail := view.AppSeriesDetail(sed, editions, dls)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(sr.Title(), "detail", detail), nil
		}

		all, err := tx.SeriesWorkList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppSeries(sr.Title(), all, detail)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) appEpisodeDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		ep, err := tx.EpisodeBySlug(ctx,
			req.PathValue("slug"),
			req.PathValue("edslug"),
			req.PathValue("epslug"),
		)
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/series", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		videos, err := tx.VideoListByEpisodeID(ctx, ep.ID())
		if err != nil {
			return nil, err
		}

		renditions, err := tx.RenditionForStreamingListByEpisodeID(ctx, ep.ID())
		if err != nil {
			return nil, err
		}

		episodeEditions, err := tx.EpisodeEditions(ctx, ep.ID())
		if err != nil {
			return nil, err
		}

		detail := view.AppEpisodeDetail(ep, episodeEditions, videos, renditions)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(ep.Title(), "detail", detail), nil
		}

		all, err := tx.SeriesWorkList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppSeries(ep.Title(), all, detail)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) doSeriesSetTitle(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		title := req.FormValue("title")
		if id == "" || title == "" {
			return nil, &model.ValidationError{Op: "set series title", Err: errNotFound}
		}
		err := tx.SeriesTitleSet(ctx, id, title)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doSeasonSetTitle(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		title := req.FormValue("title")
		if id == "" || title == "" {
			return nil, &model.ValidationError{Op: "set season title", Err: errNotFound}
		}
		err := tx.SeasonTitleSet(ctx, id, title)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doSeriesEditionSetLabel(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		label := req.FormValue("label")
		if id == "" || label == "" {
			return nil, &model.ValidationError{Op: "set series edition label", Err: errNotFound}
		}
		err := tx.SeriesEditionLabelSet(ctx, id, label)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doSeriesEditionSetSummary(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		if id == "" {
			return nil, &model.ValidationError{Op: "set series edition summary", Err: errNotFound}
		}
		summary := strings.TrimSpace(req.FormValue("summary"))
		err := tx.SeriesEditionSummarySet(ctx, id, summary)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doEpisodeSetAirdate(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		airdate := req.FormValue("airdate")
		if id == "" {
			return nil, &model.ValidationError{Op: "set episode airdate", Err: errNotFound}
		}
		err := tx.EpisodeAirdateSet(ctx, id, airdate)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doEpisodeSetType(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		typ := req.FormValue("type")
		if id == "" || typ == "" {
			return nil, &model.ValidationError{Op: "set episode type", Err: errNotFound}
		}
		err := tx.EpisodeTypeSet(ctx, id, typ)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doEpisodeSetSummary(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		summary := req.FormValue("summary")
		if id == "" {
			return nil, &model.ValidationError{Op: "set episode summary", Err: errNotFound}
		}
		err := tx.EpisodeSummarySet(ctx, id, summary)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doEpisodeSetTitle(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		title := req.FormValue("title")
		if id == "" || title == "" {
			return nil, &model.ValidationError{Op: "set episode title", Err: errNotFound}
		}
		err := tx.EpisodeTitleSet(ctx, id, title)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) seriesAddDialogReq(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.AppSeriesAddDialog(), nil
}

func (c *Config) dialogSeriesEditionPoster(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		sed, err := tx.SeriesEdition(ctx, req.PathValue("id"))
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		} else if err != nil {
			return nil, err
		}
		return view.AppSeriesEditionPosterDialog(sed), nil
	})
}

func (c *Config) dialogEpisodeThumbnail(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ep, err := tx.EpisodeHead(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		return view.AppEpisodeThumbnailDialog(ep), nil
	})
}

func (c *Config) doVideoReimport(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	_, err := c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		return nil, tx.ReImportVideo(ctx, req.PathValue("id"))
	})
	if err != nil {
		return nil, err
	}
	http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
	return nil, nil
}

func (c *Config) doVideoReencode(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	err := c.Model.ReencodeVideo(ctx, req.PathValue("id"))
	if err != nil {
		return nil, err
	}
	http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
	return nil, nil
}

func (c *Config) seriesSearch(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	query := req.FormValue("q")
	slog.InfoContext(ctx, "search", "q", query)
	if strings.TrimSpace(query) == "" {
		return turbo.Frame("results"), nil
	}
	series, err := c.TVmaze.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "results", "len", len(series))
	ids := make([]*int64, len(series))
	for i := range series {
		id := int64(series[i].Show.ID)
		ids[i] = &id
	}

	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		summaries, err := tx.SeriesHeadListByTVmazeID(ctx, ids)
		if err != nil {
			return nil, err
		}
		m := make(map[int64]*model.SeriesHead)
		for _, s := range summaries {
			m[*s.TVmazeID()] = s
		}
		results := make([]view.SeriesSearchResult, len(series))
		for i, res := range series {
			results[i] = view.SeriesSearchResult{
				TVmaze: res.Show,
				Local:  m[int64(res.Show.ID)],
			}
		}
		return view.AppSeriesSearchResults(results), nil
	})
}

func (c *Config) doSeasonAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		editionID := req.FormValue("edition-id")
		if editionID == "" {
			return nil, &model.ValidationError{Op: "add season", Err: errNotFound}
		}
		if err := tx.SeasonAdd(ctx, editionID); err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doSeriesEpisodeAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		seasonID := req.FormValue("season-id")
		if seasonID == "" {
			return nil, &model.ValidationError{Op: "add episode", Err: errNotFound}
		}
		if err := tx.SeasonEpisodeAdd(ctx, seasonID); err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doEpisodeMove(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		episodeID := req.FormValue("episode-id")
		fromSeasonID := req.FormValue("from-season-id")
		seasonID := req.FormValue("season-id")
		indexStr := req.FormValue("index")
		if episodeID == "" || fromSeasonID == "" || seasonID == "" {
			return nil, &model.ValidationError{Op: "move episode", Err: errNotFound}
		}
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return nil, &model.ValidationError{Op: "move episode", Err: errNotFound}
		}
		if err := tx.EpisodeMove(ctx, episodeID, fromSeasonID, seasonID, index); err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doSeriesEditionAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		editionID := req.FormValue("edition-id")
		if editionID == "" {
			return nil, &model.ValidationError{Op: "add series edition", Err: errNotFound}
		}
		sw, err := tx.SeriesEditionClone(ctx, editionID)
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, sw.EditorPath(), http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) doSeriesAdd(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	id, err := strconv.Atoi(req.FormValue("id"))
	if err != nil {
		return nil, &model.ValidationError{Op: "TVmaze ID", Err: err}
	}
	show, err := c.TVmaze.GetShow(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ss, err := tx.SeriesCreateByTVmazeID(ctx, show)
		if err != nil {
			return nil, err
		}
		return turbo.Frame("tvmaze-"+strconv.FormatInt(*ss.TVmazeID(), 10))(
			view.SeriesResultLink(ss.EditorPath()),
			turbo.Prepend(view.AppSeriesListItems,
				ListItems([]*model.SeriesWork{ss}, view.AppSeriesListItem),
			),
		), nil
	})
}
