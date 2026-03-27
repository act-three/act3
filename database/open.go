package database

import (
	"context"
	"crypto/rand"
	"crypto/sha3"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"path"
	"strings"

	"kr.dev/errorfmt"
	"modernc.org/sqlite"
	_ "modernc.org/sqlite"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
	"ily.dev/act3/encoding/base32c"
)

//go:generate sqlc generate

//go:embed ddl/*_*.up.sql
var ddl embed.FS

const (
	// Per-connection pragmas.
	// These are run on each connection when it's opened.
	pragmasPerConn = `
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -1048576;
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 5000;
	`
)

func init() {
	ctx := context.Background()
	sqlite.MustRegisterScalarFunction("newID", 0, newID)
	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, dsn string) error {
		// hr := conn.(sqlite.HookRegisterer)
		_, err := conn.ExecContext(ctx, pragmasPerConn, nil)
		if err != nil {
			return err
		}
		return nil
	})
}

// SchemaMismatchError is returned when the stored schema digest
// does not match the expected digest for a given version.
type SchemaMismatchError struct {
	Version        string
	StoredDigest   string
	ExpectedDigest string
	DBPath         string
}

func (e *SchemaMismatchError) Error() string {
	return fmt.Sprintf("schema %s digest mismatch: stored %s != expected %s (db %s)",
		e.Version, e.StoredDigest, e.ExpectedDigest, e.DBPath,
	)
}

func Open(name string) (dbr, dbw *sql.DB, err error) {
	wname := name + "?_txlock=immediate"
	rname := name + "?mode=ro"
	if name == ":memory:" {
		s := rand.Text()[:8]
		wname = "file:" + s + "?mode=memory&cache=shared&_txlock=immediate"
		rname = "file:" + s + "?mode=memory&cache=shared"
	}

	defer errorfmt.Handlef("open db: %w", &err)

	slog.Info("open", slog.Group("db", "name", name))

	dbw, err = sql.Open("sqlite", wname)
	if err != nil {
		return nil, nil, err
	}
	dbw.SetMaxOpenConns(1)
	err = updateSchema(dbw)
	if err != nil {
		var sme *SchemaMismatchError
		if errors.As(err, &sme) {
			sme.DBPath = name
			dbw.Close()
			return nil, nil, sme
		}
		return nil, nil, err
	}

	dbr, err = sql.Open("sqlite", rname)
	if err != nil {
		return nil, nil, err
	}

	return dbr, dbw, nil
}

func updateSchema(db *sql.DB) (err error) {
	defer errorfmt.Handlef("update schema: %w", &err)
	log := slog.Default().WithGroup("schema")
	q := schema.New(db)
	cur, _ := q.SchemaVersionGet(context.Background())
	if cur.Version == "" {
		cur.Version = "###"
	}
	if cur.Digest == "" {
		cur.Digest = "00000000"
	}
	log.Info("loaded", "version", cur.Version, "digest", cur.Digest)
	ent, err := fs.ReadDir(ddl, "ddl")
	if err != nil {
		return err
	}
	computedDigest := ""
	for _, d := range ent {
		if d.IsDir() {
			panic("directory in ddl: " + d.Name())
		}
		version, _, _ := strings.Cut(d.Name(), "_")
		b, err := fs.ReadFile(ddl, path.Join("ddl", d.Name()))
		if err != nil {
			panic("cannot read ddl: " + d.Name())
		}
		update := string(b)
		computedDigest = hash(computedDigest, update)
		if cur.Version == version && cur.Digest != computedDigest {
			return &SchemaMismatchError{
				Version:        version,
				StoredDigest:   cur.Digest,
				ExpectedDigest: computedDigest,
			}
		}
		if cur.Version >= version {
			continue
		}
		err = applySchemaUpdate(log, db, d.Name(), version, computedDigest, update)
		if err != nil {
			return err
		}
		cur.Version = version
		cur.Digest = computedDigest
	}
	return nil
}

func applySchemaUpdate(log *slog.Logger, db *sql.DB, name, version, digest, update string) error {
	log.With("version", version, "digest", digest)
	log = log.WithGroup("update")
	desc, _, _ := strings.Cut(update, "\n")
	log.Info("apply",
		"name", name,
		"desc", strings.TrimPrefix(desc, "-- "),
	)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(update)
	if err != nil {
		return err
	}
	q := schema.New(tx)
	err = q.SchemaVersionSet(context.Background(), schema.SchemaVersionSetParams{
		Version: version,
		Digest:  digest,
	})
	if err != nil {
		return err
	}
	return tx.Commit()
}

func newID(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	return flurry.NewID(), nil
}

func hash(prevDigest, query string) string {
	if len(prevDigest) > math.MaxUint8 {
		panic("prev digest too long")
	}
	h := sha3.New256()
	h.Write([]uint8{uint8(len(prevDigest))})
	io.WriteString(h, prevDigest)
	io.WriteString(h, query)
	sum := h.Sum(nil)
	return strings.ToLower(base32c.EncodeToString(sum)[:16])
}
