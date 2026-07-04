// Command fenc is the encoder agent that runs inside the fenc box:
// a minimal HTTP service that executes ffmpeg and ffprobe jobs on
// act3's behalf. See ily.dev/act3/video/fenc for the protocol.
//
// Configuration comes from the environment:
//
//	FENCLISTEN  address to listen on (default :4446)
//	FENCSPOOL   dir holding job directories, shared with act3
//	            (default /mnt/shared)
//	FENCSTATS   dir holding pass-1 encoder stats
//	            (default /mnt/private/stats)
//	FENCSWEEP   age above which job and stats dirs are removed at
//	            startup; 0 disables (default 168h)
//
// The directory defaults match the shim appliance layout.
package main

import (
	"cmp"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"ily.dev/act3/video/fenc"
)

var (
	listen   = cmp.Or(os.Getenv("FENCLISTEN"), ":4446")
	spool    = cmp.Or(os.Getenv("FENCSPOOL"), "/mnt/shared")
	stats    = cmp.Or(os.Getenv("FENCSTATS"), "/mnt/private/stats")
	sweepAge = cmp.Or(os.Getenv("FENCSWEEP"), "168h")
)

var verbose bool

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose output (log level = debug)")
}

func main() {
	flag.Parse()
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	sweep, err := time.ParseDuration(sweepAge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FENCSWEEP: %v\n", err)
		os.Exit(1)
	}
	// The spool is a mount shared with act3, not a directory the
	// agent may create: MkdirAll on an absent mount would invent a
	// local directory and silently absorb jobs. Fail fast instead
	// so the box's service state shows the misconfiguration.
	if fi, err := os.Stat(spool); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else if !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "spool %s is not a directory\n", spool)
		os.Exit(1)
	}
	if err := os.MkdirAll(stats, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := &fenc.Server{Spool: spool, Stats: stats}
	if sweep > 0 {
		s.SweepOrphans(sweep)
	}
	slog.Info("listen", "listen", listen)
	panic(http.ListenAndServe(listen, s.Handler()))
}
