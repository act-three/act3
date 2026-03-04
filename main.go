package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"ily.dev/act3/database"
	"ily.dev/act3/http/panicstack"
	"ily.dev/act3/http/requestid"
	"ily.dev/act3/http/timing"
	"ily.dev/act3/log/logcontext"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/storage"
	"ily.dev/act3/web"
)

//go:generate tailwindcss -i main.css -o web/static/static/bundle.css
//go:generate go tool esbuild --bundle --outfile=web/static/static/bundle.js main.js

var (
	databaseDir = getenv("A3DATABASE", ".")
	storageDir  = getenv("A3STORAGE", "/var/lib/act3")
)

func getenv(name, def string) string {
	if s := os.Getenv(name); s != "" {
		return s
	}
	return def
}

var (
	verbose bool
	listen  string
)

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose output (log level = debug)")
	flag.StringVar(&listen, "listen", ":4444", "`address` to listen on")
}

func main() {
	flag.Parse()
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		panic("can't read build info")
	}
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	mainPackagePrefix := bi.Main.Path + "/"
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey, slog.LevelKey:
				return slog.Attr{}
			case slog.SourceKey:
				s := a.Value.Any().(*slog.Source).Function
				s = strings.TrimPrefix(s, mainPackagePrefix)
				s, _, _ = strings.Cut(s, ".")
				return slog.String(a.Key, s)
			}
			return a
		},
	})))
	slog.SetDefault(slog.New(logcontext.Handler(slog.Default().Handler())))
	slog.Info("startup", "mod", bi.Main.Path, "version", bi.Main.Version)

	dbPath := filepath.Join(databaseDir, "act3.db")
	var dbr, dbw *sql.DB
	for {
		var err error
		dbr, dbw, err = database.Open(dbPath)
		if err == nil {
			break
		}
		var sme *database.SchemaMismatchError
		if !errors.As(err, &sme) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		slog.Warn("schema mismatch, entering degraded mode",
			"version", sme.Version,
			"stored", sme.StoredDigest,
			"expected", sme.ExpectedDigest,
		)
		serveDegraded(sme, dbPath)
	}

	casDir := filepath.Join(storageDir, "cas")
	tmpDir := filepath.Join(storageDir, "tmp")
	if err := os.MkdirAll(casDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := must(storage.Open(casDir))
	tvmazeClient := must(tvmaze.New(dbw))
	m := must(model.New(dbr, dbw, model.Config{
		Store:         store,
		PersistentTmp: tmpDir,
		TVmaze:        tvmazeClient,
	}))

	if err := initConfig(m); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mux := &http.ServeMux{}
	web.Handle(mux, &web.Config{
		Model:  m,
		Store:  store.FS(),
		TVmaze: tvmazeClient,
	})
	h := http.Handler(mux)
	h = panicstack.Handler(h)
	h = timing.Handler(h)
	h = requestid.Handler(h)
	h = (&http.CrossOriginProtection{}).Handler(h)
	slog.Info("listen", "listen", listen)
	panic(http.ListenAndServe(listen, h))
}

func initConfig(m *model.Model) error {
	return m.WithTxRW(func(tx *model.TxRW) error {
		ctx := context.Background()
		ct, err := tx.Transmission(ctx)
		if err != nil {
			return err
		}
		if ct.BaseURL == "" {
			ct.BaseURL = os.Getenv("A3TRANSMISSION")
			err = tx.TransmissionSet(ctx, *ct)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func serveDegraded(sme *database.SchemaMismatchError, dbPath string) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()

	shutdown := make(chan bool)
	srv := &http.Server{Addr: listen}
	mux := &http.ServeMux{}
	web.HandleDegraded(mux, sme, db, dbPath, func() {
		go func() {
			srv.Shutdown(context.Background())
			close(shutdown)
		}()
	})
	var h http.Handler = mux
	h = panicstack.Handler(h)
	h = timing.Handler(h)
	h = requestid.Handler(h)
	srv.Handler = h
	slog.Info("degraded mode", "listen", listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	<-shutdown
}

func must[T any](v T, err error) T {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return v
}
