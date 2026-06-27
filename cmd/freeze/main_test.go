package main

import (
	"database/sql"
	"encoding/json/v2"
	"os"
	"path/filepath"
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

func TestParseTouches(t *testing.T) {
	tests := []struct {
		name    string
		ddl     string
		want    []string
		wantErr bool
	}{
		{"after description", "-- add slug to movie\n-- touches: movie\nALTER TABLE movie ...", []string{"movie"}, false},
		{"multiple", "-- desc\n-- touches: movie, episode\nSQL", []string{"movie", "episode"}, false},
		{"empty list", "-- desc\n-- touches:\nSQL", nil, false},
		{"missing", "-- desc\nALTER TABLE movie ...", nil, true},
		{"directive after sql is not found", "-- desc\nSQL\n-- touches: movie", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTouches([]byte(tt.ddl))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !equalStrings(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotCounts(t *testing.T) {
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

	total, per, err := snapshotCounts(path, []string{"movie", "episode"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if per["movie"] != 3 || per["episode"] != 1 {
		t.Errorf("per-table = %v", per)
	}
	if _, _, err := snapshotCounts(path, []string{"nope"}); err == nil {
		t.Error("want error for a missing touched table")
	}
}

// TestApplyAndRead exercises the real database.Open update path: it seeds a
// database at schema version 000, then confirms applyAndRead applies 001
// and reads back the rolling digest that frozen.txt records for it.
func TestApplyAndRead(t *testing.T) {
	path, frozen := seedAt000(t)
	version, digest, err := applyAndRead(path)
	if err != nil {
		t.Fatal(err)
	}
	if version != frozen[1].Version || digest != frozen[1].Digest {
		t.Fatalf("applyAndRead = (%q, %q), want (%q, %q)",
			version, digest, frozen[1].Version, frozen[1].Digest)
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
		Rows:     rows{Total: 42, Touched: map[string]int{"movie": 3, "episode": 1}},
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
	if got.Update != rep.Update || got.Result != "ok" || got.Rows.Total != 42 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
