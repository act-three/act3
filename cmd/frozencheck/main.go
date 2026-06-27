// Command frozencheck verifies database/frozen.txt against its previous
// revision. Two checks run:
//
//   - Append-only: the previous revision must be a line-prefix of the
//     current one. Already-frozen schema updates may never change or be
//     removed; new entries may only be appended.
//   - Attestation: each newly-frozen update applied on top of a
//     previously published schema must carry a matching report under
//     database/report/, proving the freeze tool ran against real prod
//     data. See the report fields validated by checkReports.
//
// Usage:
//
//	git show <base>:database/frozen.txt | frozencheck database/frozen.txt
//
// The previous revision is read from stdin and the current file is
// named by the sole argument. Reports are read from a report/ directory
// beside that file. Intended for CI, where <base> is the merge base
// with the target branch.
package main

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: frozencheck <file>  (previous revision on stdin)")
		os.Exit(2)
	}
	old, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading stdin:", err)
		os.Exit(2)
	}
	cur, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading file:", err)
		os.Exit(2)
	}
	if err := check(old, cur); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	reports := os.DirFS(filepath.Join(filepath.Dir(os.Args[1]), "report"))
	if err := checkReports(old, cur, reports); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// check reports whether cur preserves old as a line-prefix: every line
// of old must appear unchanged at the same position in cur, and cur may
// only add lines after them. The returned error names the first
// divergence.
func check(old, cur []byte) error {
	o, c := lines(old), lines(cur)
	if len(c) < len(o) {
		return fmt.Errorf("frozen.txt: %d frozen entries removed; entries are append-only", len(o)-len(c))
	}
	for i := range o {
		if o[i] != c[i] {
			return fmt.Errorf("frozen.txt:%d: frozen entry changed; entries are append-only\n  was: %s\n  now: %s",
				i+1, o[i], c[i])
		}
	}
	return nil
}

// entry is a parsed frozen.txt line: "version digest name".
type entry struct {
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Name    string `json:"name"`
}

// report is the subset of a database/report/<name>.json attestation that
// CI can verify against the repository. The freeze tool writes more
// (snapshot identity, per-table counts); those are the data claim it
// vouches for and are ignored here.
type report struct {
	Update entry `json:"update"`
	Base   entry `json:"base"`
	Rows   struct {
		Total int `json:"total"`
	} `json:"rows"`
	Result string `json:"result"`
}

// checkReports verifies the attestation report for each newly-frozen
// update. A line is newly frozen when it appears in cur but not old;
// since check has confirmed old is a line-prefix of cur, those are
// exactly the lines past len(old).
//
// A report is required only when the update is applied on top of an
// already-published schema, i.e. its base (the preceding line) was
// already frozen in old. The bootstrap updates, frozen before any prod
// data existed, have no earlier frozen base and so need no report.
func checkReports(old, cur []byte, reports fs.FS) error {
	o, c := lines(old), lines(cur)
	for i := len(o); i < len(c); i++ {
		if i == 0 || i-1 >= len(o) {
			// No base, or a base frozen in this same change: there was
			// no published prod schema to update from.
			continue
		}
		up, err := parseEntry(c[i])
		if err != nil {
			return err
		}
		base, err := parseEntry(c[i-1])
		if err != nil {
			return err
		}
		rep, err := readReport(reports, up.Name)
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("database/report/%s.json: missing; a frozen schema update needs an attestation report", up.Name)
		} else if err != nil {
			return err
		}
		if rep.Update != up {
			return fmt.Errorf("database/report/%s.json: update %+v does not match frozen entry %+v", up.Name, rep.Update, up)
		}
		if rep.Base != base {
			return fmt.Errorf("database/report/%s.json: base %+v does not match prior frozen entry %+v", up.Name, rep.Base, base)
		}
		if rep.Result != "ok" {
			return fmt.Errorf("database/report/%s.json: result is %q, want \"ok\"", up.Name, rep.Result)
		}
		// The base's report, when it has one, sets the row-count floor.
		// The first real report has no predecessor and so no floor.
		// Reading the base report from the working tree is sound because
		// committed reports are immutable, enforced in CI beside the
		// frozen.txt append-only check; otherwise a PR could lower an
		// earlier report's total to slip a lower new total past the floor.
		prev, err := readReport(reports, base.Name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if rep.Rows.Total < prev.Rows.Total {
			return fmt.Errorf("database/report/%s.json: rows.total %d is below the previous report's %d",
				up.Name, rep.Rows.Total, prev.Rows.Total)
		}
	}
	return nil
}

func parseEntry(line string) (entry, error) {
	f := strings.Fields(line)
	if len(f) != 3 {
		return entry{}, fmt.Errorf("frozen.txt: malformed entry %q", line)
	}
	return entry{Version: f[0], Digest: f[1], Name: f[2]}, nil
}

func readReport(reports fs.FS, name string) (report, error) {
	b, err := fs.ReadFile(reports, name+".json")
	if err != nil {
		return report{}, err
	}
	var r report
	if err := json.Unmarshal(b, &r); err != nil {
		return report{}, fmt.Errorf("database/report/%s.json: %w", name, err)
	}
	return r, nil
}

func lines(b []byte) []string {
	b = bytes.TrimRight(b, "\n")
	if len(b) == 0 {
		return nil
	}
	return strings.Split(string(b), "\n")
}
