package web

import (
	"mime"
	"net/http"
	"strings"

	"ily.dev/act3/model"
)

// subtitleExts is the set of URL suffixes the subtitleFile handler
// recognises.
var subtitleExts = []string{".vtt", ".ass", ".srt"}

func (c *Config) audioFile(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		id, found := strings.CutSuffix(req.PathValue("id"), ".mp4")
		if !found {
			return nil, errNotFound
		}
		key := tx.FindAudioRenditionMediaKey(id)
		if key == "" {
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
		http.ServeFileFS(w, req, c.Store, key)
		return nil, nil
	})
}

func (c *Config) audioMediaPlaylist(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		pl := tx.FindAudioRenditionPlaylist(id)
		if pl == "" {
			return nil, errNotFound
		}
		stringHandler("application/vnd.apple.mpegurl", pl).ServeHTTP(w, req)
		return nil, nil
	})
}

func (c *Config) subtitleFile(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
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
		key, contentType := tx.FindSubtitleFile(id, ext)
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

func (c *Config) subtitleMediaPlaylist(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		pl := tx.FindSubtitleMediaPlaylist(id)
		if pl == "" {
			return nil, errNotFound
		}
		stringHandler("application/vnd.apple.mpegurl", pl).ServeHTTP(w, req)
		return nil, nil
	})
}

// videoPlaylist serves the multivariant HLS playlist for a video,
// optionally narrowed via ?q=<videoRendID>&a=<audioRendID> query
// params. Both are optional; an absent or empty param includes the
// full set on that side. The combined-pin form is what the player
// JS sends in Chrome (where source-swapping is the only way to
// change audio); Safari uses ?q= alone and switches audio via the
// native audioTracks API.
func (c *Config) videoPlaylist(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		filter := model.MVFilter{
			VideoRenditionID: req.URL.Query().Get("q"),
			AudioRenditionID: req.URL.Query().Get("a"),
		}
		pl := tx.FindMVPlaylist(id, filter)
		if pl == "" {
			return nil, errNotFound
		}
		stringHandler("application/vnd.apple.mpegurl", pl).ServeHTTP(w, req)
		return nil, nil
	})
}

func (c *Config) videoRenditionPlaylist(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		id, found := strings.CutSuffix(req.PathValue("id"), ".m3u8")
		if !found {
			return nil, errNotFound
		}
		pl := tx.FindRenditionPlaylist(id)
		if pl == "" {
			return nil, errNotFound
		}
		stringHandler("application/vnd.apple.mpegurl", pl).ServeHTTP(w, req)
		return nil, nil
	})
}

func (c *Config) videoStream(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		id, found := strings.CutSuffix(req.PathValue("id"), ".mp4")
		if !found {
			return nil, errNotFound
		}
		key := tx.FindRenditionMediaKey(id)
		if key == "" {
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
		http.ServeFileFS(w, req, c.Store, key)
		return nil, nil
	})
}

func (c *Config) videoDownloadForEpisode(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		dl, ok := tx.FindVideoDownloadForEpisode(
			req.PathValue("id"),
			req.PathValue("epID"),
			req.PathValue("sedID"),
		)
		if !ok {
			return nil, errNotFound
		}
		c.serveDownload(w, req, dl)
		return nil, nil
	})
}

func (c *Config) videoDownloadForMovie(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		dl, ok := tx.FindVideoDownloadForMovieEdition(
			req.PathValue("id"),
			req.PathValue("medID"),
		)
		if !ok {
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
