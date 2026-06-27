// Command freeze freezes the pending schema update: it pulls a
// consistent snapshot of the prod database, checks the snapshot is at the
// current published schema, applies the update to a copy via the real
// database.Open path to prove it applies cleanly, checks that the tables
// the update depends on hold real data, writes an attestation report
// under database/report/, and appends the update to database/frozen.txt.
//
// The snapshot is pulled live and without downtime. By default freeze
// runs "VACUUM INTO" through the prod host's sqlite3, which yields a
// single consistent file; -snapshot uses an already-pulled file instead.
// The ssh destination defaults to root@$A3PRODHOST and the remote
// database path to $A3PRODHOSTDB, both overridable by flag.
//
// CI (cmd/frozencheck) re-checks the report against the repository, but
// the data claim — that the snapshot was real and the update applied —
// can only be made here, against the snapshot. See the package doc of
// cmd/frozencheck for the division of labor.
//
// Usage:
//
//	A3PRODHOST=box A3PRODHOSTDB=/var/lib/act3/act3.db freeze
//	freeze -snapshot /tmp/prod.db   # use a local snapshot, no pull
//	freeze -n                       # dry run; print, don't write
package main

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"ily.dev/act3/database"
	"ily.dev/act3/database/schema"
)

// prodHost is the prod hostname. It feeds the default ssh destination.
var prodHost = cmp.Or(os.Getenv("A3PRODHOST"), "localhost")

// databaseDir is the repo's database directory, holding ddl/, frozen.txt,
// and report/. It is not configurable: the schema freeze applies is the
// one compiled into this binary, so it must run against the tree the
// binary was built from. Run freeze from the repo root.
const databaseDir = "database"

func main() {
	snapshot := flag.String("snapshot", "", "use this local snapshot file instead of pulling from the prod host")
	dest := flag.String("ssh", "root@"+prodHost, "ssh destination of the prod host")
	remoteDB := flag.String("remote-db", os.Getenv("A3PRODHOSTDB"), "path to act3.db on the prod host (required unless -snapshot)")
	dryRun := flag.Bool("n", false, "compute and print the report, but write nothing")
	flag.Parse()

	if err := freeze(config{
		snapshot: *snapshot,
		dest:     *dest,
		remoteDB: *remoteDB,
		dryRun:   *dryRun,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "freeze:", err)
		os.Exit(1)
	}
}

type config struct {
	snapshot string
	dest     string
	remoteDB string
	dryRun   bool
}

// entry is a parsed frozen.txt line and a report's update/base object.
type entry struct {
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Name    string `json:"name"`
}

type report struct {
	Update   entry    `json:"update"`
	Base     entry    `json:"base"`
	Snapshot snapshot `json:"snapshot"`
	Rows     rows     `json:"rows"`
	Result   string   `json:"result"`
}

type snapshot struct {
	Digest string `json:"digest"`
	Mtime  string `json:"mtime"`
	Size   int64  `json:"size"`
}

type rows struct {
	Total   int            `json:"total"`
	Touched map[string]int `json:"touched"`
}

func freeze(cfg config) error {
	ddlDir := filepath.Join(databaseDir, "ddl")
	frozenPath := filepath.Join(databaseDir, "frozen.txt")
	reportDir := filepath.Join(databaseDir, "report")

	frozenBytes, err := os.ReadFile(frozenPath)
	if err != nil {
		return err
	}
	frozen, err := parseFrozen(frozenBytes)
	if err != nil {
		return err
	}
	ddlNames, err := updateNames(ddlDir)
	if err != nil {
		return err
	}
	name, version, base, err := pendingUpdate(ddlNames, frozen)
	if err != nil {
		return err
	}

	ddl, err := os.ReadFile(filepath.Join(ddlDir, name))
	if err != nil {
		return err
	}
	touched, err := parseTouches(ddl)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	snapPath := cfg.snapshot
	if snapPath == "" {
		snapPath, err = pullSnapshot(cfg.dest, cfg.remoteDB)
		if err != nil {
			return err
		}
		defer os.Remove(snapPath)
	}

	snap, err := describeSnapshot(snapPath)
	if err != nil {
		return err
	}

	if err := checkSnapshotAtBase(snapPath, base); err != nil {
		return err
	}

	total, perTable, err := snapshotCounts(snapPath, touched)
	if err != nil {
		return err
	}
	for _, t := range touched {
		if perTable[t] == 0 {
			return fmt.Errorf("touched table %q is empty in the snapshot; the update was not exercised against real data", t)
		}
	}

	gotVersion, digest, err := applyAndRead(snapPath)
	if err != nil {
		return fmt.Errorf("applying %s: %w", name, err)
	}
	if gotVersion != version {
		return fmt.Errorf("snapshot is not at the expected schema: applying reached version %s, want %s (is the snapshot at %s?)",
			gotVersion, version, base.Version)
	}

	rep := report{
		Update:   entry{Version: version, Digest: digest, Name: name},
		Base:     base,
		Snapshot: snap,
		Rows:     rows{Total: total, Touched: perTable},
		Result:   "ok",
	}
	repJSON, err := marshalReport(rep)
	if err != nil {
		return err
	}
	frozenLine := fmt.Sprintf("%s %s %s\n", version, digest, name)

	if cfg.dryRun {
		fmt.Printf("would write %s:\n%s\n", filepath.Join(reportDir, name+".json"), repJSON)
		fmt.Printf("would append to %s:\n%s", frozenPath, frozenLine)
		return nil
	}

	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(reportDir, name+".json"), repJSON, 0o644); err != nil {
		return err
	}
	if err := appendLine(frozenPath, frozenLine); err != nil {
		return err
	}
	fmt.Printf("froze %s (digest %s); wrote report and updated frozen.txt\n", name, digest)
	return nil
}

