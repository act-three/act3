package model

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"kr.dev/errorfmt"

	"ily.dev/act3/storage"
)

// blobGCGrace is the minimum age of an unreferenced blob before
// gcBlobsOnce removes it. Every blob-creation flow records its key
// in the database within at most one ffmpeg probe or subtitle
// extraction pass of the blob acquiring its keyed name, so a full
// day is a comfortably conservative bound.
const blobGCGrace = 24 * time.Hour

// liveStorageKeys returns every storage key referenced anywhere in
// the database. Rather than enumerating the columns known to hold
// keys, it scans every text and blob value of every table for
// key-shaped substrings, so a future table holding keys can't be
// forgotten as a GC root. A false positive merely retains garbage.
func (tx *TxR) liveStorageKeys() (map[string]bool, error) {
	rows, err := tx.tx.QueryContext(tx.ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	live := map[string]bool{}
	for _, table := range tables {
		if err := tx.scanTableKeys(table, live); err != nil {
			return nil, err
		}
	}
	return live, nil
}

// scanTableKeys adds to live every key-shaped substring found in
// table's text and blob values.
func (tx *TxR) scanTableKeys(table string, live map[string]bool) (err error) {
	defer errorfmt.Handlef("scan %s: %w", table, &err)
	rows, err := tx.tx.QueryContext(tx.ctx,
		`SELECT * FROM "`+strings.ReplaceAll(table, `"`, `""`)+`"`)
	if err != nil {
		return err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		for _, v := range vals {
			var s string
			switch v := v.(type) {
			case string:
				s = v
			case []byte:
				s = string(v)
			default:
				continue
			}
			for key := range storage.ScanKeys(s) {
				live[key] = true
			}
		}
	}
	return rows.Err()
}

// gcBlobsOnce removes every blob whose key appears nowhere in the
// database and whose file is older than cutoff.
func (m *Model) gcBlobsOnce(ctx context.Context, cutoff time.Time) (err error) {
	defer errorfmt.Handlef("blob gc: %w", &err)
	var live map[string]bool
	err = m.WithTxR(ctx, func(tx *TxR) (err error) {
		live, err = tx.liveStorageKeys()
		return err
	})
	if err != nil {
		return err
	}
	// Placeholder images are inserted at boot, so an empty result
	// means the scan is broken, not that no blob is live.
	if len(live) == 0 {
		return errors.New("no live keys found")
	}
	removed, err := m.store.Sweep(cutoff, func(key string) bool { return live[key] })
	if removed > 0 {
		slog.InfoContext(ctx, "blob gc", "removed", removed)
	}
	return err
}

func (m *Model) gcBlobsLoop() {
	for {
		time.Sleep(24 * time.Hour)
		cutoff := time.Now().Add(-blobGCGrace)
		if err := m.gcBlobsOnce(context.Background(), cutoff); err != nil {
			slog.Error("blob gc", "error", err)
		}
	}
}
