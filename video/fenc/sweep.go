package fenc

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// SweepOrphans removes every entry directly under the spool and
// stats roots that was last modified before now−age.
// Both roots exist solely to hold job and stats state,
// so anything old there — including stray files left by a crash
// mid-staging — is garbage.
// act3 normally removes its own job directories and releases
// stats explicitly;
// the sweep is the backstop that reclaims space after a crash on
// either side.
// The agent runs it once at startup.
func (s *Server) SweepOrphans(age time.Duration) {
	cutoff := time.Now().Add(-age)
	for _, root := range []string{s.Spool, s.Stats} {
		entries, err := os.ReadDir(root)
		if err != nil {
			slog.Warn("sweep-readdir", "dir", root, "err", err)
			continue
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil || !info.ModTime().Before(cutoff) {
				continue
			}
			if err := os.RemoveAll(filepath.Join(root, e.Name())); err != nil {
				slog.Warn("sweep-remove", "dir", root, "name", e.Name(), "err", err)
				continue
			}
			slog.Info("sweep-removed", "dir", root, "name", e.Name(),
				"age", time.Since(info.ModTime()).Round(time.Hour))
		}
	}
}
