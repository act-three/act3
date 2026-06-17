package web

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"ily.dev/act3/view"
	"ily.dev/act3/web/static"
	"ily.dev/domi"
)

// HandleDegraded registers routes for the degraded mode UI
// shown when the database schema does not match. The reset
// callback must remove the database files and stop the
// degraded server.
func HandleDegraded(mux *http.ServeMux, page node, reset func()) {
	mux.Handle("GET /-/static/", static.Handler())

	buf := &bytes.Buffer{}
	domi.RenderTo(buf, view.Document("Schema Mismatch", page))
	body := buf.Bytes()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := w.Write(body)
		if err != nil {
			slog.WarnContext(req.Context(), err.Error())
			return
		}
	})

	mux.HandleFunc("POST /-/do/database-reset",
		func(w http.ResponseWriter, req *http.Request) {
			reset()
			// Reload the same URL after a delay; by then the
			// degraded server has shut down and the normal server
			// is (hopefully) up. Its GET handler for this path
			// 303s back to /. Reload via Refresh header issues a
			// GET, so no "resubmit form?" prompt and the back
			// button doesn't strand the user on an invalid URL.
			w.Header().Set("Refresh", "2")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, `<!doctype html>
				<html><head></head>
				<body style="background:#111;color:#eee;font-family:system-ui;padding:2rem">
				<p>Reinitializing database…</p>
				</body></html>
			`)
			http.NewResponseController(w).Flush()
		})
}
