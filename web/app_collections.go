package web

import (
	"database/sql"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
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
