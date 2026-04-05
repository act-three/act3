package web

import (
	"database/sql"
	"net/http"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
)

func (c *Config) appCollections(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		all, err := tx.CollectionHeadList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppCollections("Collections", all)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) appCollectionsDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		col, err := tx.CollectionBySlug(ctx, req.PathValue("slug"))
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/collections", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		detail := view.AppCollectionDetail(col)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(col.Title(), "detail", detail), nil
		}

		all, err := tx.CollectionHeadList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppCollections(col.Title(), all, detail)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) dialogCollectionMovieAdd(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.AppCollectionMovieAddDialog(req.PathValue("id")), nil
}

func (c *Config) collectionMovieSearch(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		query := req.FormValue("q")
		colID := req.FormValue("col-id")
		if strings.TrimSpace(query) == "" {
			return turbo.Frame("results"), nil
		}
		col, err := tx.Collection(ctx, colID)
		if err != nil {
			return nil, err
		}
		all, err := tx.MovieWorkList(ctx)
		if err != nil {
			return nil, err
		}
		existing := make(map[string]bool, len(col.Movies()))
		for _, mo := range col.Movies() {
			existing[mo.MovieHead.ID()] = true
		}
		query = strings.ToLower(query)
		var matches []view.CollectionMovieSearchResult
		for _, mw := range all {
			if strings.Contains(strings.ToLower(mw.Title()), query) {
				matches = append(matches, view.CollectionMovieSearchResult{
					Movie:        mw,
					InCollection: existing[mw.MovieHead.ID()],
				})
			}
		}
		return view.AppCollectionMovieSearchResults(colID, matches), nil
	})
}

func (c *Config) doCollectionMovieAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		colID := req.FormValue("col-id")
		movieID := req.FormValue("movie-id")
		if colID == "" || movieID == "" {
			return nil, &model.ValidationError{Op: "add movie to collection", Err: errNotFound}
		}
		err := tx.CollectionMovieAdd(ctx, colID, movieID)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doCollectionMovieRemove(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		colID := req.FormValue("col-id")
		movieID := req.FormValue("movie-id")
		if colID == "" || movieID == "" {
			return nil, &model.ValidationError{Op: "remove movie from collection", Err: errNotFound}
		}
		err := tx.CollectionMovieRemove(ctx, colID, movieID)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) dialogCollectionSeriesAdd(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.AppCollectionSeriesAddDialog(req.PathValue("id")), nil
}

func (c *Config) collectionSeriesSearch(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		query := req.FormValue("q")
		colID := req.FormValue("col-id")
		if strings.TrimSpace(query) == "" {
			return turbo.Frame("results"), nil
		}
		col, err := tx.Collection(ctx, colID)
		if err != nil {
			return nil, err
		}
		all, err := tx.SeriesWorkList(ctx)
		if err != nil {
			return nil, err
		}
		existing := make(map[string]bool, len(col.Series()))
		for _, sr := range col.Series() {
			existing[sr.SeriesHead.ID()] = true
		}
		query = strings.ToLower(query)
		var matches []view.CollectionSeriesSearchResult
		for _, sw := range all {
			if strings.Contains(strings.ToLower(sw.Title()), query) {
				matches = append(matches, view.CollectionSeriesSearchResult{
					Series:       sw,
					InCollection: existing[sw.SeriesHead.ID()],
				})
			}
		}
		return view.AppCollectionSeriesSearchResults(colID, matches), nil
	})
}

func (c *Config) doCollectionSeriesAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		colID := req.FormValue("col-id")
		seriesID := req.FormValue("series-id")
		if colID == "" || seriesID == "" {
			return nil, &model.ValidationError{Op: "add series to collection", Err: errNotFound}
		}
		err := tx.CollectionSeriesAdd(ctx, colID, seriesID)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doCollectionSeriesRemove(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		colID := req.FormValue("col-id")
		seriesID := req.FormValue("series-id")
		if colID == "" || seriesID == "" {
			return nil, &model.ValidationError{Op: "remove series from collection", Err: errNotFound}
		}
		err := tx.CollectionSeriesRemove(ctx, colID, seriesID)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) collectionOverview(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.Collection(req.Context(), req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		return turbo.Frame("collection-content")(
			view.TheaterCollectionOverview(col),
		), nil
	})
}

func (c *Config) collectionPlaylist(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id := req.PathValue("id")
		col, err := tx.Collection(ctx, id)
		if err != nil {
			return nil, err
		}
		ps, err := tx.CollectionPlayables(ctx, id)
		if err != nil {
			return nil, err
		}
		return turbo.Frame("collection-content")(
			view.TheaterCollectionPlaylist(col, ps),
		), nil
	})
}

func (c *Config) dialogCollectionBanner(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		col, err := tx.CollectionHead(ctx, req.PathValue("id"))
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		} else if err != nil {
			return nil, err
		}
		return view.AppCollectionBannerDialog(col), nil
	})
}

func (c *Config) doCollectionSetTitle(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		title := req.FormValue("title")
		if id == "" || title == "" {
			return nil, &model.ValidationError{Op: "set collection title", Err: errNotFound}
		}
		err := tx.CollectionTitleSet(ctx, id, title)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doCollectionAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		col, err := tx.CollectionCreate(ctx, "New Collection")
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, col.EditorPath(), http.StatusSeeOther)
		return nil, nil
	})
}
