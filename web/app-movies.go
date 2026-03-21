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
		all, err := tx.MovieHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.AppMovies("All Movies", all), nil
	})
}

func (c *Config) appMoviesDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		mo, err := tx.MovieBySlug(ctx, req.PathValue("slug"))
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/movies", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		var med *model.MovieEdition
		if edSlug := req.PathValue("edslug"); edSlug != "" {
			med = mo.EditionBySlug(edSlug)
		} else {
			med = mo.DefaultEdition()
		}
		if med == nil {
			http.Redirect(w, req, mo.EditURL(), http.StatusSeeOther)
			return nil, nil
		}

		dls, err := tx.DownloadHeadListByMovieEditionID(ctx, med.ID())
		if err != nil {
			return nil, err
		}

		detail := view.AppMoviesDetail(mo, med, dls)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(mo.Title(), "detail", detail), nil
		}

		all, err := tx.MovieHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.AppMovies(mo.Title(), all, detail), nil
	})
}

func (c *Config) doAddMovieEdition(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		movieID := req.FormValue("movie-id")
		if movieID == "" {
			return nil, &model.ValidationError{Op: "add movie edition", Err: errNotFound}
		}
		_, err := tx.MovieEditionCreate(ctx, "New Edition", movieID)
		if err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doAddMovie(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		title := req.FormValue("title")
		if title == "" {
			return nil, &model.ValidationError{
				Op:  "add movie",
				Err: errNotFound,
			}
		}
		mo, err := tx.MovieCreate(ctx, title, 0)
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, mo.EditURL(), http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) doAddMovieTMDB(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
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
			view.MovieResultLink(mo),
			turbo.Prepend(view.AppMoviesListItems,
				ListItems([]*model.MovieHead{mo}, view.AppMoviesListItem),
			),
		), nil
	})
}

func (c *Config) movieAddDialogReq(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	frameID := req.Header.Get("Turbo-Frame")
	return view.AppMovieAddDialog(frameID), nil
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
