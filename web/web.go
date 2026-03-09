package web

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/http/timing"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tvmaze"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/icon"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/web/static"
)

var errNotFound = errors.New("not found")

// Max request body for all requests except video uploads.
const maxReqBody = 10 * 1000 * 1000

type handlerFunc func(http.ResponseWriter, *http.Request) (html.Node, error)

type Config struct {
	Model  *model.Model
	Store  fs.FS
	TVmaze *tvmaze.Client
}

func Handle(mux *http.ServeMux, c *Config) {
	handle(mux, "GET /app/profile", c.accountProfile)
	handle(mux, "GET /app/security", c.accountSecurity)
	handle(mux, "GET /-/dialog/series-add", c.seriesAddDialogReq)
	handle(mux, "GET /-/dialog/episode-edit/{id}", c.dialogEditEpisode)
	handle(mux, "GET /-/dl/{hash}/{name}", c.videoDownload)
	handle(mux, "GET /app/downloads", c.editDownloads)
	handle(mux, "GET /app/downloads/{id}", c.editDownloadsDetail)
	handle(mux, "GET /app/movies", c.editMovies)
	handle(mux, "GET /app/movies/{id}", c.editMoviesDetail)
	handle(mux, "GET /app/series", c.editSeries)
	handle(mux, "GET /app/series/{id}", c.editSeriesDetail)
	handle(mux, "GET /-/player/{id}/{epID}/{sedID}", c.showPlayerForEpisode)
	handle(mux, "GET /-/player/{id}/{moID}", c.showPlayerForMovie)
	handle(mux, "GET /-/part/series-search", c.seriesSearch)
	handle(mux, "GET /app/storage", c.systemStorage)
	handle(mux, "GET /app/tasks", c.systemTasks)
	handle(mux, "GET /app/transmission", c.systemTransmission)
	handle(mux, "GET /-/plr/{id}", c.videoPlaylist)
	handle(mux, "GET /-/pls/{id}", c.videoRenditionPlaylist)
	handle(mux, "GET /-/vid/{hash}", c.videoStream)
	handle(mux, "GET /{$}", c.home)
	handle(mux, "GET /{slug}", c.showSeriesOrMovie)
	handle(mux, "GET /{slug}/{epSlug}", c.showEpisode)
	handle(mux, "POST /-/do/add-movie", c.doAddMovie)
	handle(mux, "POST /-/do/add-series", c.doAddSeries)
	handle(mux, "POST /-/do/add-torrent", c.doAddTorrent)
	handle(mux, "POST /-/do/update-transmission-settings", c.doUpdateTransmissionSettings)
	handle(mux, "POST /-/do/run-task/{id}", c.doRunTask)
	handle(mux, "POST /-/do/delete-task/{id}", c.doDeleteTask)
	handle(mux, "POST /-/do/kill-task/{id}", c.doKillTask)
	handle(mux, "POST /-/do/reimport-video/{id}", c.doReimportVideo)
	handle(mux, "POST /-/do/reencode-video/{id}", c.doReencodeVideo)
	mux.HandleFunc("GET /-/events", c.events)
	mux.Handle("GET /-/icon/", http.StripPrefix("/-/icon", icon.Handler()))
	mux.Handle("GET /-/static/", static.Handler())
}

func (c *Config) withTxR(f func(*model.TxR) (html.Node, error)) (n html.Node, err error) {
	err = c.Model.WithTxR(func(tx *model.TxR) error {
		n, err = f(tx)
		return err
	})
	return n, err
}

func (c *Config) withTxRW(f func(*model.TxRW) (html.Node, error)) (n html.Node, err error) {
	err = c.Model.WithTxRW(func(tx *model.TxRW) error {
		n, err = f(tx)
		return err
	})
	return n, err
}

func handle(mux *http.ServeMux, pattern string, hf handlerFunc) {
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		slog.InfoContext(ctx, "request", "url", req.URL)
		var node html.Node
		var err error
		timing.Measure(ctx, "page", func() {
			node, err = hf(w, req)
		})
		var ve *model.ValidationError
		if errors.As(err, &ve) {
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
	h = http.MaxBytesHandler(h, maxReqBody)
	mux.Handle(pattern, h)
}

func rawHandler(code int, node ...html.Node) http.Handler {
	buf := &bytes.Buffer{}
	err := html.Render(buf, html.Group(node...))
	if err != nil {
		panic(err)
	}
	body := buf.Bytes()
	contentType := "text/html"
	if turbo.SniffStream(body) {
		contentType = "text/vnd.turbo-stream.html"
	}
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
	h := rawHandler(400,
		Note(NoteError)(
			NoteTitle()(html.Text(title)),
			NoteDescription()(html.Text(desc)),
		),
	)
	h.ServeHTTP(w, req)
}

func handleNotFound(w http.ResponseWriter, req *http.Request, path string) {
	h := rawHandler(404,
		Note(NoteError)(
			NoteTitle()(html.Text("Not Found")),
			NoteDescription()(html.Text(path)),
		),
	)
	h.ServeHTTP(w, req)
}

func handleInternal(w http.ResponseWriter, req *http.Request) {
	h := rawHandler(500,
		Note(NoteError)(
			NoteTitle()(html.Text("Internal Error")),
		),
	)
	h.ServeHTTP(w, req)
}

func urlPathHasPrefix(req *http.Request, prefix string) bool {
	prefix = path.Clean(prefix)
	if prefix == "/" {
		return true
	}
	return req.URL.Path == prefix || strings.HasPrefix(req.URL.Path, prefix+"/")
}
