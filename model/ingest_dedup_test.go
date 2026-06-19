package model

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"lukechampine.com/blake3"

	"ily.dev/act3/database/schema"
)

// TestMergeDuplicateVideoMovesJunctions exercises the merge helper
// directly: two Videos with identical ContentHash, both attached to
// one episode (via separate junctions). After merge, the loser's row
// is gone, its junctions are folded into the winner's, and the
// winner's storage key is untouched.
func TestMergeDuplicateVideoMovesJunctions(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, _, sedID, snID, epWinID, vidWinnerID := createSeriesRow(t, m, "merge-helper", "Helper")
	// Second episode in the same edition so the loser has a distinct
	// junction the reassign must preserve.
	var epSecondID string
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		ep, err := tx.q.EpisodeCreate(schema.EpisodeCreateParams{
			Title: "E2", Type: "regular", Runtime: 30,
		})

		if err != nil {
			return err
		}
		epSecondID = ep.ID
		return tx.q.SeasonEpisodeCreate(schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: snID,
			EpisodeID: epSecondID, SortKey: 2, Label: "2", Number: 2, Slug: "s1e2-e2",
		})

	}); err != nil {
		t.Fatal(err)
	}

	// Winner's stored bytes + populated metadata. The loser's bytes
	// will be in the store under a separate key.
	fileBytes := []byte("merge helper content")
	winnerKey, err := m.store.Copy(bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatal(err)
	}
	loserKey, err := m.store.Copy(bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatal(err)
	}
	h := blake3.New(32, nil)
	h.Write(fileBytes)
	contentHash := h.Sum(nil)

	// Two Downloads so Video.InfoHash FKs are satisfied.
	infoHash1 := fortyCharHex(1)
	infoHash2 := fortyCharHex(2)
	createTrashableDownload(t, m, infoHash1, "downloaded", sedID)
	createTrashableDownload(t, m, infoHash2, "downloaded", sedID)
	attachVideoToDownload(t, m, vidWinnerID, infoHash1)

	// Set winner's OriginalKey + ContentHash.
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		_, err := tx.q.VideoUpdateOriginalKey(schema.VideoUpdateOriginalKeyParams{
			ID: vidWinnerID, OriginalKey: winnerKey, ContentHash: contentHash,
		})

		return err
	}); err != nil {
		t.Fatal(err)
	}

	// Create the loser Video and attach it to the second episode.
	// It also gets pinned to Download #2 so bumpDownloadActivity has
	// something to update.
	var vidLoserID string
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		vid, err := tx.q.VideoCreate(schema.VideoCreateParams{
			InfoHash: &infoHash2, Name: "pilot.mkv",
		})

		if err != nil {
			return err
		}
		vidLoserID = vid.ID
		_, err = tx.q.EpisodeVideoCreate(schema.EpisodeVideoCreateParams{
			EpisodeID: epSecondID, VideoID: vidLoserID,
		})

		return err
	}); err != nil {
		t.Fatal(err)
	}

	// Drive the merge.
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		loser, err := tx.q.VideoGet(vidLoserID)
		if err != nil {
			return err
		}
		winner, err := tx.q.VideoGet(vidWinnerID)
		if err != nil {
			return err
		}
		return tx.mergeDuplicateVideo(loser, winner, loserKey)
	}); err != nil {
		t.Fatal(err)
	}

	// Loser is gone; winner's junctions now cover both episodes.
	if err := m.WithTxR(ctx, func(tx *TxR) error {
		_, err := tx.q.VideoGet(vidLoserID)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("loser Video should be gone: err=%v", err)
		}
		winner, err := tx.q.VideoGet(vidWinnerID)
		if err != nil {
			return err
		}
		if winner.OriginalKey != winnerKey {
			t.Errorf("winner OriginalKey changed: got %q, want %q", winner.OriginalKey, winnerKey)
		}
		evs, err := tx.q.EpisodeVideoListByVideoID(vidWinnerID)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, ev := range evs {
			seen[ev.EpisodeID] = true
		}
		if !seen[epWinID] || !seen[epSecondID] {
			t.Errorf("winner junctions = %+v, want both %s and %s", evs, epWinID, epSecondID)
		}
		loserEvs, err := tx.q.EpisodeVideoListByVideoID(vidLoserID)
		if err != nil {
			return err
		}
		if len(loserEvs) != 0 {
			t.Errorf("loser junctions = %+v, want none", loserEvs)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// The loser's bytes are scheduled for removal via onCommit after
	// the WithTxRW above completes. Give the store a direct check.
	if _, err := m.store.Open(loserKey); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("loser storage key should be removed after commit: err=%v", err)
	}
	if _, err := m.store.Open(winnerKey); err != nil {
		t.Errorf("winner storage key unexpectedly gone: %v", err)
	}
}

