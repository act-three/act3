package model

import (
	"context"
	"errors"
	"testing"

	"ily.dev/act3/database/schema"
)

// attachEpisodeVideo links videoID to episodeID via EpisodeVideoCreate.
func attachEpisodeVideo(t *testing.T, m *Model, episodeID, videoID string) {
	t.Helper()
	ctx := context.Background()
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		_, err := tx.q.EpisodeVideoCreate(schema.EpisodeVideoCreateParams{
			EpisodeID: episodeID, VideoID: videoID,
		})

		return err
	}); err != nil {
		t.Fatalf("attach %s→%s: %v", videoID, episodeID, err)
	}
}

// makeVideoPlayable sets Playable=1 on videoID and runs the
// active-video promotion that ingest's recomputePlayable would run.
func makeVideoPlayable(t *testing.T, m *Model, videoID string) {
	t.Helper()
	ctx := context.Background()
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		if _, err := tx.q.VideoUpdatePlayable(schema.VideoUpdatePlayableParams{
			ID: videoID, Playable: 1,
		}); err != nil {
			return err
		}
		return tx.ensureActiveVideoForVideoID(videoID)
	}); err != nil {
		t.Fatalf("make %s playable: %v", videoID, err)
	}
}

// setVideoDuration sets the probed duration (in milliseconds) on a
// video, as ffmpeg ingest would.
func setVideoDuration(t *testing.T, m *Model, videoID string, ms int64) {
	t.Helper()
	ctx := context.Background()
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.q.VideoUpdateProbe(schema.VideoUpdateProbeParams{
			Duration: ms, ID: videoID,
		})
	}); err != nil {
		t.Fatalf("set duration on %s: %v", videoID, err)
	}
}

func episodeRuntime(t *testing.T, m *Model, episodeID string) int64 {
	t.Helper()
	ctx := context.Background()
	var runtime int64
	if err := m.WithTxR(ctx, func(tx *TxR) error {
		ep, err := tx.q.EpisodeGet(episodeID)
		if err != nil {
			return err
		}
		runtime = ep.Runtime
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return runtime
}

func episodeVideoActive(t *testing.T, m *Model, episodeID, videoID string) bool {
	t.Helper()
	ctx := context.Background()
	var active bool
	err := m.WithTxR(ctx, func(tx *TxR) error {
		evs, err := tx.q.EpisodeVideoListByEpisodeID(episodeID)
		if err != nil {
			return err
		}
		for _, ev := range evs {
			if ev.VideoID == videoID {
				active = ev.Active != 0
				return nil
			}
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	return active
}

// TestActiveVideoFirstPlayablePromotes verifies that the first video to
// become playable on an episode is auto-promoted to Active, while a
// non-playable attached video isn't.
func TestActiveVideoFirstPlayablePromotes(t *testing.T) {
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)

	if episodeVideoActive(t, m, fx.episodeID, v1) {
		t.Error("v1 should not be Active before becoming playable")
	}

	makeVideoPlayable(t, m, v1)
	if !episodeVideoActive(t, m, fx.episodeID, v1) {
		t.Error("v1 should be Active after becoming playable")
	}
	if episodeVideoActive(t, m, fx.episodeID, v2) {
		t.Error("v2 should not be Active; v1 already claims it")
	}

	// v2 becoming playable later must not steal Active from v1.
	makeVideoPlayable(t, m, v2)
	if !episodeVideoActive(t, m, fx.episodeID, v1) {
		t.Error("v1 should remain Active after v2 also becomes playable")
	}
	if episodeVideoActive(t, m, fx.episodeID, v2) {
		t.Error("v2 should not become Active when v1 is already Active")
	}
}

// TestActiveVideoPromoteSyncsRuntime verifies that auto-promoting the
// first playable video overrides the episode's TVmaze runtime with the
// video's duration, rounded to whole minutes.
func TestActiveVideoPromoteSyncsRuntime(t *testing.T) {
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	// The fixture seeds the episode with a TVmaze runtime of 30 min.
	if got := episodeRuntime(t, m, fx.episodeID); got != 30 {
		t.Fatalf("seeded runtime = %d, want 30", got)
	}

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	setVideoDuration(t, m, v1, 44*60_000+20_000) // 44m20s → 44
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	makeVideoPlayable(t, m, v1)

	if got := episodeRuntime(t, m, fx.episodeID); got != 44 {
		t.Errorf("runtime after promote = %d, want 44", got)
	}
}

// TestEpisodeVideoSetActiveSyncsRuntime verifies that switching the
// active video updates the episode runtime to the new video's duration.
func TestEpisodeVideoSetActiveSyncsRuntime(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "", nil)
	setVideoDuration(t, m, v1, 44*60_000)        // 44m → 44
	setVideoDuration(t, m, v2, 51*60_000+40_000) // 51m40s → 52
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)
	makeVideoPlayable(t, m, v1)
	makeVideoPlayable(t, m, v2)

	if got := episodeRuntime(t, m, fx.episodeID); got != 44 {
		t.Fatalf("runtime after v1 active = %d, want 44", got)
	}

	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.EpisodeVideoSetActive(fx.episodeID, v2)
	}); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if got := episodeRuntime(t, m, fx.episodeID); got != 52 {
		t.Errorf("runtime after switching to v2 = %d, want 52", got)
	}
}

