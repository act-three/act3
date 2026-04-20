package model

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"ily.dev/act3/database/schema"
)

// createTrashableDownload inserts a Download row directly (bypassing
// torrent parsing) in the given state. Returns its InfoHash.
//
// LastActivityAt is set from Go's time.Now() (rather than SQLite's
// unixepoch() default) so tests running inside synctest see a
// LastActivityAt on the bubble clock, matching what time.Sleep
// advances and what autoTrashDownloadsOnce compares against.
func createTrashableDownload(
	t *testing.T, m *Model, infoHash, state, sedID string,
) {
	t.Helper()
	ctx := context.Background()
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.DownloadCreate(ctx, schema.DownloadCreateParams{
			InfoHash:        infoHash,
			State:           state,
			Title:           "Test Torrent",
			Torrent:         []byte("stub"),
			SeriesEditionID: &sedID,
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.dbw.ExecContext(ctx,
		"UPDATE Download SET LastActivityAt = ? WHERE InfoHash = ?",
		time.Now().UnixMilli(), infoHash,
	); err != nil {
		t.Fatal(err)
	}
}

// attachVideoToDownload sets Video.InfoHash to pin it to the given
// Download. Video must already exist.
func attachVideoToDownload(t *testing.T, m *Model, vidID, infoHash string) {
	t.Helper()
	ctx := context.Background()
	if _, err := m.dbw.ExecContext(ctx, "UPDATE Video SET InfoHash = ? WHERE ID = ?", infoHash, vidID); err != nil {
		t.Fatal(err)
	}
}

// fortyCharHex is a valid SHA-1-shaped info hash for tests.
func fortyCharHex(seed byte) string {
	var b [40]byte
	for i := range b {
		b[i] = "0123456789abcdef"[(int(seed)+i)%16]
	}
	return string(b[:])
}

func TestKindOfInfoHash(t *testing.T) {
	if got := kindOf(fortyCharHex(0)); got != TrashKindDownload {
		t.Errorf("kindOf(40-char hex) = %v, want TrashKindDownload", got)
	}
	// Wrong length.
	if got := kindOf("abcdef1234"); got == TrashKindDownload {
		t.Errorf("kindOf(10-char hex) = %v; should not be TrashKindDownload", got)
	}
	// Right length, non-hex.
	notHex := strings.Repeat("z", 40)
	if got := kindOf(notHex); got == TrashKindDownload {
		t.Errorf("kindOf(40 z's) = %v; should not be TrashKindDownload", got)
	}
	// Flurry prefixes still win.
	if got := kindOf("mo" + strings.Repeat("a", 38)); got != TrashKindMovie {
		t.Errorf("kindOf(mo... 40 chars) = %v; want TrashKindMovie", got)
	}
}

func TestDownloadTrashAnyState(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	for _, state := range []string{"queued", "downloading", "downloaded", "imported", "error"} {
		_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-"+state, state)
		infoHash := fortyCharHex(byte(len(state)))
		createTrashableDownload(t, m, infoHash, state, sedID)
		if err := m.WithTxRW(func(tx *TxRW) error { return tx.Trash(ctx, infoHash) }); err != nil {
			t.Errorf("Trash(%s) state=%s: %v", infoHash, state, err)
		}
	}
}

func TestDownloadTrashReapsOrphanVideos(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-orphan", "Orphan")
	infoHash := fortyCharHex(1)
	createTrashableDownload(t, m, infoHash, "imported", sedID)
	vidID := createVideoRow(t, m, "orphan.mkv", "", nil)
	attachVideoToDownload(t, m, vidID, infoHash)

	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Trash(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}
	// Video had no EV/MV junctions; its only pin was the Download. After
	// trash, the Download-pin is gone, so the Video is orphan-reaped.
	if videoLive(t, m, vidID) {
		t.Errorf("video should be trashed as orphan of download cascade")
	}
}

func TestDownloadTrashKeepsJunctionedVideos(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	srID, _, sedID, _, epID, vidID := createSeriesRow(t, m, "dl-shared", "Shared")
	_ = srID
	_ = epID
	infoHash := fortyCharHex(2)
	createTrashableDownload(t, m, infoHash, "imported", sedID)
	attachVideoToDownload(t, m, vidID, infoHash)
	// Video is already EV-linked to the episode via createSeriesRow.

	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Trash(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}
	if !videoLive(t, m, vidID) {
		t.Errorf("video should stay live (pinned by episode)")
	}
}

func TestDownloadRestoreRehydratesOrphanVideos(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-restore", "Restore")
	infoHash := fortyCharHex(3)
	createTrashableDownload(t, m, infoHash, "imported", sedID)
	vidID := createVideoRow(t, m, "restore.mkv", "", nil)
	attachVideoToDownload(t, m, vidID, infoHash)

	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Trash(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}
	if videoLive(t, m, vidID) {
		t.Fatal("precondition: video should be trashed")
	}
	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Restore(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}
	if !videoLive(t, m, vidID) {
		t.Errorf("video should be restored with download")
	}
}

func TestDownloadPurgeNullsSurvivingVideoInfoHash(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, _, vidID := createSeriesRow(t, m, "dl-purge", "Purge")
	infoHash := fortyCharHex(4)
	createTrashableDownload(t, m, infoHash, "imported", sedID)
	attachVideoToDownload(t, m, vidID, infoHash)

	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Trash(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Purge(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxR(func(tx *TxR) error {
		v, err := tx.q.VideoGet(ctx, vidID)
		if err != nil {
			return err
		}
		if v.InfoHash != nil {
			t.Errorf("video InfoHash should be NULL after download purge; got %q", *v.InfoHash)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadCreateOnTrashedRestores(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-readd", "Readd")
	infoHash := fortyCharHex(5)
	createTrashableDownload(t, m, infoHash, "imported", sedID)

	// Trash the Download.
	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Trash(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}

	// Simulate re-add: DownloadCreate finds the trashed row and restores
	// it rather than creating a duplicate.
	if err := m.WithTxRW(func(tx *TxRW) error {
		dl, err := tx.q.DownloadGet(ctx, infoHash)
		if err != nil {
			return err
		}
		if dl.DeletedAt == nil {
			return errors.New("precondition: download should be trashed")
		}
		if err := tx.Restore(ctx, infoHash); err != nil {
			return err
		}
		dl, err = tx.q.DownloadGet(ctx, infoHash)
		if err != nil {
			return err
		}
		if dl.DeletedAt != nil {
			t.Errorf("download should be live after restore; DeletedAt=%v", *dl.DeletedAt)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadUpdateProgressBumpsActivity(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-bump", "Bump")
	infoHash := fortyCharHex(6)
	createTrashableDownload(t, m, infoHash, "queued", sedID)

	var before int64
	if err := m.WithTxR(func(tx *TxR) error {
		dl, err := tx.q.DownloadGet(ctx, infoHash)
		before = dl.LastActivityAt
		return err
	}); err != nil {
		t.Fatal(err)
	}

	// Backdate LastActivityAt so we can observe an increase even on fast hardware.
	if _, err := m.dbw.ExecContext(ctx, "UPDATE Download SET LastActivityAt = ? WHERE InfoHash = ?", before-10_000, infoHash); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.DownloadUpdateProgress(ctx, schema.DownloadUpdateProgressParams{
			State:          "downloading",
			Progress:       0.5,
			LastActivityAt: time.Now().UnixMilli(),
			InfoHash:       infoHash,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxR(func(tx *TxR) error {
		dl, err := tx.q.DownloadGet(ctx, infoHash)
		if err != nil {
			return err
		}
		if dl.LastActivityAt <= before-10_000 {
			t.Errorf("LastActivityAt not bumped; got %d, want > %d", dl.LastActivityAt, before-10_000)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEpisodeVideoSetBumpsDownloadActivity(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, epID, vidID := createSeriesRow(t, m, "ev-bump", "EVBump")
	infoHash := fortyCharHex(11)
	createTrashableDownload(t, m, infoHash, "imported", sedID)
	// Seed the Video so VideoGetByName finds it under this Download.
	if _, err := m.dbw.ExecContext(ctx,
		"UPDATE Video SET InfoHash = ?, Name = ? WHERE ID = ?",
		infoHash, "ev-bump.mkv", vidID,
	); err != nil {
		t.Fatal(err)
	}
	// Backdate the Download's LastActivityAt so the bump is observable.
	stale := time.Now().Add(-2 * time.Hour).UnixMilli()
	if _, err := m.dbw.ExecContext(ctx,
		"UPDATE Download SET LastActivityAt = ? WHERE InfoHash = ?",
		stale, infoHash,
	); err != nil {
		t.Fatal(err)
	}

	// Detach from the episode: should bump LastActivityAt.
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.EpisodeVideoSet(ctx, infoHash, "ev-bump.mkv", epID, false)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxR(func(tx *TxR) error {
		dl, err := tx.q.DownloadGet(ctx, infoHash)
		if err != nil {
			return err
		}
		if dl.LastActivityAt <= stale {
			t.Errorf("LastActivityAt not bumped after EV detach; got %d, want > %d", dl.LastActivityAt, stale)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadRestoreBumpsActivity(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-r-bump", "RestoreBump")
	infoHash := fortyCharHex(7)
	createTrashableDownload(t, m, infoHash, "imported", sedID)

	// Trash; then backdate the still-live-column to simulate a stale
	// restore candidate.
	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Trash(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
	if _, err := m.dbw.ExecContext(ctx, "UPDATE Download SET LastActivityAt = ? WHERE InfoHash = ?", stale, infoHash); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error { return tx.Restore(ctx, infoHash) }); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxR(func(tx *TxR) error {
		dl, err := tx.q.DownloadGet(ctx, infoHash)
		if err != nil {
			return err
		}
		if dl.LastActivityAt <= stale {
			t.Errorf("LastActivityAt not bumped on restore; got %d, want > %d", dl.LastActivityAt, stale)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadAutoTrashStaleTerminal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-stale", "Stale")
		infoHash := fortyCharHex(8)
		createTrashableDownload(t, m, infoHash, "imported", sedID)

		// Fast-forward past the idle timeout.
		time.Sleep(downloadIdleTimeout + time.Hour)

		if err := m.autoTrashDownloadsOnce(ctx); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxR(func(tx *TxR) error {
			dl, err := tx.q.DownloadGet(ctx, infoHash)
			if err != nil {
				return err
			}
			if dl.DeletedAt == nil {
				t.Errorf("stale terminal download should be auto-trashed")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDownloadAutoTrashSkipsActive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		_, _, sedID, _, _, _ := createSeriesRow(t, m, "dl-skip", "Skip")
		infoHash := fortyCharHex(9)
		createTrashableDownload(t, m, infoHash, "downloading", sedID)

		time.Sleep(downloadIdleTimeout + time.Hour)

		if err := m.autoTrashDownloadsOnce(ctx); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxR(func(tx *TxR) error {
			dl, err := tx.q.DownloadGet(ctx, infoHash)
			if err != nil {
				return err
			}
			if dl.DeletedAt != nil {
				t.Errorf("active-state download should NOT be auto-trashed")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestVideoListOrphansExcludesDownloadPinned(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	_, _, sedID, _, _, _ := createSeriesRow(t, m, "orph-ex", "OrphExclude")
	infoHash := fortyCharHex(10)
	createTrashableDownload(t, m, infoHash, "imported", sedID)
	vidID := createVideoRow(t, m, "pinned.mkv", "", nil)
	attachVideoToDownload(t, m, vidID, infoHash)

	// Video has no EV/MV junctions but is pinned by live Download. It
	// must NOT appear in the orphan list.
	if err := m.WithTxR(func(tx *TxR) error {
		ids, err := tx.q.VideoListOrphans(ctx)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if id == vidID {
				t.Errorf("download-pinned video should not be listed as orphan")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