// TestMergeDuplicateVideoRestoresSoftDeletedJunction covers the edge
// case where the winner already had a soft-deleted junction for the
// same episode the loser is live-attached to. The merge should
// resurrect the winner's junction rather than leaving the episode
// detached.
func TestMergeDuplicateVideoRestoresSoftDeletedJunction(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, _, sedID, _, epID, vidWinnerID := createSeriesRow(t, m, "restore-junction", "Restore")

	infoHash1 := fortyCharHex(1)
	infoHash2 := fortyCharHex(2)
	createTrashableDownload(t, m, infoHash1, "downloaded", sedID)
	createTrashableDownload(t, m, infoHash2, "downloaded", sedID)
	attachVideoToDownload(t, m, vidWinnerID, infoHash1)

	// Soft-delete the winner's existing junction to epID (simulating a
	// user detach) and pin its hash.
	fileBytes := []byte("restore-junction-test")
	winnerKey, err := m.store.Copy(bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatal(err)
	}
	h := blake3.New(32, nil)
	h.Write(fileBytes)
	contentHash := h.Sum(nil)
	deletedAt := int64(1)

	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		if _, err := tx.q.VideoUpdateOriginalKey(schema.VideoUpdateOriginalKeyParams{
			ID: vidWinnerID, OriginalKey: winnerKey, ContentHash: contentHash,
		}); err != nil {
			return err
		}
		return tx.q.EpisodeVideoSoftDelete(schema.EpisodeVideoSoftDeleteParams{
			EpisodeID: epID, VideoID: vidWinnerID, DeletedAt: &deletedAt,
		})

	}); err != nil {
		t.Fatal(err)
	}

	// Create loser Video with a LIVE junction to the same episode.
	var vidLoserID string
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		vid, err := tx.q.VideoCreate(schema.VideoCreateParams{
			InfoHash: &infoHash2, Name: "pilot.mkv",
		})

		if err != nil {
			return err
		}
		vidLoserID = vid.ID
		_, err = tx.q.EpisodeVideoCreate(schema.EpisodeVideoCreateParams{
			EpisodeID: epID, VideoID: vidLoserID,
		})

		return err
	}); err != nil {
		t.Fatal(err)
	}

	loserKey, err := m.store.Copy(bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		loser, err := tx.q.VideoGet(vidLoserID)
		if err != nil {
			return err
		}
		winner, err := tx.q.VideoGet(vidWinnerID)
		if err != nil {
			return err
		}
		return tx.mergeDuplicateVideo(loser, winner, loserKey)
	}); err != nil {
		t.Fatal(err)
	}

	// The winner's junction is back to live (DeletedAt cleared); the
	// episode has a live attachment again.
	if err := m.WithTxR(ctx, func(tx *TxR) error {
		evs, err := tx.q.EpisodeVideoListByVideoID(vidWinnerID)
		if err != nil {
			return err
		}
		if len(evs) != 1 {
			t.Fatalf("winner junctions = %+v, want exactly one", evs)
		}
		if evs[0].EpisodeID != epID {
			t.Errorf("winner junction episode = %s, want %s", evs[0].EpisodeID, epID)
		}
		if evs[0].DeletedAt != nil {
			t.Errorf("winner junction still soft-deleted: DeletedAt=%v", *evs[0].DeletedAt)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestVideoListByContentHashFiltersDeleted verifies the detection
// query ignores soft-deleted Videos, so a past-trashed duplicate
// doesn't trigger a spurious merge.
func TestVideoListByContentHashFiltersDeleted(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, _, _, _, _, vidID := createSeriesRow(t, m, "by-hash", "ByHash")
	h := blake3.New(32, nil)
	h.Write([]byte("some bytes"))
	sum := h.Sum(nil)

	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		_, err := tx.q.VideoUpdateOriginalKey(schema.VideoUpdateOriginalKeyParams{
			ID: vidID, OriginalKey: "key", ContentHash: sum,
		})

		return err
	}); err != nil {
		t.Fatal(err)
	}

	// Live video is returned.
	if err := m.WithTxR(ctx, func(tx *TxR) error {
		got, err := tx.q.VideoListByContentHash(sum)
		if err != nil {
			return err
		}
		if len(got) != 1 || got[0].ID != vidID {
			t.Errorf("live VideoListByContentHash = %v, want one with %s", got, vidID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// After soft-delete, the query excludes it.
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.q.VideoSoftDelete(schema.VideoSoftDeleteParams{
			ID: vidID, DeletedAt: &[]int64{1}[0],
		})

	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxR(ctx, func(tx *TxR) error {
		got, err := tx.q.VideoListByContentHash(sum)
		if err != nil {
			return err
		}
		if len(got) != 0 {
			t.Errorf("soft-deleted VideoListByContentHash = %v, want empty", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestCopyToStoreHashedRoundTrip confirms that copyToStoreHashed
// returns a key that opens back to the original bytes and a matching
// blake3 digest.
func TestCopyToStoreHashedRoundTrip(t *testing.T) {
	m := newTestModel(t)

	content := bytes.Repeat([]byte("abc"), 10_000)
	tmp := filepath.Join(t.TempDir(), "vid.bin")
	if err := os.WriteFile(tmp, content, 0644); err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	key, sum, err := m.copyToStoreHashed(src)
	if err != nil {
		t.Fatal(err)
	}

	h := blake3.New(32, nil)
	h.Write(content)
	if want := h.Sum(nil); !bytes.Equal(sum, want) {
		t.Errorf("hash mismatch: got %x want %x", sum, want)
	}

	f, err := m.store.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var roundtrip bytes.Buffer
	if _, err := roundtrip.ReadFrom(f); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(roundtrip.Bytes(), content) {
		t.Errorf("stored bytes differ from source (len got=%d want=%d)",
			roundtrip.Len(), len(content))
	}
}