func parseFrozen(b []byte) ([]entry, error) {
	var out []entry
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 3 {
			return nil, fmt.Errorf("frozen.txt: malformed entry %q", line)
		}
		out = append(out, entry{Version: f[0], Digest: f[1], Name: f[2]})
	}
	return out, sc.Err()
}

func updateNames(ddlDir string) ([]string, error) {
	ents, err := os.ReadDir(ddlDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// pendingUpdate finds the one schema update present in ddl but not yet in
// frozen.txt — the unfrozen tip to be frozen — and returns its name and
// version along with the base: the last frozen entry, the schema the
// update is applied on top of.
func pendingUpdate(ddlNames []string, frozen []entry) (name, version string, base entry, err error) {
	isFrozen := make(map[string]bool, len(frozen))
	for _, f := range frozen {
		isFrozen[f.Name] = true
	}
	var pending []string
	for _, n := range ddlNames {
		if !isFrozen[n] {
			pending = append(pending, n)
		}
	}
	switch len(pending) {
	case 0:
		return "", "", entry{}, errors.New("no unfrozen schema update to freeze")
	case 1:
	default:
		return "", "", entry{}, fmt.Errorf("%d unfrozen updates; only one may be frozen at a time: %s",
			len(pending), strings.Join(pending, ", "))
	}
	if len(frozen) == 0 {
		return "", "", entry{}, errors.New("frozen.txt is empty; freeze the bootstrap updates by hand first")
	}
	name = pending[0]
	base = frozen[len(frozen)-1]
	version, _, _ = strings.Cut(name, "_")
	if version <= base.Version {
		return "", "", entry{}, fmt.Errorf("pending update %s does not sort after base %s", name, base.Name)
	}
	return name, version, base, nil
}

// parseTouches reads the "-- touches: a, b, c" directive from a schema
// update's leading comments. It names the existing tables whose data the
// update depends on, so freeze can confirm they are non-empty. The
// directive is required (an update with no such dependency writes an
// empty list), which forces the author to consider it.
func parseTouches(ddl []byte) ([]string, error) {
	sc := bufio.NewScanner(bytes.NewReader(ddl))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		rest, ok := strings.CutPrefix(line, "--")
		if !ok {
			break // reached SQL; the directive must be in the header comments
		}
		list, ok := strings.CutPrefix(strings.TrimSpace(rest), "touches:")
		if !ok {
			continue
		}
		var tables []string
		for t := range strings.SplitSeq(list, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tables = append(tables, t)
			}
		}
		return tables, nil
	}
	return nil, errors.New(`missing "-- touches:" directive in the header comment`)
}

// snapshotCounts returns the total row count across all user tables in the
// snapshot and the per-table counts for the touched tables. A touched
// table that does not exist is an error.
func snapshotCounts(path string, touched []string) (total int, perTable map[string]int, err error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return 0, nil, err
	}
	defer db.Close()

	names, err := userTables(db)
	if err != nil {
		return 0, nil, err
	}
	for _, n := range names {
		c, err := tableCount(db, n)
		if err != nil {
			return 0, nil, err
		}
		total += c
	}
	perTable = make(map[string]int, len(touched))
	for _, t := range touched {
		c, err := tableCount(db, t)
		if err != nil {
			return 0, nil, fmt.Errorf("touched table %q: %w", t, err)
		}
		perTable[t] = c
	}
	return total, perTable, nil
}

func userTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

func tableCount(db *sql.DB, table string) (int, error) {
	var n int
	// Table names are quoted; they come from sqlite_master or the
	// repo's own touches directive, never from untrusted input.
	err := db.QueryRow(`SELECT count(*) FROM "` + strings.ReplaceAll(table, `"`, `""`) + `"`).Scan(&n)
	return n, err
}

// checkSnapshotAtBase requires the snapshot to be exactly at the base
// schema, so applying the pending update exercises it on top of the real
// published schema and applies only that one update. database.Open would
// otherwise roll a snapshot that is behind the base forward through the
// intervening updates, and the report would falsely claim the update was
// tested from base.
func checkSnapshotAtBase(snapshotPath string, base entry) error {
	version, digest, err := snapshotVersion(snapshotPath)
	if err != nil {
		return fmt.Errorf("reading snapshot schema version: %w", err)
	}
	if version != base.Version || digest != base.Digest {
		return fmt.Errorf("snapshot is at schema %s (%s), but the pending update applies on top of base %s (%s); pull a current snapshot",
			version, digest, base.Version, base.Digest)
	}
	return nil
}

// snapshotVersion reads the schema version and rolling digest the snapshot
// is currently at.
func snapshotVersion(snapshotPath string) (version, digest string, err error) {
	db, err := sql.Open("sqlite", snapshotPath+"?mode=ro")
	if err != nil {
		return "", "", err
	}
	defer db.Close()
	v, err := schema.New(context.Background(), db).SchemaVersionGet()
	if err != nil {
		return "", "", err
	}
	return v.Version, v.Digest, nil
}

// applyAndRead copies the snapshot and applies the pending update through
// the real database.Open path, returning the schema version and rolling
// digest recorded for the update. The copy is necessary because Open
// applies updates in place.
func applyAndRead(snapshotPath string) (version, digest string, err error) {
	tmp, err := os.CreateTemp("", "act3-freeze-*.db")
	if err != nil {
		return "", "", err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			os.Remove(tmpPath + suffix)
		}
	}()
	if err := copyFile(snapshotPath, tmpPath); err != nil {
		return "", "", err
	}

	dbr, dbw, err := database.Open(tmpPath)
	if err != nil {
		return "", "", err
	}
	defer dbw.Close()
	defer dbr.Close()

	v, err := schema.New(context.Background(), dbr).SchemaVersionGet()
	if err != nil {
		return "", "", err
	}
	return v.Version, v.Digest, nil
}

func describeSnapshot(path string) (snapshot, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return snapshot{}, err
	}
	digest, err := fileDigest(path)
	if err != nil {
		return snapshot{}, err
	}
	return snapshot{
		Digest: digest,
		Mtime:  fi.ModTime().UTC().Format(time.RFC3339),
		Size:   fi.Size(),
	}, nil
}

func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// pullSnapshot fetches a consistent, downtime-free snapshot of the prod
// database via "VACUUM INTO" run by the prod host's sqlite3, then copies
// it back. VACUUM INTO yields a single compacted file with no -wal
// sidecar. dest is an ssh destination such as root@host.
func pullSnapshot(dest, remoteDB string) (string, error) {
	if remoteDB == "" {
		return "", errors.New("set A3PRODHOSTDB or -remote-db (path to act3.db on the prod host), or use -snapshot")
	}
	local, err := os.CreateTemp("", "act3-snapshot-*.db")
	if err != nil {
		return "", err
	}
	local.Close()

	remoteTmp := fmt.Sprintf("/tmp/act3-snapshot-%d.db", os.Getpid())
	vacuum := fmt.Sprintf("sqlite3 %s \"VACUUM INTO '%s'\"", shellQuote(remoteDB), remoteTmp)
	if err := run("ssh", dest, vacuum); err != nil {
		os.Remove(local.Name())
		return "", fmt.Errorf("remote VACUUM INTO: %w", err)
	}
	defer run("ssh", dest, "rm -f "+shellQuote(remoteTmp))
	if err := run("scp", dest+":"+remoteTmp, local.Name()); err != nil {
		os.Remove(local.Name())
		return "", fmt.Errorf("copying snapshot: %w", err)
	}
	return local.Name(), nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func marshalReport(rep report) ([]byte, error) {
	b, err := json.Marshal(rep, jsontext.WithIndent("\t"), json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func appendLine(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(line); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
