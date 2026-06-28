//go:build prod

package main

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"ily.dev/act3/database"
)

// newLogHandler returns the production slog backend: a plain text
// handler writing to w. Timestamps and levels are dropped, and source
// locations are reduced to the package name with mainPackagePrefix
// stripped.
func newLogHandler(w io.Writer, level slog.Level, mainPackagePrefix string) slog.Handler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{
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
	})
}

// handleSchemaMismatch logs remediation guidance and exits non-zero.
// In production, a schema mismatch is a fatal error.
func handleSchemaMismatch(sme *database.SchemaMismatchError, dbPath string) {
	slog.Error("schema mismatch, cannot start",
		"version", sme.Version,
		"stored", sme.StoredDigest,
		"expected", sme.ExpectedDigest,
		"path", dbPath,
		"hint", "restore from a backup or roll back the schema to the expected version",
	)
	os.Exit(1)
}
