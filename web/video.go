package web

import (
	"mime"
	"net/http"
	"strings"

	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (w *web) showPlayerForEpisode(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tr *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		v, err := tr.Video(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		ep, err := tr.EpisodeInEdition(ctx,
			req.PathValue("epID"),
			req.PathValue("sedID"),
		)
		if err != nil {
			return nil, err
		}
		qualityOpts, err := tr.QualityOptions(ctx, v)
		if err != nil {
			return nil, err
		}
		return page(view.MediaPlayerForEpisode(v, ep, qualityOpts)), nil
	})
}

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
