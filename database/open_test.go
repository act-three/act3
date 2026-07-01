// The seed helper walks readUpdates, whose prod backstop refuses the
// unfrozen updates these tests exist to exercise.
//go:build !prod

package database

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

// seedAt001 builds a database parked at frozen update 001 and runs
// the given statements on it, so Open applies 002 next.
func seedAt001(t *testing.T, stmts []string) string {
	t.Helper()
	updates, err := readUpdates()
	if err != nil {
		t.Fatal(err)
	}
	if updates[1].version != "001" {
		t.Fatalf("updates[1] = %s, want 001", updates[1].name)
	}
	path := filepath.Join(t.TempDir(), "at001.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, u := range updates[:2] {
		if _, err := db.Exec(string(u.ddl)); err != nil {
			t.Fatalf("apply %s: %v", u.name, err)
		}
	}
	if _, err := db.Exec(`UPDATE schema SET version = ?, digest = ?`,
		updates[1].version, updates[1].digest); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed: %v\nstatement: %s", err, stmt)
		}
	}
	return path
}

// TestTrashKindBackfill covers update 002: rows trashed before the
// Kind column existed get their kind from the entity tables, cascade
// children and info-hash IDs included.
func TestTrashKindBackfill(t *testing.T) {
	infoHash := strings.Repeat("ab", 20)
	path := seedAt001(t, []string{
		`INSERT INTO Image (ID, OriginalKey, Type) VALUES
			('iplaceholderposter', 'kposter', 'image/png'),
			('iplaceholderbanner', 'kbanner', 'image/png')`,
		`INSERT INTO Movie (ID, Slug, DeletedAt) VALUES ('movA', 'mov-a', 1000)`,
		`INSERT INTO MovieEdition (ID, MovieID, Slug, Title, Label, Summary, ReleaseDate, Runtime, DeletedAt)
			VALUES ('medA', 'movA', '', 'Movie A', 'Theatrical', '', '', 90, 1000)`,
		`INSERT INTO Download (InfoHash, State, Title, Torrent, MovieEditionID, DeletedAt)
			VALUES ('` + infoHash + `', 'downloaded', 'Movie A', x'00', 'medA', 1000)`,
		`INSERT INTO Collection (ID, Slug, Title, DeletedAt)
			VALUES ('colA', 'col-a', 'Faves', 1000)`,
		`INSERT INTO Trash (ID, Title, Subtitle, DeletedAt, CascadeOf) VALUES
			('movA', 'Movie A', '', 1000, NULL),
			('medA', 'Movie A · Theatrical', '', 1000, 'movA'),
			('` + infoHash + `', 'Movie A', '', 1000, 'movA'),
			('colA', 'Faves', '', 1000, NULL)`,
	})

	dbr, dbw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dbw.Close()
	defer dbr.Close()

	want := map[string]string{
		"movA":   "Movie",
		"medA":   "MovieEdition",
		infoHash: "Download",
		"colA":   "Collection",
	}
	rows, err := dbr.Query(`SELECT ID, Kind FROM Trash`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := make(map[string]string)
	for rows.Next() {
		var id, kind string
		if err := rows.Scan(&id, &kind); err != nil {
			t.Fatal(err)
		}
		got[id] = kind
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Errorf("backfilled %d rows, want %d", len(got), len(want))
	}
	for id, kind := range want {
		if got[id] != kind {
			t.Errorf("Trash[%s].Kind = %q, want %q", id, got[id], kind)
		}
	}
}

// A trash row whose ID matches no entity table has no kind; the
// backfill must fail loudly rather than invent one.
func TestTrashKindBackfillUnmatched(t *testing.T) {
	path := seedAt001(t, []string{
		`INSERT INTO Trash (ID, Title, Subtitle, DeletedAt, CascadeOf)
			VALUES ('ghost', 'Ghost', '', 1000, NULL)`,
	})
	if _, _, err := Open(path); err == nil {
		t.Fatal("Open applied 002 to an unmatchable trash row without error")
	}
}
