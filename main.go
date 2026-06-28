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
	"ily.dev/act3/http/primaryredirect"
	"ily.dev/act3/http/requestid"
	"ily.dev/act3/http/secureheader"
	"ily.dev/act3/http/timing"
	"ily.dev/act3/log/logcontext"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tmdb"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/storage"
	"ily.dev/act3/video/ffmpeg"
	"ily.dev/act3/web"
	"ily.dev/domi"
)

//go:generate sh -c "cp assets/*.png assets/*.jpeg assets/*.svg web/static/static"
//go:generate go run web/static/gen.go
//go:generate web/static/genjs.sh

var (
	databaseDir = getenv("A3DATABASE", ".")
	storageDir  = getenv("A3STORAGE", "/var/lib/act3")
	primaryURL  = os.Getenv("A3URL")
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
	slog.SetDefault(slog.New(logcontext.Handler(slog.Default().Handler(),
		domiSessionLogAttrs,
	)))
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
		handleSchemaMismatch(sme, dbPath)
	}

	datDir := filepath.Join(storageDir, "dat")
	pass1Dir := filepath.Join(storageDir, "pass1")
	tmpDir := filepath.Join(storageDir, "tmp")
	if err := os.MkdirAll(datDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(pass1Dir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Scratch space for in-flight encodes. Contents are worthless
	// after a crash or restart, so start from empty.
	if err := os.RemoveAll(tmpDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ffmpeg.SetScratchDir(tmpDir)

	if v := os.Getenv("A3TRANSMISSION"); v != "" {
		model.SettingDefaultString(model.SettingKeyTransmissionBaseURL, v)
	}
	if v := os.Getenv("A3TMDBTOKEN"); v != "" {
		model.SettingDefaultString(model.SettingKeyTMDBAccessToken, v)
	}
	if v := os.Getenv("A3FFMPEGVIDEOPRESET"); v != "" {
		ffmpeg.OverridePreset(v)
	}

	store := must(storage.Open(datDir))
	tmdbClient := tmdb.New()
	tvmazeClient := tvmaze.New()
	m := must(model.New(dbr, dbw, model.Config{
		Store:    store,
		Pass1Dir: pass1Dir,
		TMDB:     tmdbClient,
		TVmaze:   tvmazeClient,
	}))

	mux := &http.ServeMux{}
	web.Handle(mux, &web.Config{
		Model:  m,
		Store:  store.FS(),
		TMDB:   tmdbClient,
		TVmaze: tvmazeClient,
	})
	h := http.Handler(mux)
	h = primaryredirect.Handler(primaryURL, h)
	h = panicstack.Handler(h)
	h = timing.Handler(h)
	h = requestid.Handler(h)
	h = (&http.CrossOriginProtection{}).Handler(h)
	h = secureheader.Handler(h)
	slog.Info("listen", "listen", listen)
	panic(http.ListenAndServe(listen, h))
}

// domiSessionLogAttrs is a logcontext extractor
// that surfaces the domi session ID on log lines
// emitted within an App or Cmd.
func domiSessionLogAttrs(ctx context.Context) []slog.Attr {
	if id := domi.SessionID(ctx); id != "" {
		return []slog.Attr{slog.Group("domi", "sessionid", id)}
	}
	return nil
}

func must[T any](v T, err error) T {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return v
}
