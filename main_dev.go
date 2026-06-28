//go:build !prod

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	charmlog "github.com/charmbracelet/log"

	"ily.dev/act3/database"
	"ily.dev/act3/http/panicstack"
	"ily.dev/act3/http/primaryredirect"
	"ily.dev/act3/http/requestid"
	"ily.dev/act3/http/secureheader"
	"ily.dev/act3/http/timing"
	"ily.dev/act3/view"
	"ily.dev/act3/web"
)

// newLogHandler returns the development slog backend: charmbracelet/log,
// which renders colorized, human-readable log lines. Timestamps are
// omitted to keep lines terse during local development, and the caller
// is shown as a project-root-relative "path/to/file.go:line".
//
// mainPackagePrefix is unused here; the dev caller format is derived
// from source file paths rather than package-qualified function names.
func newLogHandler(w io.Writer, level slog.Level, _ string) slog.Handler {
	root := projectRoot()
	return charmlog.NewWithOptions(w, charmlog.Options{
		Level:        charmlog.Level(level),
		ReportCaller: true,
		CallerFormatter: func(file string, line int, _ string) string {
			return fmt.Sprintf("%s:%d", strings.TrimPrefix(file, root), line)
		},
	})
}

// projectRoot returns the module root as an absolute path with a
// trailing separator, derived from the compile-time location of this
// source file (which sits at the module root). It is used to render
// caller locations relative to the project root.
func projectRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Dir(file) + string(filepath.Separator)
}

// handleSchemaMismatch serves the schema-mismatch UI
// with a "reinitialize database" affordance.
// This is a development convenience
// that wipes the database and starts over with a fresh schema.
// It returns once the database has been reset,
// so the caller can retry opening it,
// with no need to exit or start a new process.
//
// This destructive affordance is for dev builds only.
// In prod it is compiled out. See main_prod.go.
func handleSchemaMismatch(sme *database.SchemaMismatchError, dbPath string) {
	slog.Warn("schema mismatch, entering degraded mode",
		"version", sme.Version,
		"stored", sme.StoredDigest,
		"expected", sme.ExpectedDigest,
	)
	stats, err := database.TableStats(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var dbFileSize int64
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if info, err := os.Stat(dbPath + suffix); err == nil {
			dbFileSize += info.Size()
		}
	}
	page := view.Degraded(sme, stats, dbFileSize)

	srv := &http.Server{Addr: listen}
	mux := &http.ServeMux{}
	web.HandleDegraded(mux, page, func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			os.Remove(dbPath + suffix)
		}
		go srv.Shutdown(context.Background())
	})
	var h http.Handler = mux
	h = primaryredirect.Handler(primaryURL, h)
	h = panicstack.Handler(h)
	h = timing.Handler(h)
	h = requestid.Handler(h)
	h = (&http.CrossOriginProtection{}).Handler(h)
	h = secureheader.Handler(h)
	srv.Handler = h
	slog.Info("degraded mode", "listen", listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
