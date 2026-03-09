package web

import (
	"database/sql"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
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
		_, selID, _ := xstrings.LastCut(req.PathValue("id"), "-")

		mo, err := tx.Movie(ctx, selID)
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/movies", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		detail := view.EditMediaMoviesDetail(mo)
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
