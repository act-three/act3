package model

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"ily.dev/act3/database/schema"
)

// TestGCBlobsOnce exercises the conservative mark-sweep: a blob
// referenced by a key column survives, a blob whose key appears only
// inside a larger text value survives, and an unreferenced blob is
// removed. The cutoff is set in the future so file age never
// protects a blob; age handling is covered by the storage tests.
func TestGCBlobsOnce(t *testing.T) {
	ctx := t.Context()
	m := newTestModel(t)

	blob := func() string {
		key, err := m.store.Copy(strings.NewReader("blob"))
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	refKey, embeddedKey, orphanKey := blob(), blob(), blob()

	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		if _, err := tx.q.ImageCreate(schema.ImageCreateParams{
			OriginalKey: refKey,
			Type:        "image/png",
		}); err != nil {
			return err
		}
		return tx.q.SettingSet(schema.SettingSetParams{
			Key:   "gc-test",
			Group: "test",
			Value: `{"note":"see blob ` + embeddedKey + ` for details"}`,
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := m.gcBlobsOnce(ctx, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{refKey, embeddedKey} {
		f, err := m.store.Open(key)
		if err != nil {
			t.Errorf("live blob %s: %v", key, err)
			continue
		}
		f.Close()
	}
	if _, err := m.store.Open(orphanKey); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("orphan blob: err = %v, want fs.ErrNotExist", err)
	}
}
