package web

import (
	"mime"
	"net/http"
)

func (wb *web) stream(req *http.Request) (http.Handler, error) {
	hash := req.PathValue("hash")
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFileFS(w, req, wb.store, hash)
	}), nil
}

func (wb *web) download(req *http.Request) (http.Handler, error) {
	hash := req.PathValue("hash")
	disposition := mime.FormatMediaType("attachment", map[string]string{
		"filename": req.PathValue("name"),
	})
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Disposition", disposition)
		http.ServeFileFS(w, req, wb.store, hash)
	}), nil
}
