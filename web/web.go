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
	"ily.dev/act3/html/attr"
	"ily.dev/act3/http/timing"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/web/static"
)

//go:generate go tool esbuild --bundle --outfile=static/static/bundle.js main.js

var errNotFound = errors.New("not found")

// Max request body for all requests except video uploads.
const maxReqBody = 10 * 1000 * 1000

type handlerFunc func(*http.Request) (http.Handler, error)

type Config struct {
	Model  *model.Model
	Store  fs.FS
	TVmaze *tvmaze.Client
}

func Handle(mux *http.ServeMux, c *Config) {
	w := &web{
		model:  c.Model,
		store:  c.Store,
		tvmaze: c.TVmaze,
	}

	handle(mux, "GET /account/profile", w.accountProfile)
	handle(mux, "GET /account/security", w.accountSecurity)
	handle(mux, "GET /dialog/series-add", w.seriesAddDialogReq)
	handle(mux, "GET /dialog/edit-episode/{id}", w.dialogEditEpisode)
	handle(mux, "GET /dl/{hash}/{name}", w.videoDownload)
	handle(mux, "GET /edit/downloads", w.editDownloads)
	handle(mux, "GET /edit/downloads/{id}", w.editDownloadsDetail)
	handle(mux, "GET /edit/series", w.editSeries)
	handle(mux, "GET /edit/series/{id}", w.editSeriesDetail)
	handle(mux, "GET /ep/{id}", w.showEpisode)
	handle(mux, "GET /player/{id}/{epID}/{sedID}", w.showPlayerForEpisode)
	handle(mux, "GET /search-series", w.seriesSearch)
	handle(mux, "GET /series", w.listSeries)
	handle(mux, "GET /series/{id}", w.showSeries)
	handle(mux, "GET /system/storage", w.systemStorage)
	handle(mux, "GET /system/tasks", w.systemTasks)
	handle(mux, "GET /system/transmission", w.systemTransmission)
	handle(mux, "GET /vid/{id}", w.videoPlaylist)
	handle(mux, "GET /vidr/{id}", w.videoRenditionPlaylist)
	handle(mux, "GET /vids/{hash}", w.videoStream)
	handle(mux, "GET /{$}", w.home)
	handle(mux, "POST /do/add-series", w.doAddSeries)
	handle(mux, "POST /do/add-torrent", w.doAddTorrent)
	handle(mux, "POST /do/update-transmission-settings", w.doUpdateTransmissionSettings)
	handle(mux, "POST /do/run-task/{id}", w.doRunTask)
	handle(mux, "POST /do/delete-task/{id}", w.doDeleteTask)
	mux.Handle("GET /static/", static.FS)
}

type web struct {
	model  *model.Model
	store  fs.FS
	tvmaze *tvmaze.Client
}

func (w *web) withTxR(f func(*model.TxR) (http.Handler, error)) (h http.Handler, err error) {
	err = w.model.WithTxR(func(tx *model.TxR) error {
		h, err = f(tx)
		return err
	})
	return h, err
}

func (w *web) withTxRW(f func(*model.TxRW) (http.Handler, error)) (h http.Handler, err error) {
	err = w.model.WithTxRW(func(tx *model.TxRW) error {
		h, err = f(tx)
		return err
	})
	return h, err
}

func handle(mux *http.ServeMux, pattern string, hf handlerFunc) {
	h := http.MaxBytesHandler(handler(hf), maxReqBody)
	mux.Handle(pattern, h)
}

func handler(hf handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		slog.InfoContext(ctx, "request", "url", req.URL)
		var h http.Handler
		timing.Measure(ctx, "page", func() {
			var err error
			h, err = hf(req)
			var ve *model.ValidationError
			if errors.As(err, &ve) {
				h = errorBadRequest(errorFrameID(err),
					ve.Op,
					ve.Err.Error(),
				)
			} else if errors.Is(err, errNotFound) {
				h = errorNotFound(errorFrameID(err), req.URL.Path)
			} else if err != nil {
				slog.ErrorContext(ctx, "error", "error", err)
				h = errorInternal(errorFrameID(err))
			}
		})
		h.ServeHTTP(w, req)
	}
}

func page(node ...html.Node) http.Handler {
	return rawHandler("text/html", 200, node...)
}

func pageBadRequest(node ...html.Node) http.Handler {
	return rawHandler("text/html", 400, node...)
}

func stream(node ...html.Node) http.Handler {
	return rawHandler("text/vnd.turbo-stream.html", 200, node...)
}

func rawHandler(contentType string, code int, node ...html.Node) http.Handler {
	buf := &bytes.Buffer{}
	err := html.Render(buf, html.Group(node...))
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(code)
		_, err := w.Write(buf.Bytes())
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
func errorBadRequest(turboFrameID, title, desc string) http.Handler {
	return rawHandler("text/vnd.turbo-stream.html", 400,
		turbo.Prepend(turboFrameID,
			html.Div(
				attr.Class("border p-1 bg-crimson-3"),
			)(
				html.Div()(html.Text(title)),
				html.Div()(html.Text(desc)),
			),
		),
	)
}

func errorNotFound(turboFrameID, path string) http.Handler {
	return rawHandler("text/vnd.turbo-stream.html", 404,
		turbo.Prepend(turboFrameID,
			html.Div(
				attr.Class("border p-1 bg-crimson-3"),
			)(
				html.Div()(html.Text("Not Found")),
				html.Div()(html.Text(path)),
			),
		),
	)
}

func errorInternal(turboFrameID string) http.Handler {
	return rawHandler("text/vnd.turbo-stream.html", 500,
		turbo.Prepend(turboFrameID,
			html.Div(
				attr.Class("border p-1 bg-crimson-3"),
			)(
				html.Div()(html.Text("Internal Error")),
			),
		),
	)
}

type frameIDError struct {
	id  string
	err error
}

func decorateErrorFrame(id string, err *error) {
	if *err != nil {
		*err = frameIDError{id, *err}
	}
}

func (e frameIDError) Error() string {
	return e.err.Error()
}

func (e frameIDError) Unwrap() error {
	return e.err
}

func urlPathHasPrefix(req *http.Request, prefix string) bool {
	prefix = path.Clean(prefix)
	if prefix == "/" {
		return true
	}
	return req.URL.Path == prefix || strings.HasPrefix(req.URL.Path, prefix+"/")
}

func errorFrameID(err error) string {
	var fe frameIDError
	if errors.As(err, &fe) {
		return fe.id
	}
	return "errors"
}
