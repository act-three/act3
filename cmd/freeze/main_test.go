package main

import (
	"database/sql"
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFrozen(t *testing.T) {
	got, err := parseFrozen([]byte("000 d0 000_meta.up.sql\n001 d1 001_init.up.sql\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []entry{
		{"000", "d0", "000_meta.up.sql"},
		{"001", "d1", "001_init.up.sql"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("parseFrozen = %v, want %v", got, want)
	}
	if _, err := parseFrozen([]byte("001 onlytwo\n")); err == nil {
		t.Error("parseFrozen accepted a malformed entry")
	}
}

func TestPendingUpdate(t *testing.T) {
	e000 := entry{"000", "d0", "000_meta.up.sql"}
	e001 := entry{"001", "d1", "001_init.up.sql"}

	t.Run("one pending", func(t *testing.T) {
		name, version, base, err := pendingUpdate(
			[]string{"000_meta.up.sql", "001_init.up.sql", "002_foo.up.sql"},
			[]entry{e000, e001},
		)
		if err != nil {
			t.Fatal(err)
		}
		if name != "002_foo.up.sql" || version != "002" || base != e001 {
			t.Fatalf("got (%q, %q, %v)", name, version, base)
		}
	})
	t.Run("none pending", func(t *testing.T) {
		_, _, _, err := pendingUpdate([]string{"000_meta.up.sql", "001_init.up.sql"}, []entry{e000, e001})
		if err == nil {
			t.Error("want error when nothing is unfrozen")
		}
	})
	t.Run("two pending", func(t *testing.T) {
		_, _, _, err := pendingUpdate(
			[]string{"000_meta.up.sql", "001_init.up.sql", "002_a.up.sql", "003_b.up.sql"},
			[]entry{e000, e001},
		)
		if err == nil {
			t.Error("want error when more than one is unfrozen")
		}
	})
}

func TestSnapshotRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snap.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE movie (id INTEGER PRIMARY KEY)`,
		`INSERT INTO movie (id) VALUES (1), (2), (3)`,
		`CREATE TABLE episode (id INTEGER PRIMARY KEY)`,
		`INSERT INTO episode (id) VALUES (1)`,
		`CREATE TABLE empty (id INTEGER PRIMARY KEY)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	total, err := snapshotRows(path)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
}

// TestApplyAndRead exercises the real database.Open update path: it seeds a
// database at schema version 000, then confirms applyAndRead walks every
// later update, reads back the version and rolling digest recorded for the
// newest one (matching frozen.txt when it is frozen), and counts no changed
// data rows — the seeded tables are empty, so backfills touch nothing.
func TestApplyAndRead(t *testing.T) {
	path, frozen := seedAt000(t)
	names, err := updateNames(filepath.FromSlash("../../database/ddl"))
	if err != nil {
		t.Fatal(err)
	}
	newest := frozen[len(frozen)-1]
	if name := names[len(names)-1]; name != newest.Name {
		// The newest update is still unfrozen; its rolling digest is
		// recorded nowhere, so expect only its version.
		version, _, _ := strings.Cut(name, "_")
		newest = entry{Version: version, Name: name}
	}

	version, digest, affected, err := applyAndRead(path)
	if err != nil {
		t.Fatal(err)
	}
	if version != newest.Version || (newest.Digest != "" && digest != newest.Digest) {
		t.Fatalf("applyAndRead = (%q, %q), want (%q, %q)",
			version, digest, newest.Version, newest.Digest)
	}
	// applyAndRead subtracts one bookkeeping row assuming a single applied
	// update; each additional update applied from 000 leaves one more.
	if want := len(names) - 2; affected != want {
		t.Errorf("affected = %d, want %d (updates change no data rows on empty tables)", affected, want)
	}
}

func TestCheckSnapshotAtBase(t *testing.T) {
	path, frozen := seedAt000(t)
	base000, base001 := frozen[0], frozen[1]

	if err := checkSnapshotAtBase(path, base000); err != nil {
		t.Errorf("snapshot at 000, base 000: unexpected error %v", err)
	}
	// Snapshot behind the base: the stale-snapshot bug this guards against.
	if err := checkSnapshotAtBase(path, base001); err == nil {
		t.Error("want error when the snapshot is behind the base (000 vs 001)")
	}
	wrongDigest := base000
	wrongDigest.Digest = "0000000000000000"
	if err := checkSnapshotAtBase(path, wrongDigest); err == nil {
		t.Error("want error when the snapshot digest differs from the base")
	}
}

// seedAt000 builds a database parked at the frozen 000 schema version: it
// runs 000_meta and stamps the schema row with the frozen 000 entry, so
// database.Open applies 001 next. It returns the db path and the frozen
// entries (frozen[0] is 000, frozen[1] is 001).
func seedAt000(t *testing.T) (path string, frozen []entry) {
	t.Helper()
	dbDir := filepath.FromSlash("../../database")
	frozenBytes, err := os.ReadFile(filepath.Join(dbDir, "frozen.txt"))
	if err != nil {
		t.Fatal(err)
	}
	frozen, err = parseFrozen(frozenBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(frozen) < 2 {
		t.Skipf("need at least 000 and 001 frozen; have %d", len(frozen))
	}
	meta, err := os.ReadFile(filepath.Join(dbDir, "ddl", "000_meta.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(t.TempDir(), "at000.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(meta)); err != nil {
		t.Fatal(err)
	}
	// Park the seed at the frozen 000 version so Open applies only 001.
	if _, err := db.Exec(`UPDATE schema SET version = ?, digest = ?`, frozen[0].Version, frozen[0].Digest); err != nil {
		t.Fatal(err)
	}
	db.Close()
	return path, frozen
}

func TestMarshalReport(t *testing.T) {
	rep := report{
		Update:   entry{"002", "dd", "002_foo.up.sql"},
		Base:     entry{"001", "d1", "001_init.up.sql"},
		Snapshot: snapshot{Digest: "abc", Mtime: "2026-06-27T00:00:00Z", Size: 1024},
		Rows:     rows{Total: 42, Affected: 7},
		Result:   "ok",
	}
	a, err := marshalReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	b, err := marshalReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("marshalReport is not deterministic")
	}
	if a[len(a)-1] != '\n' {
		t.Error("report should end with a newline")
	}
	var got report
	if err := json.Unmarshal(a, &got); err != nil {
		t.Fatalf("report does not round-trip: %v", err)
	}
	if got.Update != rep.Update || got.Result != "ok" || got.Rows.Total != 42 || got.Rows.Affected != 7 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// TestSnapshotVacuumScript runs the generated remote script under a real
// shell, with stub zfs and sqlite3 on PATH, to verify the shell logic:
// the snapshot is taken and destroyed, and VACUUM INTO reads the
// database through the snapshot's .zfs path (derived from the dataset
// mountpoint) rather than the live file.
func TestSnapshotVacuumScript(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "calls.log")
	// The zfs stub reports a fixed mountpoint for "get mountpoint" so the
	// script can derive the database's path relative to the dataset.
	writeStub(t, filepath.Join(bin, "zfs"), `echo "zfs $*" >>"`+log+`"
[ "$1" = get ] && echo /database
exit 0`)
	writeStub(t, filepath.Join(bin, "sqlite3"), `echo "sqlite3 $*" >>"`+log+`"`)

	script := snapshotVacuumScript("tank/act3", "tank/act3@act3-freeze-1", "/database/act3.db", "/tmp/out.db")
	cmd := exec.Command("sh", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("script failed: %v\n%s", err, out)
	}

	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{
		"zfs snapshot tank/act3@act3-freeze-1",
		"zfs get -H -o value mountpoint tank/act3",
		"sqlite3 /database/.zfs/snapshot/act3-freeze-1/act3.db VACUUM INTO '/tmp/out.db'",
		"zfs destroy tank/act3@act3-freeze-1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("script calls missing %q; got:\n%s", want, got)
		}
	}
}

// writeStub writes an executable /bin/sh stub script at path.
func writeStub(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
