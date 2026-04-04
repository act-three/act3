package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) appCollections(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		title, body := view.AppCollections("Collections")
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) appCollectionsDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		slug := req.PathValue("slug")

		detail := view.AppCollectionDetail(slug)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(slug, "detail", detail), nil
		}

		title, body := view.AppCollections(slug, detail)
		return c.app(ctx, tx, title, body)
	})
}
