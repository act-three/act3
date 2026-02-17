package web

import (
	"mime"
	"net/http"
	"strings"

	"ily.dev/act3/model"
)

func (w *web) videoPlaylist(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		vid, err := tx.Video(ctx, id)
		if err != nil {
			return nil, err
		}
		if vid.MVPlaylist() == "" {
			return nil, errNotFound
		}
		return stringHandler("application/vnd.apple.mpegurl", vid.MVPlaylist()), nil
	})
}

func (w *web) videoRenditionPlaylist(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		rend, err := tx.RenditionForStreaming(ctx, id)
		if err != nil {
			return nil, err
		}
		if rend.Playlist == "" {
			return nil, errNotFound
		}
		return stringHandler("application/vnd.apple.mpegurl", rend.Playlist), nil
	})
}
func (wb *web) videoStream(req *http.Request) (http.Handler, error) {
	hash, _ := strings.CutSuffix(req.PathValue("hash"), ".mp4")
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFileFS(w, req, wb.store, hash)
	}), nil
}

func (wb *web) videoDownload(req *http.Request) (http.Handler, error) {
	hash := req.PathValue("hash")
	disposition := mime.FormatMediaType("attachment", map[string]string{
		"filename": req.PathValue("name"),
	})
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Disposition", disposition)
		http.ServeFileFS(w, req, wb.store, hash)
	}), nil
}
