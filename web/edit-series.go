package web

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
)

func (c *Config) editSeries(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		all, err := tx.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaSeries("Edit Series", all), nil
	})
}

func (c *Config) editSeriesDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		sr, err := tx.SeriesBySlug(ctx, req.PathValue("slug"))
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/series", http.StatusSeeOther)
			return nil, nil
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
			return view.PageFrame(sr.Title(), "detail", detail), nil
		}

		all, err := tx.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaSeries(sr.Title(), all, detail), nil
	})
}

func (c *Config) seriesAddDialogReq(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	frameID := req.Header.Get("Turbo-Frame")
	return view.EditSeriesAddDialog(frameID), nil
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
		return view.EditEpisodeDialog(frameID, ep, videos, renditions), nil
	})
}

func (c *Config) doReimportVideo(w http.ResponseWriter, req *http.Request) (html.Node, error) {
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

func (c *Config) doReencodeVideo(w http.ResponseWriter, req *http.Request) (html.Node, error) {
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
		return view.EditSeriesSearchResults(results), nil
	})
}
