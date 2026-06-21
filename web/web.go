package web

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"ily.dev/domi"

	"ily.dev/act3/http/timing"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	"ily.dev/act3/service/tmdb"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/ui/icon"
	"ily.dev/act3/view"
	"ily.dev/act3/web/jassub"
	"ily.dev/act3/web/static"
)

// optionally set by build flags
var tag = "(unset)"

var errNotFound = errors.New("not found")

// Max request body for all requests except video uploads.
const maxReqBody = 10 * 1000 * 1000

type handlerFunc func(http.ResponseWriter, *http.Request) (node, error)

type Config struct {
	Model  *model.Model
	Store  fs.FS
	TMDB   *tmdb.Client
	TVmaze *tvmaze.Client
}

func Handle(mux *http.ServeMux, c *Config) {
	mux.Handle("/", domi.Handler(
		func(ctx context.Context, u *url.URL) (*app, cmd) {
			return newApp(ctx, c, u)
		},
		msg.OnURLRequest,
		msg.OnURLChange,
		domi.Document(view.Document),
		domi.InternalURLPrefix("/-/domi"),
	))

	for req, dest := range redirects {
		h := http.RedirectHandler(dest, http.StatusSeeOther)
		mux.Handle("GET "+path.Clean(req), h)
		mux.Handle("GET "+path.Join(req, "{$}"), h)
	}

	// Keep this block alphabetized (with e.g. `sort`).
	handle(mux, "GET /-/aud/{id}", c.audioFile)
	handle(mux, "GET /-/audpls/{id}", c.audioMediaPlaylist)
	handle(mux, "GET /-/dl/{id}/{epID}/{sedID}", c.videoDownloadForEpisode)
	handle(mux, "GET /-/dl/{id}/{medID}", c.videoDownloadForMovie)
	handle(mux, "GET /-/img/{id}/{width}", c.image)
	handle(mux, "GET /-/plr/{id}", c.videoPlaylist)
	handle(mux, "GET /-/pls/{id}", c.videoRenditionPlaylist)
	handle(mux, "GET /-/status", c.status)
	handle(mux, "GET /-/sub/{id}", c.subtitleFile)
	handle(mux, "GET /-/subpls/{id}", c.subtitleMediaPlaylist)
	handle(mux, "GET /-/vid/{id}", c.videoStream)
	handle(mux, "POST /-/do/torrent-add", c.doTorrentAdd)
	handle(mux, "POST /-/do/upload", c.doUpload)
	handleStreaming(mux, "POST /-/do/video-upload", c.doVideoUpload)
	mux.Handle("GET /-/icon/{type}/{name}", http.StripPrefix("/-/icon", icon.Handler()))
	mux.Handle("GET /-/static/{name}", static.Handler())
	mux.Handle("GET /-/jassub/{name}", jassub.Handler())

	// Landing point after the degraded server resets the DB and
	// hands control back to the normal server. The browser arrives
	// here via a Refresh-header reload of the original POST URL.
	mux.HandleFunc("GET /-/do/database-reset", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
	})
}

func (c *Config) image(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	id := req.PathValue("id")
	width, err := strconv.Atoi(req.PathValue("width"))
	if err != nil || width <= 0 {
		return nil, errNotFound
	}
	return c.withTxR(ctx, func(tx *model.TxR) (node, error) {
		key, err := tx.ImageVariantKey(id, width)
		if err != nil {
			return nil, errNotFound
		}
		// Pin Content-Type so http.ServeFileFS doesn't fall
		// through to mime sniffing on extensionless keys, and
		// override the middleware CSP with a tighter one so even
		// if a non-image blob ever lands here it can't execute
		// as one. The URL is logically immutable — replacing an
		// image creates a new Image with a fresh ID — so we can
		// serve it as permanently cacheable.
		w.Header().Set("Content-Type", "image/webp")
		w.Header().Set("Cache-Control", "max-age=31536000, immutable")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; img-src 'self'; style-src 'none'; sandbox")
		http.ServeFileFS(w, req, c.Store, key)
		return nil, nil
	})
}

func (c *Config) withTxR(ctx context.Context, f func(*model.TxR) (node, error)) (n node, err error) {
	err = c.Model.WithTxR(ctx, func(tx *model.TxR) error {
		n, err = f(tx)
		return err
	})
	return n, err
}

func (c *Config) withTxRW(ctx context.Context, f func(*model.TxRW) (node, error)) (n node, err error) {
	err = c.Model.WithTxRW(ctx, func(tx *model.TxRW) error {
		n, err = f(tx)
		return err
	})
	return n, err
}

func handle(mux *http.ServeMux, pattern string, hf handlerFunc) {
	mux.Handle(pattern, http.MaxBytesHandler(makeHandler(hf), maxReqBody))
}

// handleStreaming registers a route without the global request-body
// cap. Reserved for endpoints that legitimately need large or
// indefinite bodies — currently just video upload. Such handlers
// MUST consume req.Body via a streaming API so the process doesn't
// buffer the upload into memory.
func handleStreaming(mux *http.ServeMux, pattern string, hf handlerFunc) {
	mux.Handle(pattern, makeHandler(hf))
}

func makeHandler(hf handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		slog.InfoContext(ctx, "request", "url", req.URL)
		var node node
		var err error
		timing.Measure(ctx, "page", func() {
			node, err = hf(w, req)
		})
		if ve, ok := errors.AsType[*model.ValidationError](err); ok {
			handleBadRequest(w, req, ve.Op, ve.Err.Error())
			return
		} else if errors.Is(err, errNotFound) {
			handleNotFound(w, req, req.URL.Path)
			return
		} else if err != nil {
			slog.ErrorContext(ctx, "error", "error", err)
			handleInternal(w, req)
			return
		}
		if node != nil {
			rawHandler(200, node).ServeHTTP(w, req)
		}
	})
}

func rawHandler(code int, node ...node) http.Handler {
	buf := &bytes.Buffer{}
	err := domi.RenderTo(buf, domi.Fragment(node...))
	if err != nil {
		panic(err)
	}
	body := buf.Bytes()
	contentType := "text/html; charset=utf-8"
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(code)
		_, err := w.Write(body)
		if err != nil {
			slog.WarnContext(ctx, err.Error())
			return
		}
	})
}

func stringHandler(contentType, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		w.Header().Set("Content-Type", contentType)
		_, err := io.WriteString(w, body)
		if err != nil {
			slog.WarnContext(ctx, err.Error())
			return
		}
	}
}

func handleBadRequest(w http.ResponseWriter, req *http.Request, title, desc string) {
	http.Error(w, title+": "+desc, http.StatusBadRequest)
}

func handleNotFound(w http.ResponseWriter, req *http.Request, path string) {
	http.Error(w, "not found: "+path, http.StatusNotFound)
}

func handleInternal(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func urlPathHasPrefix(req *http.Request, prefix string) bool {
	prefix = path.Clean(prefix)
	if prefix == "/" {
		return true
	}
	return req.URL.Path == prefix || strings.HasPrefix(req.URL.Path, prefix+"/")
}
