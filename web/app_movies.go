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

func (c *Config) appMovies(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		all, err := tx.MovieWorkList(ctx)
		if err != nil {
			return nil, err
		}
		return view.AppMovies("All Movies", all), nil
	})
}

func (c *Config) appMoviesDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		med, err := tx.MovieEditionBySlug(ctx, req.PathValue("slug"), req.PathValue("edslug"))
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/movies", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		editions, err := tx.MovieEditionList(ctx, med.MovieHead())
		if err != nil {
			return nil, err
		}

		dls, err := tx.DownloadHeadListByMovieEditionID(ctx, med.ID())
		if err != nil {
			return nil, err
		}

		detail := view.AppMoviesDetail(med, editions, dls)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(med.Title(), "detail", detail), nil
		}

		all, err := tx.MovieWorkList(ctx)
		if err != nil {
			return nil, err
		}
		return view.AppMovies(med.Title(), all, detail), nil
	})
}

func (c *Config) doMovieEditionSetTitle(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		title := req.FormValue("title")
		if id == "" || title == "" {
			return nil, &model.ValidationError{Op: "set movie edition title", Err: errNotFound}
		}
		err := tx.MovieEditionTitleSet(ctx, id, title)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doMovieEditionSetLabel(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		label := req.FormValue("label")
		if id == "" || label == "" {
			return nil, &model.ValidationError{Op: "set movie edition label", Err: errNotFound}
		}
		err := tx.MovieEditionLabelSet(ctx, id, label)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doMovieEditionSetSummary(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		if id == "" {
			return nil, &model.ValidationError{Op: "set movie edition summary", Err: errNotFound}
		}
		summary := strings.TrimSpace(req.FormValue("summary"))
		err := tx.MovieEditionSummarySet(ctx, id, summary)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doMovieEditionSetYear(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		if id == "" {
			return nil, &model.ValidationError{Op: "set movie edition year", Err: errNotFound}
		}
		year := strings.TrimSpace(req.FormValue("year"))
		err := tx.MovieEditionYearSet(ctx, id, year)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doMovieEditionSetRuntime(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		if id == "" {
			return nil, &model.ValidationError{Op: "set movie edition runtime", Err: errNotFound}
		}
		var runtime int64
		if s := strings.TrimSpace(req.FormValue("runtime")); s != "" {
			var err error
			runtime, err = strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, &model.ValidationError{Op: "set movie edition runtime", Err: err}
			}
		}
		err := tx.MovieEditionRuntimeSet(ctx, id, runtime)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doMovieEditionSetDefault(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		editionID := req.FormValue("edition-id")
		if editionID == "" {
			return nil, &model.ValidationError{Op: "set default movie edition", Err: errNotFound}
		}
		err := tx.MovieEditionSetDefault(ctx, editionID)
		if err != nil {
			return nil, err
		}
		mo, err := tx.MovieHeadByEditionID(ctx, editionID)
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, mo.EditorPath(), http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) doMovieEditionAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		editionID := req.FormValue("edition-id")
		if editionID == "" {
			return nil, &model.ValidationError{Op: "add movie edition", Err: errNotFound}
		}
		mw, err := tx.MovieEditionClone(ctx, editionID)
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, mw.EditorPath(), http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) doMovieAdd(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		title := req.FormValue("title")
		if title == "" {
			return nil, &model.ValidationError{
				Op:  "add movie",
				Err: errNotFound,
			}
		}
		mo, err := tx.MovieCreate(ctx, title, "")
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, mo.EditorPath(), http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) doMovieAddByTMDB(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	id, err := strconv.Atoi(req.FormValue("id"))
	if err != nil {
		return nil, &model.ValidationError{Op: "TMDB ID", Err: err}
	}
	movie, err := c.TMDB.GetMovie(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		mo, err := tx.MovieCreateByTMDBID(ctx, movie)
		if err != nil {
			return nil, err
		}
		return turbo.Frame("tmdb-"+strconv.FormatInt(*mo.TMDBID(), 10))(
			view.MovieResultLink(mo.EditorPath()),
			turbo.Prepend(view.AppMoviesListItems,
				ListItems([]*model.MovieWork{mo}, view.AppMoviesListItem),
			),
		), nil
	})
}

func (c *Config) movieAddDialogReq(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.AppMovieAddDialog(), nil
}

func (c *Config) dialogMoviePoster(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		med, err := tx.MovieEdition(ctx, req.PathValue("id"))
		if err == sql.ErrNoRows {
			// TODO: show notification sayng "not found".
			// Maybe by hooking into turbo events!
			return nil, sql.ErrNoRows
		} else if err != nil {
			return nil, err
		}
		return view.AppMoviePosterDialog(med), nil
	})
}

func (c *Config) movieSearch(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	query := req.FormValue("q")
	slog.InfoContext(ctx, "movie search", "q", query)
	if strings.TrimSpace(query) == "" {
		return turbo.Frame("results"), nil
	}
	movies, err := c.TMDB.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "movie results",
		"len", len(movies))
	ids := make([]*int64, len(movies))
	for i := range movies {
		id := int64(movies[i].ID)
		ids[i] = &id
	}

	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		existing, err := tx.MovieHeadListByTMDBID(ctx, ids)
		if err != nil {
			return nil, err
		}
		m := make(map[int64]*model.MovieHead)
		for _, mo := range existing {
			m[*mo.TMDBID()] = mo
		}
		results := make([]view.MovieSearchResult, len(movies))
		for i, res := range movies {
			results[i] = view.MovieSearchResult{
				TMDB:  res,
				Local: m[int64(res.ID)],
			}
		}
		return view.AppMovieSearchResults(results), nil
	})
}
