package web

import (
	"io"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/web/static"
)

// HandleDegraded registers routes for the degraded mode UI
// shown when the database schema does not match. The reset
// callback must remove the database files and stop the
// degraded server.
func HandleDegraded(mux *http.ServeMux, page html.Node, reset func()) {
	mux.Handle("GET /-/static/", static.Handler())

	mux.Handle("GET /", rawHandler(200, page))

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
<html><head></head><body style="background:#111;color:#eee;
font-family:system-ui;padding:2rem">
<p>Reinitializing database…</p>
</body></html>`)
			http.NewResponseController(w).Flush()
		})
}
