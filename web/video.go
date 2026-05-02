package web

import (
	"mime"
	"net/http"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/video"
	"ily.dev/act3/view"
)

// subtitleExts is the set of URL suffixes the subtitleFile handler
// recognises.
var subtitleExts = []string{".vtt", ".ass", ".srt"}

func (c *Config) audioFile(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".mp4")
		if !found {
			return nil, errNotFound
		}
		ar, err := tx.AudioRendition(ctx, id)
		if err != nil {
			return nil, errNotFound
		}
		if ar.Key == "" {
			return nil, errNotFound
		}
		// Every audio rendition is fMP4; pin the Content-Type so
		// http.ServeFileFS can't fall through to mime sniffing on
		// our extensionless blob keys. A tight CSP overrides the
		// middleware default as defense in depth even if a non-audio
		// blob ever lands here.
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; media-src 'self'; sandbox")
		http.ServeFileFS(w, req, c.Store, ar.Key)
		return nil, nil
	})
}

func (c *Config) audioMediaPlaylist(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		ar, err := tx.AudioRendition(ctx, id)
		if err != nil {
			return nil, errNotFound
		}
		if ar.Playlist == "" {
			return nil, errNotFound
		}
		stringHandler("application/vnd.apple.mpegurl", ar.Playlist).ServeHTTP(w, req)
		return nil, nil
	})
}

func (c *Config) subtitleFile(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		raw := req.PathValue("id")
		var id, ext string
		for _, e := range subtitleExts {
			if rest, ok := strings.CutSuffix(raw, e); ok {
				id = rest
				ext = strings.TrimPrefix(e, ".")
				break
			}
		}
		if id == "" {
			return nil, errNotFound
		}
		st, err := tx.Subtitle(ctx, id)
		if err != nil {
			return nil, errNotFound
		}

		var key, contentType string
		switch {
		case ext == "vtt":
			key = st.WebVTTKey
			contentType = "text/vtt; charset=utf-8"
		case ext == model.SubtitleOriginalExt(st.OriginalCodec):
			key = st.OriginalKey
			contentType = model.SubtitleContentType(st.OriginalCodec)
		default:
			return nil, errNotFound
		}
		if key == "" {
			return nil, errNotFound
		}

		// Pin Content-Type so http.ServeFileFS can't fall through to
		// mime sniffing on extensionless blob keys. A tight CSP
		// overrides the middleware default as defense in depth: even
		// if a non-subtitle blob ever lands here, nothing in the
		// response can load as active content.
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		http.ServeFileFS(w, req, c.Store, key)
		return nil, nil
	})
}

func (c *Config) subtitleMediaPlaylist(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		st, err := tx.Subtitle(ctx, id)
		if err != nil {
			return nil, errNotFound
		}
		if st.WebVTTKey == "" {
			return nil, errNotFound
		}
		vid, err := tx.Video(ctx, st.VideoID)
		if err != nil {
			return nil, errNotFound
		}
		body := video.GenerateSubtitleMediaPlaylist(vid.Duration(), "/-/sub/"+st.ID+".vtt")
		stringHandler("application/vnd.apple.mpegurl", body).ServeHTTP(w, req)
		return nil, nil
	})
}

func (c *Config) playerForEpisode(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
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
		captionsOpts, err := tr.SubtitleOptions(ctx, v)
		if err != nil {
			return nil, err
		}
		audioOpts, err := tr.AudioOptions(ctx, v)
		if err != nil {
			return nil, err
		}
		return view.PlayerForEpisode(v, ep, qualityOpts, captionsOpts, audioOpts), nil
	})
}

func (c *Config) playerForMovie(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
		ctx := req.Context()
		v, err := tr.Video(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		med, err := tr.MovieEditionHead(ctx, req.PathValue("medID"))
		if err != nil {
			return nil, err
		}
		qualityOpts, err := tr.QualityOptions(ctx, v)
		if err != nil {
			return nil, err
		}
		captionsOpts, err := tr.SubtitleOptions(ctx, v)
		if err != nil {
			return nil, err
		}
		audioOpts, err := tr.AudioOptions(ctx, v)
		if err != nil {
			return nil, err
		}
		return view.PlayerForMovie(v, med, qualityOpts, captionsOpts, audioOpts), nil
	})
}

func (c *Config) videoPlaylist(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		pl, err := tx.MVPlaylist(ctx, id)
		if err != nil || pl == "" {
			return nil, errNotFound
		}
		stringHandler("application/vnd.apple.mpegurl", pl).ServeHTTP(w, req)
		return nil, nil
	})
}

// variantPlaylist serves a single-variant MV playlist for one video
// rendition. The MV form (rather than the bare rendition media
// playlist) is required because audio renditions live in the MV
// EXT-X-MEDIA AUDIO group; a bare media playlist for an `-an` video
// rendition would play silent.
func (c *Config) variantPlaylist(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		pl, err := tx.VariantMVPlaylist(ctx, id)
		if err != nil || pl == "" {
			return nil, errNotFound
		}
		stringHandler("application/vnd.apple.mpegurl", pl).ServeHTTP(w, req)
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
		rend, err := tx.Rendition(ctx, id)
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
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		id, found := strings.CutSuffix(req.PathValue("id"), ".mp4")
		if !found {
			return nil, errNotFound
		}
		rend, err := tx.Rendition(ctx, id)
		if err != nil {
			return nil, errNotFound
		}
		if rend.Key == "" {
			return nil, errNotFound
		}
		// Every streaming rendition is fMP4; pin the Content-Type
		// so http.ServeFileFS can't fall through to mime sniffing
		// on our extensionless blob keys. A tight CSP overrides the
		// middleware default as defense in depth even if a non-video
		// blob ever lands here.
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; media-src 'self'; sandbox")
		http.ServeFileFS(w, req, c.Store, rend.Key)
		return nil, nil
	})
}

func (c *Config) videoDownloadForEpisode(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		dl, err := tx.VideoDownloadForEpisode(req.Context(),
			req.PathValue("id"),
			req.PathValue("epID"),
			req.PathValue("sedID"),
		)
		if err != nil {
			return nil, errNotFound
		}
		c.serveDownload(w, req, dl)
		return nil, nil
	})
}

func (c *Config) videoDownloadForMovie(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		dl, err := tx.VideoDownloadForMovieEdition(req.Context(),
			req.PathValue("id"),
			req.PathValue("medID"),
		)
		if err != nil {
			return nil, errNotFound
		}
		c.serveDownload(w, req, dl)
		return nil, nil
	})
}

// serveDownload pins the Content-Type from the DB record and sets a
// server-generated Content-Disposition so the browser never sees an
// attacker-controlled filename or sniffs the response body. A tight
// CSP overrides the middleware default as defense in depth: if the
// attachment handling ever regresses, nothing in the response can
// still load as active content.
func (c *Config) serveDownload(w http.ResponseWriter, req *http.Request, dl model.VideoDownload) {
	w.Header().Set("Content-Type", dl.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment",
		map[string]string{"filename": dl.Filename},
	))
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeFileFS(w, req, c.Store, dl.Key)
}