// TestEpisodeVideoSetActiveSwitches verifies the admin can switch the
// active video to another playable one.
func TestEpisodeVideoSetActiveSwitches(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)
	makeVideoPlayable(t, m, v1)
	makeVideoPlayable(t, m, v2)

	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.EpisodeVideoSetActive(fx.episodeID, v2)
	}); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if episodeVideoActive(t, m, fx.episodeID, v1) {
		t.Error("v1 should be deactivated after switch")
	}
	if !episodeVideoActive(t, m, fx.episodeID, v2) {
		t.Error("v2 should be Active after switch")
	}
}

// TestEpisodeVideoSetActiveRejectsUnplayable verifies SetActive errors
// when the target video isn't playable.
func TestEpisodeVideoSetActiveRejectsUnplayable(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)
	makeVideoPlayable(t, m, v1)
	// v2 is not playable.

	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.EpisodeVideoSetActive(fx.episodeID, v2)
	})

	if err == nil {
		t.Fatal("expected error setting unplayable video active")
	}
}

// TestActiveVideoLockedAgainstTrash verifies that a Video which is
// Active on a work with another playable video cannot be trashed.
func TestActiveVideoLockedAgainstTrash(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)
	makeVideoPlayable(t, m, v1)
	makeVideoPlayable(t, m, v2)

	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.Trash(v1)
	})

	if !errors.Is(err, ErrActiveVideoLocked) {
		t.Fatalf("expected ErrActiveVideoLocked, got %v", err)
	}
}

// TestActiveVideoSoleAllowsTrash verifies that the active video can be
// trashed when it's the only playable one on the work.
func TestActiveVideoSoleAllowsTrash(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	makeVideoPlayable(t, m, v1)

	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.Trash(v1)
	}); err != nil {
		t.Fatalf("Trash sole active: %v", err)
	}
}

// TestActiveVideoNonActiveTrashAllowed verifies that a non-Active
// playable video can be trashed even when an Active sibling exists.
func TestActiveVideoNonActiveTrashAllowed(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)
	makeVideoPlayable(t, m, v1)
	makeVideoPlayable(t, m, v2)
	// v1 became playable first → Active; v2 → not Active.

	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.Trash(v2)
	}); err != nil {
		t.Fatalf("trash non-active: %v", err)
	}
	if !episodeVideoActive(t, m, fx.episodeID, v1) {
		t.Error("v1 should remain Active after trashing non-active sibling")
	}
}

// TestActiveVideoLockedAgainstReencode verifies the same lock applies
// to ReencodeVideo when another playable video exists.
func TestActiveVideoLockedAgainstReencode(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "k1", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "k2", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)
	makeVideoPlayable(t, m, v1)
	makeVideoPlayable(t, m, v2)

	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.ReencodeVideo(v1)
	})

	if !errors.Is(err, ErrActiveVideoLocked) {
		t.Fatalf("expected ErrActiveVideoLocked, got %v", err)
	}
}

// TestActiveVideoUniqueIndex verifies that the partial unique index
// rejects a second Active=1 row on the same episode.
func TestActiveVideoUniqueIndex(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	v1 := createVideoRow(t, m, "v1.mkv", "", nil)
	v2 := createVideoRow(t, m, "v2.mkv", "", nil)
	attachEpisodeVideo(t, m, fx.episodeID, v1)
	attachEpisodeVideo(t, m, fx.episodeID, v2)
	makeVideoPlayable(t, m, v1)
	makeVideoPlayable(t, m, v2)

	// v1 is already Active. Attempting to mark v2 Active without first
	// clearing v1 must fail at the partial unique index.
	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		_, err := tx.q.EpisodeVideoMarkActive(schema.EpisodeVideoMarkActiveParams{
			EpisodeID: fx.episodeID, VideoID: v2,
		})

		return err
	})

	if err == nil {
		t.Fatal("expected unique-index violation when marking second Active without clearing first")
	}
}
