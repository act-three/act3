package web

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"os"

	"ily.dev/act3/database"
	"ily.dev/act3/view"
	"ily.dev/act3/web/static"
)

// HandleDegraded registers routes for the degraded mode UI
// shown when the database schema does not match. The shutdown
// callback is called after the database files are removed so
// the caller can stop the degraded server.
func HandleDegraded(
	mux *http.ServeMux,
	sme *database.SchemaMismatchError,
	db *sql.DB,
	dbPath string,
	shutdown func(),
) {
	mux.Handle("GET /-/static/", static.Handler())

	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		stats, err := database.TableStats(db)
		if err != nil {
			slog.Error("table stats", "error", err)
			http.Error(w, "internal error", 500)
			return
		}
		var dbFileSize int64
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if info, err := os.Stat(dbPath + suffix); err == nil {
				dbFileSize += info.Size()
			}
		}
		node := view.Degraded(sme, stats, fmtSize(dbFileSize))
		rawHandler(200, node).ServeHTTP(w, req)
	})

	mux.HandleFunc("POST /-/do/database-reset",
		func(w http.ResponseWriter, req *http.Request) {
			db.Close()
			for _, suffix := range []string{"", "-wal", "-shm"} {
				os.Remove(dbPath + suffix)
			}
			rc := http.NewResponseController(w)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, `<!doctype html>
<html><head></head><body style="background:#111;color:#eee;
font-family:system-ui;padding:2rem">
<p>Reinitializing database…</p>
<script>setTimeout(function(){location.replace("/")},2000)</script>
</body></html>`)
			rc.Flush()
			shutdown()
		})
}
