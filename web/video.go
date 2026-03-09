package web

import (
	"mime"
	"net/http"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) showPlayerForEpisode(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
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
		return view.MediaPlayerForEpisode(v, ep, qualityOpts), nil
	})
}

func (c *Config) showPlayerForMovie(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
		ctx := req.Context()
		v, err := tr.Video(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		mo, err := tr.Movie(ctx, req.PathValue("moID"))
		if err != nil {
			return nil, err
		}
		qualityOpts, err := tr.QualityOptions(ctx, v)
		if err != nil {
			return nil, err
		}
		return view.MediaPlayerForMovie(v, mo, qualityOpts), nil
	})
}

func (c *Config) videoPlaylist(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
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
		stringHandler("application/vnd.apple.mpegurl", vid.MVPlaylist()).ServeHTTP(w, req)
		return nil, nil
	})
}

func (c *Config) videoRenditionPlaylist(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
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
		stringHandler("application/vnd.apple.mpegurl", rend.Playlist).ServeHTTP(w, req)
		return nil, nil
	})
}

func (c *Config) videoStream(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	hash, _ := strings.CutSuffix(req.PathValue("hash"), ".mp4")
	http.ServeFileFS(w, req, c.Store, hash)
	return nil, nil
}

func (c *Config) videoDownload(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	hash := req.PathValue("hash")
	disposition := mime.FormatMediaType("attachment", map[string]string{
		"filename": req.PathValue("name"),
	})
	w.Header().Set("Content-Disposition", disposition)
	http.ServeFileFS(w, req, c.Store, hash)
	return nil, nil
}
