package web

import (
	"context"
	"database/sql"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) appTrash(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		items, err := tx.TrashList(ctx)
		if err != nil {
			return nil, err
		}
		title, body := view.AppTrash("Trash", items)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) appTrashDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id := req.PathValue("id")
		it, err := tx.TrashItem(ctx, id)
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/trash", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		detail := view.AppTrashDetail(it)
		title := trashDetailTitle(it)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(title, "detail", detail), nil
		}

		items, err := tx.TrashList(ctx)
		if err != nil {
			return nil, err
		}
		pageTitle, body := view.AppTrash(title, items, detail)
		return c.app(ctx, tx, pageTitle, body)
	})
}

func trashDetailTitle(it model.TrashItem) string {
	if it.Title != "" {
		return it.Title
	}
	return it.ID
}

func (c *Config) doTrash(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		if id == "" {
			return nil, &model.ValidationError{Op: "trash", Err: errNotFound}
		}
		redirectTo, err := trashRedirectTarget(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if err := tx.Trash(ctx, id); err != nil {
			return nil, err
		}
		if redirectTo != "" {
			http.Redirect(w, req, redirectTo, http.StatusSeeOther)
			return nil, nil
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

// trashRedirectTarget returns the path the acting user should be sent
// to after trashing id, or "" if the current page is still valid. It
// runs before tx.Trash so parent lookups see the live rows. Editions
// redirect to their parent movie/series root, which re-routes to the
// newly-promoted default edition.
func trashRedirectTarget(ctx context.Context, tx *model.TxRW, id string) (string, error) {
	switch model.KindOf(id) {
	case model.TrashKindMovie:
		return "/app/movies", nil
	case model.TrashKindSeries:
		return "/app/series", nil
	case model.TrashKindEpisode:
		eps, err := tx.EpisodeEditions(ctx, id)
		if err != nil {
			return "", err
		}
		if len(eps) == 0 {
			return "/app/series", nil
		}
		return model.SeriesEditionEditorPath(eps[0].SeriesHead(), eps[0].SeriesEditionHead()), nil
	case model.TrashKindMovieEdition:
		mo, err := tx.MovieHeadByEditionID(ctx, id)
		if err != nil {
			return "", err
		}
		return mo.EditorPath(), nil
	case model.TrashKindSeriesEdition:
		sed, err := tx.SeriesEditionHead(ctx, id)
		if err != nil {
			return "", err
		}
		sr, err := tx.SeriesHead(ctx, sed.SeriesID())
		if err != nil {
			return "", err
		}
		return sr.EditorPath(), nil
	}
	return "", nil
}

func (c *Config) doRestore(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		if id == "" {
			return nil, &model.ValidationError{Op: "restore", Err: errNotFound}
		}
		if err := tx.Restore(ctx, id); err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}

func (c *Config) doPurge(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		id := req.FormValue("id")
		if id == "" {
			return nil, &model.ValidationError{Op: "purge", Err: errNotFound}
		}
		if err := tx.Purge(ctx, id); err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
}
