package web

import (
	"database/sql"
	"fmt"
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

func (c *Config) editMovies(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		all, err := tx.MovieHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaMovies("All Movies", all), nil
	})
}

func (c *Config) editMoviesDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		mo, err := tx.MovieBySlug(ctx, req.PathValue("slug"))
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/movies", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		edTitle := req.FormValue("edition")
		if edTitle == "" {
			edTitle = model.DefaultEdition
		}
		med := mo.EditionByTitle(edTitle)
		if med == nil {
			return nil, fmt.Errorf("unknown edition: %s", edTitle)
		}

		dls, err := tx.DownloadHeadListByMovieEditionID(ctx, med.ID())
		if err != nil {
			return nil, err
		}

		detail := view.EditMediaMoviesDetail(mo, med, dls)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(mo.Title(), "detail", detail), nil
		}

		all, err := tx.MovieHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaMovies(mo.Title(), all, detail), nil
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
			turbo.Prepend(view.EditMediaMoviesListItems,
				ListItems([]*model.MovieHead{mo}, view.EditMediaMoviesListItem),
			),
		), nil
	})
}

func (c *Config) movieAddDialogReq(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	frameID := req.Header.Get("Turbo-Frame")
	return view.EditMovieAddDialog(frameID), nil
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
		return view.EditMovieSearchResults(results), nil
	})
}
