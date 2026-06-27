//go:build prod

package main

import (
	"log/slog"
	"os"

	"ily.dev/act3/database"
)

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
