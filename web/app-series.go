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
	"ily.dev/act3/xstrings"
)

func (c *Config) appSeries(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		all, err := tx.SeriesWorkList(ctx)
		if err != nil {
			return nil, err
		}
		return view.AppSeries("Edit Series", all), nil
	})
}

func (c *Config) appSeriesDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		sed, err := tx.SeriesEditionBySlug(ctx, req.PathValue("slug"), req.PathValue("edslug"))
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/series", http.StatusSeeOther)
			return nil, nil
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
		return view.AppSeries(sr.Title(), all, detail), nil
	})
}

func (c *Config) seriesAddDialogReq(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	frameID := req.Header.Get("Turbo-Frame")
	return view.AppSeriesAddDialog(frameID), nil
}

func (c *Config) dialogEditEpisode(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
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

		frameID := req.Header.Get("Turbo-Frame")
		return view.AppEpisodeDialog(frameID, ep, videos, renditions), nil
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

func (c *Config) doSeriesEditionAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		editionID := req.FormValue("edition-id")
		if editionID == "" {
			return nil, &model.ValidationError{Op: "add series edition", Err: errNotFound}
		}
		_, err := tx.SeriesEditionClone(ctx, editionID)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
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
			view.SeriesResultLink(ss.EditorURL()),
			turbo.Prepend(view.AppSeriesListItems,
				ListItems([]*model.SeriesWork{ss}, view.AppSeriesListItem),
			),
		), nil
	})
}
