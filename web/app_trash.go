package web

import (
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
		if err := tx.Trash(ctx, id); err != nil {
			return nil, err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil, nil
	})
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
