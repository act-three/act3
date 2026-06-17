package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
)

// trashTestFixture builds a Series with two Editions that share a
// Season each containing the same Episode, so trashing an Edition
// can leave the Episode live (shared) or orphan-reap it (no other
// live references).
type trashTestFixture struct {
	seriesID  string
	edition1  string
	edition2  string
	season1   string
	season2   string
	episodeID string
}

func newTrashTestFixture(t *testing.T, m *Model) trashTestFixture {
	t.Helper()
	ctx := context.Background()
	var fx trashTestFixture
	err := m.WithTxRW(func(tx *TxRW) error {
		srID := "sr" + flurry.NewID()
		sr, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID:          srID,
			Slug:        "sr-" + srID,
			Title:       "Test Series",
			Status:      "Running",
			PremieredOn: "2020-01-01",
		})
		if err != nil {
			return err
		}
		fx.seriesID = sr.ID
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: sr.Slug, Kind: "series", Target: sr.ID,
		}); err != nil {
			return err
		}
		// A default edition kept alive throughout the tests so the
		// series invariant (one live default) holds even when
		// edition1/edition2 are trashed.
		if _, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: sr.ID, Summary: "Default",
		}); err != nil {
			return err
		}
		sed1, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Edition 1", Slug: "edition-1", SeriesID: sr.ID, Summary: "First",
		})
		if err != nil {
			return err
		}
		fx.edition1 = sed1.ID
		sed2, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Edition 2", Slug: "edition-2", SeriesID: sr.ID, Summary: "Second",
		})
		if err != nil {
			return err
		}
		fx.edition2 = sed2.ID
		sn1, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sed1.ID, SortKey: "01", Title: "Season 1", Number: 1,
		})
		if err != nil {
			return err
		}
		fx.season1 = sn1.ID
		sn2, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sed2.ID, SortKey: "01", Title: "Season 1", Number: 1,
		})
		if err != nil {
			return err
		}
		fx.season2 = sn2.ID
		ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "Pilot", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		fx.episodeID = ep.ID
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sed1.ID, SeasonID: sn1.ID, EpisodeID: ep.ID,
			SortKey: 1, Label: "1", Number: 1, Slug: "s1e1-pilot",
		}); err != nil {
			return err
		}
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sed2.ID, SeasonID: sn2.ID, EpisodeID: ep.ID,
			SortKey: 1, Label: "1", Number: 1, Slug: "s1e1-pilot",
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fx
}

func episodeState(t *testing.T, m *Model, id string) trashState {
	t.Helper()
	ctx := context.Background()
	var st trashState
	err := m.WithTxR(func(tx *TxR) error {
		var err error
		st, err = tx.trashState(ctx, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// TestTrashOrphanReapAcrossEditions covers Scenario A: an Episode
// shared by two Editions stays live when the first Edition is
// trashed (other Edition still references it) and is orphan-reaped
// when the second Edition is trashed. Restoring the second Edition
// un-reaps the Episode.
func TestTrashOrphanReapAcrossEditions(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, fx.edition1)
	}); err != nil {
		t.Fatalf("trash edition 1: %v", err)
	}
	if st := episodeState(t, m, fx.episodeID); !st.live() {
		t.Fatalf("episode should stay live after edition1 trashed; got cascadeOf=%v",
			st.cascadeOf)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, fx.edition2)
	}); err != nil {
		t.Fatalf("trash edition 2: %v", err)
	}
	if st := episodeState(t, m, fx.episodeID); st.live() {
		t.Fatal("episode should be reaped after edition2 trashed")
	} else if st.cascadeOf != fx.edition2 {
		t.Errorf("episode cascadeOf = %q, want %q", st.cascadeOf, fx.edition2)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, fx.edition2)
	}); err != nil {
		t.Fatalf("restore edition 2: %v", err)
	}
	if st := episodeState(t, m, fx.episodeID); !st.live() {
		t.Fatalf("episode should be live after restore; got cascadeOf=%v",
			st.cascadeOf)
	}
}

// TestTrashDirectSkippedInCascade covers Scenario B: a directly
// trashed Season is not caught by a later cascade on its Edition,
// so restoring the Edition leaves the Season trashed.
func TestTrashDirectSkippedInCascade(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, fx.season1)
	}); err != nil {
		t.Fatalf("trash season: %v", err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, fx.edition1)
	}); err != nil {
		t.Fatalf("trash edition: %v", err)
	}

	var seasonCascadeOf string
	if err := m.WithTxR(func(tx *TxR) error {
		st, err := tx.trashState(ctx, fx.season1)
		if err != nil {
			return err
		}
		seasonCascadeOf = st.cascadeOf
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if seasonCascadeOf != "" {
		t.Fatalf("directly-trashed season picked up cascadeOf = %q; should stay empty",
			seasonCascadeOf)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, fx.edition1)
	}); err != nil {
		t.Fatalf("restore edition: %v", err)
	}

	if err := m.WithTxR(func(tx *TxR) error {
		sn, err := tx.q.SeasonGet(ctx, fx.season1)
		if err != nil {
			return err
		}
		if sn.DeletedAt == nil {
			t.Error("restored edition brought back directly-trashed season; want still trashed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashAlreadyTrashedReturnsError covers Scenario C: the second
// Trash call on the same target must surface ErrAlreadyTrashed.
func TestTrashAlreadyTrashedReturnsError(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	var movieID string
	if err := m.WithTxRW(func(tx *TxRW) error {
		moID := "mo" + flurry.NewID()
		mo, err := tx.q.MovieCreate(ctx, schema.MovieCreateParams{
			ID: moID, Slug: "mo-" + moID,
		})
		if err != nil {
			return err
		}
		movieID = mo.ID
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: mo.Slug, Kind: "movie", Target: mo.ID,
		}); err != nil {
			return err
		}
		_, err = tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "Test Movie", Label: DefaultEdition, Slug: "", MovieID: mo.ID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieID)
	}); err != nil {
		t.Fatalf("first trash: %v", err)
	}

	err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieID)
	})
	if !errors.Is(err, ErrAlreadyTrashed) {
		t.Fatalf("second trash err = %v, want ErrAlreadyTrashed", err)
	}
}

// createMovieRow creates a Movie, its default MovieEdition, and a
// matching Slug row. Used by tests that need bare movie rows without
// the automatic slug-generation logic of MovieCreate.
func createMovieRow(
	t *testing.T, m *Model, slug string, tmdbID *int64,
) (movieID, medID string) {
	t.Helper()
	ctx := context.Background()
	err := m.WithTxRW(func(tx *TxRW) error {
		moID := "mo" + flurry.NewID()
		mo, err := tx.q.MovieCreate(ctx, schema.MovieCreateParams{
			ID: moID, Slug: slug, TMDBID: tmdbID,
		})
		if err != nil {
			return err
		}
		movieID = mo.ID
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: slug, Kind: "movie", Target: moID,
		}); err != nil {
			return err
		}
		med, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "Movie " + slug, Label: DefaultEdition, Slug: "",
			MovieID: moID, ReleaseDate: "2021-01-01",
		})
		if err != nil {
			return err
		}
		medID = med.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return movieID, medID
}

// createVideoRow creates a Video row with optional OriginalKey and
// Rendition keys. Returns the Video ID.
func createVideoRow(
	t *testing.T, m *Model, name string, originalKey string, rendKeys []string,
) string {
	t.Helper()
	ctx := context.Background()
	var vidID string
	err := m.WithTxRW(func(tx *TxRW) error {
		vid, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{Name: name})
		if err != nil {
			return err
		}
		vidID = vid.ID
		if originalKey != "" {
			_, err = tx.q.VideoUpdateOriginalKey(ctx, schema.VideoUpdateOriginalKeyParams{
				OriginalKey: originalKey, ID: vid.ID,
			})
			if err != nil {
				return err
			}
		}
		for i, key := range rendKeys {
			rend, err := tx.q.RenditionCreate(ctx, schema.RenditionCreateParams{
				VideoID: vid.ID, Purpose: "streaming", Codec: "h264",
				TargetBitrate: 1000, MaxHeight: 720, Priority: int64(i),
			})
			if err != nil {
				return err
			}
			_, err = tx.q.RenditionUpdateEncode(ctx, schema.RenditionUpdateEncodeParams{
				ID: rend.ID, Key: key, Playlist: "",
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return vidID
}

func videoLive(t *testing.T, m *Model, id string) bool {
	t.Helper()
	ctx := context.Background()
	var live bool
	err := m.WithTxR(func(tx *TxR) error {
		v, err := tx.q.VideoGet(ctx, id)
		if err != nil {
			return err
		}
		live = v.DeletedAt == nil
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return live
}

// TestTrashOrphanReapVideoSharedAcrossEpisodeAndMovie covers scenario
// 1: a Video referenced by both an Episode and a Movie stays live
// when the Episode is trashed, is orphan-reaped when the Movie is
// also trashed, and comes back when the Movie is restored.
func TestTrashOrphanReapVideoSharedAcrossEpisodeAndMovie(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	movieID, medID := createMovieRow(t, m, "shared-video-movie", nil)
	vidID := createVideoRow(t, m, "shared.mkv", "", nil)

	if err := m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: fx.episodeID, VideoID: vidID,
		})
		if err != nil {
			return err
		}
		_, err = tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medID, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, fx.episodeID)
	}); err != nil {
		t.Fatalf("trash episode: %v", err)
	}
	if !videoLive(t, m, vidID) {
		t.Fatal("video should stay live after trashing only the Episode; Movie still references it")
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieID)
	}); err != nil {
		t.Fatalf("trash movie: %v", err)
	}
	if videoLive(t, m, vidID) {
		t.Fatal("video should be reaped after trashing both Episode and Movie")
	}
	var vidCascadeOf string
	if err := m.WithTxR(func(tx *TxR) error {
		st, err := tx.trashState(ctx, vidID)
		if err != nil {
			return err
		}
		vidCascadeOf = st.cascadeOf
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if vidCascadeOf != movieID {
		t.Errorf("video cascadeOf = %q, want %q (reaped under movie cascade)", vidCascadeOf, movieID)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, movieID)
	}); err != nil {
		t.Fatalf("restore movie: %v", err)
	}
	if !videoLive(t, m, vidID) {
		t.Fatal("video should be live again after restoring the Movie")
	}
}

// TestTrashPurgeOrdering covers scenario 2: a Series with many
// descendant rows can be purged without FK errors and leaves zero
// rows in descendant tables.
func TestTrashPurgeOrdering(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	const numSeasons = 5
	const numEpisodesPerSeason = 12 // 60 total

	var seriesID string
	var videoIDs []string
	err := m.WithTxRW(func(tx *TxRW) error {
		srID := "sr" + flurry.NewID()
		sr, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "purge-ordering-" + srID,
			Title: "Purge Ordering Test", Status: "Running",
			PremieredOn: "2020-01-01",
		})
		if err != nil {
			return err
		}
		seriesID = sr.ID
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: sr.Slug, Kind: "series", Target: sr.ID,
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Original", Slug: "", SeriesID: sr.ID, Summary: "test",
		})
		if err != nil {
			return err
		}
		for i := range numSeasons {
			sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
				EditionID: sed.ID,
				SortKey:   fmt.Sprintf("%02d", i+1),
				Title:     fmt.Sprintf("Season %d", i+1),
				Number:    int64(i + 1),
			})
			if err != nil {
				return err
			}
			for j := range numEpisodesPerSeason {
				ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
					Title: fmt.Sprintf("Ep %d-%d", i+1, j+1),
					Type:  "regular", Runtime: 30,
				})
				if err != nil {
					return err
				}
				if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
					EditionID: sed.ID, SeasonID: sn.ID, EpisodeID: ep.ID,
					SortKey: int64(j + 1), Label: fmt.Sprintf("%d", j+1),
					Number: int64(j + 1),
					Slug:   fmt.Sprintf("s%de%d", i+1, j+1),
				}); err != nil {
					return err
				}
				vid, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{
					Name: fmt.Sprintf("ep-%d-%d.mkv", i+1, j+1),
				})
				if err != nil {
					return err
				}
				videoIDs = append(videoIDs, vid.ID)
				if _, err := tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
					EpisodeID: ep.ID, VideoID: vid.ID,
				}); err != nil {
					return err
				}
				if _, err := tx.q.AudioTrackCreate(ctx, schema.AudioTrackCreateParams{
					VideoID: vid.ID, StreamIndex: 0, Language: "eng",
					Channels: 2, ChannelLayout: "stereo",
					SampleRate: 48000, Codec: "aac", Profile: "LC",
				}); err != nil {
					return err
				}
				rend, err := tx.q.RenditionCreate(ctx, schema.RenditionCreateParams{
					VideoID: vid.ID, Purpose: "streaming", Codec: "h264",
					TargetBitrate: 1500, MaxHeight: 720, Priority: 0,
				})
				if err != nil {
					return err
				}
				_ = rend
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, seriesID)
	}); err != nil {
		t.Fatalf("trash series: %v", err)
	}

	// Threshold in the future: every trashed row should be purged.
	threshold := time.Now().Add(time.Hour)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.trashPurge(ctx, threshold)
	}); err != nil {
		t.Fatalf("trashPurge: %v", err)
	}

	counts := map[string]int{
		"Series":        -1,
		"SeriesEdition": -1,
		"Season":        -1,
		"SeasonEpisode": -1,
		"Episode":       -1,
		"EpisodeVideo":  -1,
		"Video":         -1,
		"AudioTrack":    -1,
		"Rendition":     -1,
	}
	if err := m.WithTxR(func(tx *TxR) error {
		for name := range counts {
			var n int
			row := tx.tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+name)
			if err := row.Scan(&n); err != nil {
				return fmt.Errorf("count %s: %w", name, err)
			}
			counts[name] = n
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for name, n := range counts {
		if n != 0 {
			t.Errorf("after purge: %s rows = %d, want 0", name, n)
		}
	}
}

// TestTrashSlugCollisionOnRestore covers scenario 3: a trashed
// Movie's slug can be claimed by a new Movie; restoring the original
// auto-renames it, without announcing a slug change — the trashed
// pages had no viewers, and the old slug now addresses the claimant.
func TestTrashSlugCollisionOnRestore(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	events := subscribeEvents(ctx, m)

	movieA, err := createMovieViaPublicAPI(t, m, "Dune", "2021-01-01")
	if err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieA)
	}); err != nil {
		t.Fatalf("trash movie A: %v", err)
	}

	movieB, err := createMovieViaPublicAPI(t, m, "Dune", "2024-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if slug := getMovieSlug(t, m, movieB); slug != "dune" {
		t.Fatalf("movie B slug = %q, want %q (should take trashed A's slot)", slug, "dune")
	}

	restoreErr := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, movieA)
	})
	if restoreErr != nil {
		t.Fatalf("restore: %v", restoreErr)
	}

	slugA := getMovieSlug(t, m, movieA)
	if slugA == "dune" {
		t.Fatalf("movie A slug = %q, want auto-renamed (e.g. dune-2021)", slugA)
	}
	if slugA == "" {
		t.Fatalf("movie A slug is empty after restore")
	}

	// Verify Slug table has both entries pointing correctly.
	var targetDune, targetRenamed string
	if err := m.WithTxR(func(tx *TxR) error {
		s, err := tx.q.SlugGet(ctx, "dune")
		if err != nil {
			return fmt.Errorf("slug dune: %w", err)
		}
		targetDune = s.Target
		s, err = tx.q.SlugGet(ctx, slugA)
		if err != nil {
			return fmt.Errorf("slug %s: %w", slugA, err)
		}
		targetRenamed = s.Target
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if targetDune != movieB {
		t.Errorf("slug 'dune' points to %q, want movie B %q", targetDune, movieB)
	}
	if targetRenamed != movieA {
		t.Errorf("slug %q points to %q, want movie A %q", slugA, targetRenamed, movieA)
	}

	// The restore must not announce a slug change: a session viewing
	// /dune is on movie B's page, and following would hijack it.
	for _, ev := range events.drain() {
		if ev.Type == EventMovieSetSlug && ev.ID == movieA {
			t.Errorf("EventMovieSetSlug for movie A on restore (NewText=%q)", ev.NewText)
		}
	}
}

// TestTrashPartialUniqueIndexAllowsReuse covers scenario 4: after
// trashing a Movie with Slug and TMDBID, a fresh Movie can be
// inserted with the same Slug and TMDBID.
func TestTrashPartialUniqueIndexAllowsReuse(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	var tmdb int64 = 12345
	movieA, _ := createMovieRow(t, m, "foo", &tmdb)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieA)
	}); err != nil {
		t.Fatalf("trash movie A: %v", err)
	}

	movieB, _ := createMovieRow(t, m, "foo", &tmdb)
	if movieB == movieA {
		t.Fatal("createMovieRow returned the same ID twice")
	}

	if err := m.WithTxR(func(tx *TxR) error {
		mo, err := tx.q.MovieGet(ctx, movieB)
		if err != nil {
			return err
		}
		if mo.Slug != "foo" {
			t.Errorf("movie B slug = %q, want foo", mo.Slug)
		}
		if mo.TMDBID == nil || *mo.TMDBID != tmdb {
			t.Errorf("movie B TMDBID = %v, want %d", mo.TMDBID, tmdb)
		}
		if mo.DeletedAt != nil {
			t.Error("movie B is trashed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashPurgeRemovesCASKeys covers scenario 5: a Video's
// OriginalKey and Rendition keys are passed to store.Remove when
// purged. Uses the real storage.Dir backing the test Model; the
// files must actually disappear from the filesystem.
func TestTrashPurgeRemovesCASKeys(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	originalKey, err := m.store.Copy(strings.NewReader("original bytes"))
	if err != nil {
		t.Fatal(err)
	}
	rendKey1, err := m.store.Copy(strings.NewReader("rendition 1"))
	if err != nil {
		t.Fatal(err)
	}
	rendKey2, err := m.store.Copy(strings.NewReader("rendition 2"))
	if err != nil {
		t.Fatal(err)
	}

	vidID := createVideoRow(t, m, "cas.mkv", originalKey, []string{rendKey1, rendKey2})

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, vidID)
	}); err != nil {
		t.Fatalf("trash video: %v", err)
	}

	threshold := time.Now().Add(time.Hour)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.trashPurge(ctx, threshold)
	}); err != nil {
		t.Fatalf("trashPurge: %v", err)
	}

	if err := m.WithTxR(func(tx *TxR) error {
		_, err := tx.q.VideoGet(ctx, vidID)
		if err == nil {
			t.Error("video row still exists after purge")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{originalKey, rendKey1, rendKey2} {
		f, err := m.store.Open(k)
		if err == nil {
			f.Close()
			t.Errorf("CAS blob %q still exists after purge", k)
		}
	}
}

// TestTrashCascadeDepthFullSeries covers scenario 6: a full Series
// tree (2 editions x 1 season x 3 episodes x 1 video) cascades with
// CascadeOf = seriesID in the Trash table on every descendant, and
// Restore brings everything back.
func TestTrashCascadeDepthFullSeries(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	var (
		seriesID   string
		editionIDs []string
		seasonIDs  []string
		episodeIDs []string
		videoIDs   []string
	)
	err := m.WithTxRW(func(tx *TxRW) error {
		srID := "sr" + flurry.NewID()
		sr, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "cascade-depth-" + srID,
			Title: "Cascade Depth", Status: "Running",
			PremieredOn: "2020-01-01",
		})
		if err != nil {
			return err
		}
		seriesID = sr.ID
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: sr.Slug, Kind: "series", Target: sr.ID,
		}); err != nil {
			return err
		}
		for ei := range 2 {
			slug := ""
			if ei > 0 {
				slug = fmt.Sprintf("ed-%d", ei+1)
			}
			sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
				Label: fmt.Sprintf("Edition %d", ei+1), Slug: slug,
				SeriesID: sr.ID, Summary: "",
			})
			if err != nil {
				return err
			}
			editionIDs = append(editionIDs, sed.ID)
			sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
				EditionID: sed.ID, SortKey: "01", Title: "Season 1", Number: 1,
			})
			if err != nil {
				return err
			}
			seasonIDs = append(seasonIDs, sn.ID)
			for j := range 3 {
				ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
					Title: fmt.Sprintf("Ed%d E%d", ei+1, j+1),
					Type:  "regular", Runtime: 30,
				})
				if err != nil {
					return err
				}
				episodeIDs = append(episodeIDs, ep.ID)
				if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
					EditionID: sed.ID, SeasonID: sn.ID, EpisodeID: ep.ID,
					SortKey: int64(j + 1), Label: fmt.Sprintf("%d", j+1),
					Number: int64(j + 1),
					Slug:   fmt.Sprintf("ed%de%d", ei+1, j+1),
				}); err != nil {
					return err
				}
				vid, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{
					Name: fmt.Sprintf("ed%d-ep%d.mkv", ei+1, j+1),
				})
				if err != nil {
					return err
				}
				videoIDs = append(videoIDs, vid.ID)
				if _, err := tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
					EpisodeID: ep.ID, VideoID: vid.ID,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, seriesID)
	}); err != nil {
		t.Fatalf("trash series: %v", err)
	}

	assertCascadeOf := func(id string) {
		t.Helper()
		if err := m.WithTxR(func(tx *TxR) error {
			st, err := tx.trashState(ctx, id)
			if err != nil {
				return err
			}
			if st.live() {
				t.Errorf("%s still live after cascade", id)
				return nil
			}
			if st.cascadeOf != seriesID {
				t.Errorf("%s cascadeOf = %q, want %q", id, st.cascadeOf, seriesID)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, ids := range [][]string{editionIDs, seasonIDs, episodeIDs, videoIDs} {
		for _, id := range ids {
			assertCascadeOf(id)
		}
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, seriesID)
	}); err != nil {
		t.Fatalf("restore series: %v", err)
	}

	assertLive := func(id string) {
		t.Helper()
		if err := m.WithTxR(func(tx *TxR) error {
			st, err := tx.trashState(ctx, id)
			if err != nil {
				return err
			}
			if !st.live() {
				t.Errorf("%s still trashed after restore", id)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	assertLive(seriesID)
	for _, ids := range [][]string{editionIDs, seasonIDs, episodeIDs, videoIDs} {
		for _, id := range ids {
			assertLive(id)
		}
	}
}

// TestTrashErrorCases covers scenario 7: missing and wrong-state
// targets on Trash, Restore, and Purge.
func TestTrashErrorCases(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, _ := createMovieRow(t, m, "not-trashed", nil)
	missingID := "mo" + flurry.NewID()
	missingEpID := "ep" + flurry.NewID()
	invalidID := "xxnotaprefix"

	// Set up a cascade-trashed episode via a fresh series.
	srID, _, _, _, cascadedEpID, _ := createSeriesRow(t, m, "err-cascade", "ErrCascade")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, srID)
	}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		run  func(tx *TxRW) error
		want error
	}{
		{
			"Trash non-existent",
			func(tx *TxRW) error { return tx.Trash(ctx, missingID) },
			nil, // any non-nil error OK; not ErrAlreadyTrashed
		},
		{
			"Restore non-trashed",
			func(tx *TxRW) error { return tx.Restore(ctx, movieID) },
			ErrNotTrashed,
		},
		{
			"Restore non-existent",
			func(tx *TxRW) error { return tx.Restore(ctx, missingID) },
			nil,
		},
		{
			"Purge non-trashed",
			func(tx *TxRW) error { return tx.Purge(ctx, movieID) },
			ErrNotTrashed,
		},
		{
			"Purge non-existent",
			func(tx *TxRW) error { return tx.Purge(ctx, missingID) },
			nil,
		},
		{
			"Trash cascade-trashed episode",
			func(tx *TxRW) error { return tx.Trash(ctx, cascadedEpID) },
			ErrAlreadyTrashed,
		},
		{
			"Trash invalid-kind ID",
			func(tx *TxRW) error { return tx.Trash(ctx, invalidID) },
			nil,
		},
		{
			"Restore invalid-kind ID",
			func(tx *TxRW) error { return tx.Restore(ctx, invalidID) },
			nil,
		},
		{
			"Purge invalid-kind ID",
			func(tx *TxRW) error { return tx.Purge(ctx, invalidID) },
			nil,
		},
		{
			"Trash non-existent episode",
			func(tx *TxRW) error { return tx.Trash(ctx, missingEpID) },
			sql.ErrNoRows,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.WithTxRW(tt.run)
			if err == nil {
				t.Fatal("got nil error, want error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

// createMovieViaPublicAPI routes through MovieCreate so that slug
// generation, Movie row, MovieEdition, and Slug row are all wired up
// via the same paths used in production.
func createMovieViaPublicAPI(t *testing.T, m *Model, title, releaseDate string) (string, error) {
	t.Helper()
	ctx := context.Background()
	var id string
	err := m.WithTxRW(func(tx *TxRW) error {
		mw, err := tx.MovieCreate(ctx, title, releaseDate)
		if err != nil {
			return err
		}
		id = mw.MovieHead.ID()
		return nil
	})
	return id, err
}

func getMovieSlug(t *testing.T, m *Model, id string) string {
	t.Helper()
	ctx := context.Background()
	var slug string
	err := m.WithTxR(func(tx *TxR) error {
		mo, err := tx.q.MovieGet(ctx, id)
		if err != nil {
			return err
		}
		slug = mo.Slug
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return slug
}

// eventSink collects events fired via Model.emitEvent so tests can
// assert on them after commit hooks have run. drain reads directly
// from the channel, avoiding a race with emitEvent's non-blocking
// send into the buffered channel.
type eventSink struct {
	m  *Model
	ch chan *Event
}

func subscribeEvents(ctx context.Context, m *Model) *eventSink {
	s := &eventSink{m: m, ch: make(chan *Event, 256)}
	m.subMu.Lock()
	if m.sub == nil {
		m.sub = map[chan *Event]struct{}{}
	}
	m.sub[s.ch] = struct{}{}
	m.subMu.Unlock()
	return s
}

// drain unsubscribes and returns every event currently buffered on
// the channel. Safe to call once.
func (s *eventSink) drain() []*Event {
	s.m.subMu.Lock()
	delete(s.m.sub, s.ch)
	s.m.subMu.Unlock()
	var out []*Event
	for {
		select {
		case ev := <-s.ch:
			out = append(out, ev)
		default:
			return out
		}
	}
}

func TestTrashDefaultMovieEditionPromotesSuccessor(t *testing.T) {
	m := newTestModel(t)
	ctx := context.Background()

	moID, err := createMovieViaPublicAPI(t, m, "Dune", "2021-01-01")
	if err != nil {
		t.Fatal(err)
	}
	var ed1, ed2 string
	if err := m.WithTxR(func(tx *TxR) error {
		eds, err := tx.q.MovieEditionListByMovieID(ctx, moID)
		if err != nil {
			return err
		}
		if len(eds) != 1 || eds[0].Slug != "" {
			return fmt.Errorf("want one default edition, got %+v", eds)
		}
		ed1 = eds[0].ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		mw, err := tx.MovieEditionClone(ctx, ed1)
		if err != nil {
			return err
		}
		ed2 = mw.MovieEditionHead.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if ed2 <= ed1 {
		t.Fatalf("cloned edition ID %q should sort after original %q", ed2, ed1)
	}

	sink := subscribeEvents(ctx, m)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, ed1)
	}); err != nil {
		t.Fatal(err)
	}
	events := sink.drain()

	var got schema.MovieEdition
	if err := m.WithTxR(func(tx *TxR) error {
		e, err := tx.q.MovieEditionGet(ctx, ed2)
		got = e
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if got.Slug != "" {
		t.Fatalf("ed2 slug after trash = %q, want \"\" (promoted to default)", got.Slug)
	}

	if err := m.WithTxR(func(tx *TxR) error {
		works, err := tx.MovieWorkList(ctx)
		if err != nil {
			return err
		}
		for _, mw := range works {
			if mw.MovieHead.ID() == moID {
				return nil
			}
		}
		return fmt.Errorf("movie %q missing from MovieWorkList after trashing default edition", moID)
	}); err != nil {
		t.Fatal(err)
	}

	if !hasSlugEvent(events, EventMovieEditionSetSlug, ed2, "") {
		t.Fatalf("expected EventMovieEditionSetSlug for promoted successor %s with NewText=\"\", got %+v", ed2, events)
	}
	// The demoted default also picks up a label-derived slug; verify an
	// event fired for it with OldText="".
	if !hasSlugEventWithOldText(events, EventMovieEditionSetSlug, ed1, "") {
		t.Fatalf("expected EventMovieEditionSetSlug for demoted default %s with OldText=\"\", got %+v", ed1, events)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, ed1)
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxR(func(tx *TxR) error {
		e1, err := tx.q.MovieEditionGet(ctx, ed1)
		if err != nil {
			return err
		}
		e2, err := tx.q.MovieEditionGet(ctx, ed2)
		if err != nil {
			return err
		}
		if e2.Slug != "" {
			t.Fatalf("ed2 slug after restore = %q, want \"\" (still default)", e2.Slug)
		}
		if e1.Slug == "" {
			t.Fatalf("ed1 slug after restore = \"\", want a non-empty fallback slug")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashEpisodeTrashRestoreInMiddle reproduces the scenario:
// create a season with many episodes, trash one in the middle, then
// restore it. The restore must bump subsequent live SortKeys to make
// room for the restored junction.
func TestTrashEpisodeTrashRestoreInMiddle(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	if err := m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "cowboy-bebop-" + srID, Title: "Cowboy Bebop", Status: "Ended", PremieredOn: "1998-04-03",
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var sedID, snID string
	var epIDs []string
	if err := m.WithTxRW(func(tx *TxRW) error {
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "Season 1", Number: 1,
		})
		if err != nil {
			return err
		}
		snID = sn.ID
		for i := int64(1); i <= 5; i++ {
			ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
				Title: fmt.Sprintf("Ep %d", i), Type: "regular", Runtime: 24,
			})
			if err != nil {
				return err
			}
			epIDs = append(epIDs, ep.ID)
			if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
				EditionID: sedID, SeasonID: snID, EpisodeID: ep.ID,
				SortKey: i, Label: fmt.Sprintf("%d", i), Number: i, Slug: fmt.Sprintf("ep-%d", i),
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	targetEp := epIDs[1] // "Ep 2"
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, targetEp)
	}); err != nil {
		t.Fatalf("trash: %v", err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, targetEp)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	// After restore, all five episodes are live with Num 1..5 in the
	// same order as before the trash.
	if err := m.WithTxR(func(tx *TxR) error {
		sneps, err := tx.q.SeasonEpisodeListBySeasonID(ctx, snID)
		if err != nil {
			return err
		}
		if len(sneps) != 5 {
			t.Fatalf("got %d live junctions, want 5", len(sneps))
		}
		for i, snep := range sneps {
			if snep.EpisodeID != epIDs[i] {
				t.Errorf("position %d: got episode %s, want %s", i, snep.EpisodeID, epIDs[i])
			}
			if snep.Number != int64(i+1) {
				t.Errorf("position %d: got Number=%d, want %d", i, snep.Number, i+1)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashRestoreCascadeTrashedErrors: a cascade-trashed row isn't
// individually restorable; Restore must reject the request rather than
// silently restoring its root (those rows aren't shown on the trash
// page, so reaching this code path means the caller is wrong).
func TestTrashRestoreCascadeTrashedErrors(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, fx.seriesID)
	}); err != nil {
		t.Fatal(err)
	}
	err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, fx.edition1)
	})
	if !errors.Is(err, ErrCascadeTrashed) {
		t.Fatalf("restore cascade-trashed edition: err = %v, want ErrCascadeTrashed", err)
	}
}

// TestTrashRestoreEpisodePullsUpTrashedSeries covers the scenario where
// a user direct-trashes an Episode, then direct-trashes the containing
// Series, then restores just the Episode. The Series (and its seasons
// / edition) must come back along with the Episode, otherwise the
// Episode is live but unreachable.
func TestTrashRestoreEpisodePullsUpTrashedSeries(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	var sedID, snID, epID string
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "cowboy-bebop-" + srID, Title: "Cowboy Bebop", Status: "Ended", PremieredOn: "1998-04-03",
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "Season 1", Number: 1,
		})
		if err != nil {
			return err
		}
		snID = sn.ID
		ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "Gateway Shuffle", Type: "regular", Runtime: 24,
		})
		if err != nil {
			return err
		}
		epID = ep.ID
		return tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: snID, EpisodeID: epID,
			SortKey: 2, Label: "2", Number: 2, Slug: "s1e2-gateway-shuffle",
		})
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, epID); err != nil {
			return err
		}
		return tx.Trash(ctx, srID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, epID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := m.WithTxR(func(tx *TxR) error {
		for _, x := range []struct {
			id   string
			name string
		}{
			{srID, "series"},
			{sedID, "series edition"},
			{snID, "season"},
			{epID, "episode"},
		} {
			st, err := tx.trashState(ctx, x.id)
			if err != nil {
				return err
			}
			if !st.live() {
				t.Errorf("%s still trashed after restore", x.name)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func hasSlugEventWithOldText(events []*Event, evType, id, wantOldText string) bool {
	for _, ev := range events {
		if ev.Type == evType && ev.ID == id && ev.OldText == wantOldText {
			return true
		}
	}
	return false
}

func hasSlugEvent(events []*Event, evType, id, wantNewText string) bool {
	for _, ev := range events {
		if ev.Type == evType && ev.ID == id && ev.NewText == wantNewText {
			return true
		}
	}
	return false
}

// TestTrashRoundTripMovie covers section 1.1: trashing a Movie with
// one non-default edition, two MovieVideos, and two Videos cascades
// all descendants with CascadeOf = movieID in the Trash table, and
// Restore brings everything back live.
func TestTrashRoundTripMovie(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, medDefault := createMovieRow(t, m, "rt-movie", nil)
	var medExtra string
	if err := m.WithTxRW(func(tx *TxRW) error {
		med, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "rt-movie extra", Label: "Extended", Slug: "extended",
			MovieID: movieID, ReleaseDate: "2021-01-01",
		})
		if err != nil {
			return err
		}
		medExtra = med.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	vidA := createVideoRow(t, m, "a.mkv", "", nil)
	vidB := createVideoRow(t, m, "b.mkv", "", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medDefault, VideoID: vidA,
		}); err != nil {
			return err
		}
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medExtra, VideoID: vidB,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieID)
	}); err != nil {
		t.Fatalf("trash: %v", err)
	}
	assertTrashed(t, m, movieID, "")
	for _, id := range []string{medDefault, medExtra, vidA, vidB} {
		assertTrashed(t, m, id, movieID)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, movieID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	for _, id := range []string{movieID, medDefault, medExtra, vidA, vidB} {
		assertLive(t, m, id)
	}
}

// TestTrashRoundTripMovieEdition covers section 1.2: trashing a
// non-default MovieEdition cascades its MovieVideo + Video, leaves
// the Movie's default edition untouched, and Restore brings it back.
func TestTrashRoundTripMovieEdition(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, medDefault := createMovieRow(t, m, "rt-med", nil)
	var medExtra string
	if err := m.WithTxRW(func(tx *TxRW) error {
		med, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "extra", Label: "Extended", Slug: "extended",
			MovieID: movieID, ReleaseDate: "",
		})
		if err != nil {
			return err
		}
		medExtra = med.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	vidExtra := createVideoRow(t, m, "extra.mkv", "", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medExtra, VideoID: vidExtra,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, medExtra)
	}); err != nil {
		t.Fatalf("trash: %v", err)
	}
	assertTrashed(t, m, medExtra, "")
	assertTrashed(t, m, vidExtra, medExtra)
	assertLive(t, m, movieID)
	assertLive(t, m, medDefault)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, medExtra)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, medExtra)
	assertLive(t, m, vidExtra)
}

// TestTrashRoundTripSeriesEdition covers section 1.4: trashing a
// non-default SeriesEdition cascades Season + Episode + junctions +
// Video, leaves the Series's default edition alone, and Restore
// brings it back.
func TestTrashRoundTripSeriesEdition(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID, sedDefault, sedExtra, snID, epID, vidID := createSeriesRow(t, m, "rt-sed", "rt sed")

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, sedExtra)
	}); err != nil {
		t.Fatalf("trash: %v", err)
	}
	assertTrashed(t, m, sedExtra, "")
	for _, id := range []string{snID, epID, vidID} {
		assertTrashed(t, m, id, sedExtra)
	}
	assertLive(t, m, srID)
	assertLive(t, m, sedDefault)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, sedExtra)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	for _, id := range []string{sedExtra, snID, epID, vidID} {
		assertLive(t, m, id)
	}
}

// TestTrashRoundTripSeason covers section 1.5: a Season shared with a
// sibling season (via second season containing the same Episode)
// keeps the shared Episode live on Trash; the Season-only Episode is
// reaped. Restoring brings everything back.
func TestTrashRoundTripSeason(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	var sedID, snA, snB, epShared, epOnly, vidShared, vidOnly string
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "rt-season-" + srID, Title: "rt-season",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		a, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "S1", Number: 1,
		})
		if err != nil {
			return err
		}
		snA = a.ID
		b, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "02", Title: "S2", Number: 2,
		})
		if err != nil {
			return err
		}
		snB = b.ID
		shared, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "Shared", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		epShared = shared.ID
		only, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "Only", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		epOnly = only.ID
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: snA, EpisodeID: epShared,
			SortKey: 1, Label: "1", Number: 1, Slug: "s1-shared",
		}); err != nil {
			return err
		}
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: snA, EpisodeID: epOnly,
			SortKey: 2, Label: "2", Number: 2, Slug: "s1-only",
		}); err != nil {
			return err
		}
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: snB, EpisodeID: epShared,
			SortKey: 1, Label: "1", Number: 1, Slug: "s2-shared",
		}); err != nil {
			return err
		}
		vs, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{Name: "shared.mkv"})
		if err != nil {
			return err
		}
		vidShared = vs.ID
		vo, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{Name: "only.mkv"})
		if err != nil {
			return err
		}
		vidOnly = vo.ID
		if _, err := tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: epShared, VideoID: vidShared,
		}); err != nil {
			return err
		}
		_, err = tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: epOnly, VideoID: vidOnly,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, snA)
	}); err != nil {
		t.Fatalf("trash season A: %v", err)
	}
	assertTrashed(t, m, snA, "")
	assertTrashed(t, m, epOnly, snA)
	assertTrashed(t, m, vidOnly, snA)
	assertLive(t, m, epShared)
	assertLive(t, m, vidShared)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, snA)
	}); err != nil {
		t.Fatalf("restore season A: %v", err)
	}
	for _, id := range []string{snA, epOnly, vidOnly, epShared, vidShared, snB} {
		assertLive(t, m, id)
	}
}

// TestTrashRoundTripEpisode covers section 1.6: trashing a lone
// Episode cascades its SeasonEpisode + EpisodeVideo + Video and
// Restore brings them back.
func TestTrashRoundTripEpisode(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, _, _, _, epID, vidID := createSeriesRow(t, m, "rt-ep", "rt ep")

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, epID)
	}); err != nil {
		t.Fatalf("trash: %v", err)
	}
	assertTrashed(t, m, epID, "")
	assertTrashed(t, m, vidID, epID)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, epID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, epID)
	assertLive(t, m, vidID)
}

// TestTrashRoundTripVideoShared covers section 1.7: trashing a Video
// shared between an Episode and a MovieEdition cascades the two
// junctions but leaves the Episode and MovieEdition live.
func TestTrashRoundTripVideoShared(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	movieID, medID := createMovieRow(t, m, "rt-vid", nil)
	vidID := createVideoRow(t, m, "shared.mkv", "", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: fx.episodeID, VideoID: vidID,
		}); err != nil {
			return err
		}
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medID, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, vidID)
	}); err != nil {
		t.Fatalf("trash: %v", err)
	}
	assertTrashed(t, m, vidID, "")
	assertLive(t, m, fx.episodeID)
	assertLive(t, m, medID)
	assertLive(t, m, movieID)
	if n := countRows(t, m, "EpisodeVideo",
		"VideoID = ? AND DeletedAt IS NOT NULL", vidID); n != 1 {
		t.Errorf("EpisodeVideo trashed junction rows = %d, want 1", n)
	}
	if n := countRows(t, m, "MovieVideo",
		"VideoID = ? AND DeletedAt IS NOT NULL", vidID); n != 1 {
		t.Errorf("MovieVideo trashed junction rows = %d, want 1", n)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, vidID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, vidID)
	if n := countRows(t, m, "EpisodeVideo",
		"VideoID = ? AND DeletedAt IS NULL", vidID); n != 1 {
		t.Errorf("EpisodeVideo live rows = %d, want 1", n)
	}
	if n := countRows(t, m, "MovieVideo",
		"VideoID = ? AND DeletedAt IS NULL", vidID); n != 1 {
		t.Errorf("MovieVideo live rows = %d, want 1", n)
	}
}

// TestTrashRoundTripCollection covers section 1.8: trashing a
// Collection cascades its CollectionMovie/CollectionSeries junctions
// but leaves the member Movies and Series untouched.
func TestTrashRoundTripCollection(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	colID := createCollectionRow(t, m, "my-col", "My Collection")
	movieID, _ := createMovieRow(t, m, "col-member", nil)
	srID, _, _, _, _, _ := createSeriesRow(t, m, "col-series", "Col Series")

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.q.CollectionMovieAdd(ctx, schema.CollectionMovieAddParams{
			CollectionID: colID, MovieID: movieID,
		}); err != nil {
			return err
		}
		return tx.q.CollectionSeriesAdd(ctx, schema.CollectionSeriesAddParams{
			CollectionID: colID, SeriesID: srID,
		})
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, colID)
	}); err != nil {
		t.Fatalf("trash: %v", err)
	}
	assertTrashed(t, m, colID, "")
	assertLive(t, m, movieID)
	assertLive(t, m, srID)
	if n := countRows(t, m, "CollectionMovie",
		"CollectionID = ? AND DeletedAt IS NOT NULL", colID); n != 1 {
		t.Errorf("CollectionMovie trashed rows = %d, want 1", n)
	}
	if n := countRows(t, m, "CollectionSeries",
		"CollectionID = ? AND DeletedAt IS NOT NULL", colID); n != 1 {
		t.Errorf("CollectionSeries trashed rows = %d, want 1", n)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, colID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, colID)
	if n := countRows(t, m, "CollectionMovie",
		"CollectionID = ? AND DeletedAt IS NULL", colID); n != 1 {
		t.Errorf("CollectionMovie live rows = %d, want 1", n)
	}
	if n := countRows(t, m, "CollectionSeries",
		"CollectionID = ? AND DeletedAt IS NULL", colID); n != 1 {
		t.Errorf("CollectionSeries live rows = %d, want 1", n)
	}
}

// TestTrashRestoreMEdPullsUpMovie covers section 2.1: direct-trash a
// non-default MovieEdition, then direct-trash the Movie, then Restore
// the MovieEdition — the Movie must come back live.
func TestTrashRestoreMEdPullsUpMovie(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, medDefault := createMovieRow(t, m, "pull-up-movie", nil)
	var medExtra string
	if err := m.WithTxRW(func(tx *TxRW) error {
		med, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "extra", Label: "Extended", Slug: "extended",
			MovieID: movieID, ReleaseDate: "",
		})
		if err != nil {
			return err
		}
		medExtra = med.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, medExtra); err != nil {
			return err
		}
		return tx.Trash(ctx, movieID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, medExtra)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, movieID)
	assertLive(t, m, medExtra)
	// The default edition was pulled up as part of the Movie restore
	// (cascadeOf = movieID).
	assertLive(t, m, medDefault)
}

// TestTrashRestoreSEdPullsUpSeries covers section 2.2.
func TestTrashRestoreSEdPullsUpSeries(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID, sedDefault, sedExtra, _, _, _ := createSeriesRow(t, m, "pull-up-sr", "Pull Up Sr")

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, sedExtra); err != nil {
			return err
		}
		return tx.Trash(ctx, srID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, sedExtra)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, srID)
	assertLive(t, m, sedExtra)
	assertLive(t, m, sedDefault)
}

// TestTrashRestoreSeasonPullsUpSEd covers section 2.3.
func TestTrashRestoreSeasonPullsUpSEd(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, _, sedExtra, snID, _, _ := createSeriesRow(t, m, "pull-up-sn", "Pull Up Sn")

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, snID); err != nil {
			return err
		}
		return tx.Trash(ctx, sedExtra)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, snID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, sedExtra)
	assertLive(t, m, snID)
}

// TestTrashRestoreEpisodePullsUpSeason covers section 2.4: direct-
// trash an Episode, then direct-trash its Season, then Restore the
// Episode — the Season must come back, and the Episode's SortKey is
// preserved in the restored junction.
func TestTrashRestoreEpisodePullsUpSeason(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, _, _, snID, epID, _ := createSeriesRow(t, m, "pull-up-ep", "Pull Up Ep")

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, epID); err != nil {
			return err
		}
		return tx.Trash(ctx, snID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, epID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, snID)
	assertLive(t, m, epID)
	if err := m.WithTxR(func(tx *TxR) error {
		sneps, err := tx.q.SeasonEpisodeListBySeasonID(ctx, snID)
		if err != nil {
			return err
		}
		if len(sneps) != 1 {
			t.Fatalf("season junctions = %d, want 1", len(sneps))
		}
		if sneps[0].SortKey != 1 {
			t.Errorf("SortKey = %d, want 1 (preserved)", sneps[0].SortKey)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashRestoreVideoPullsUpMEd covers section 2.5: direct-trash a
// Video, then direct-trash its containing MovieEdition; restoring the
// Video chain-restores the MovieEdition via the MovieVideo junction.
func TestTrashRestoreVideoPullsUpMEd(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, _ := createMovieRow(t, m, "pull-up-vid-med", nil)
	var medExtra string
	if err := m.WithTxRW(func(tx *TxRW) error {
		med, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "extra", Label: "Extended", Slug: "extended",
			MovieID: movieID, ReleaseDate: "",
		})
		if err != nil {
			return err
		}
		medExtra = med.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	vidID := createVideoRow(t, m, "v.mkv", "", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medExtra, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, vidID); err != nil {
			return err
		}
		return tx.Trash(ctx, medExtra)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, vidID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, vidID)
	assertLive(t, m, medExtra)
}

// TestTrashRestoreVideoPullsUpEpisode covers section 2.6: direct-trash
// a Video, then direct-trash its containing Episode; restoring the
// Video chain-restores the Episode (and everything above it).
func TestTrashRestoreVideoPullsUpEpisode(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, _, _, _, epID, vidID := createSeriesRow(t, m, "pull-up-vid-ep", "PullVidEp")

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, vidID); err != nil {
			return err
		}
		return tx.Trash(ctx, epID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, vidID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, vidID)
	assertLive(t, m, epID)
}

// TestTrashRestoreEpisodeMultiHop covers section 2.7: inside-out trash
// of Episode, Season, SEd, Series; Restore the Episode pulls up the
// whole chain.
func TestTrashRestoreEpisodeMultiHop(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID, _, sedExtra, snID, epID, vidID := createSeriesRow(t, m, "pull-up-chain", "Chain")

	if err := m.WithTxRW(func(tx *TxRW) error {
		for _, id := range []string{epID, snID, sedExtra, srID} {
			if err := tx.Trash(ctx, id); err != nil {
				return fmt.Errorf("trash %s: %w", id, err)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, epID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	for _, id := range []string{srID, sedExtra, snID, epID, vidID} {
		assertLive(t, m, id)
	}
}

// TestTrashRestoreMovieInTrashedCollection covers section 2.8: when a
// Movie is in a trashed Collection, Restoring the Movie leaves the
// Collection trashed — a Collection isn't a structural ancestor of
// its members, so there's no chain to walk.
func TestTrashRestoreMovieInTrashedCollection(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	colID := createCollectionRow(t, m, "col-with-movie", "Col With Movie")
	movieID, _ := createMovieRow(t, m, "col-with-movie-mv", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.q.CollectionMovieAdd(ctx, schema.CollectionMovieAddParams{
			CollectionID: colID, MovieID: movieID,
		})
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, movieID); err != nil {
			return err
		}
		return tx.Trash(ctx, colID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, movieID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, movieID)
	assertTrashed(t, m, colID, "")
}

// TestTrashRestoreSiblingsMovie covers section 3.1: trashing both
// Movies and restoring one leaves the other trashed.
func TestTrashRestoreSiblingsMovie(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	mA, _ := createMovieRow(t, m, "sibling-a", nil)
	mB, _ := createMovieRow(t, m, "sibling-b", nil)

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, mA); err != nil {
			return err
		}
		return tx.Trash(ctx, mB)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, mA)
	}); err != nil {
		t.Fatalf("restore A: %v", err)
	}
	assertLive(t, m, mA)
	assertTrashed(t, m, mB, "")
}

// TestTrashRestoreSiblingsEpisode covers section 3.2: two Episodes in
// a Season; trash both then restore one; the other stays trashed;
// renumberSeason correct.
func TestTrashRestoreSiblingsEpisode(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	var sedID, snID, ep1, ep2 string
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "sib-eps-" + srID, Title: "Sib Eps",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "S1", Number: 1,
		})
		if err != nil {
			return err
		}
		snID = sn.ID
		for _, i := range []int64{1, 2} {
			ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
				Title: fmt.Sprintf("Ep %d", i), Type: "regular", Runtime: 30,
			})
			if err != nil {
				return err
			}
			switch i {
			case 1:
				ep1 = ep.ID
			case 2:
				ep2 = ep.ID
			}
			if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
				EditionID: sedID, SeasonID: snID, EpisodeID: ep.ID,
				SortKey: i, Label: fmt.Sprintf("%d", i), Number: i,
				Slug: fmt.Sprintf("ep-%d", i),
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, ep1); err != nil {
			return err
		}
		return tx.Trash(ctx, ep2)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, ep1)
	}); err != nil {
		t.Fatalf("restore ep1: %v", err)
	}
	assertLive(t, m, ep1)
	assertTrashed(t, m, ep2, "")
	if err := m.WithTxR(func(tx *TxR) error {
		sneps, err := tx.q.SeasonEpisodeListBySeasonID(ctx, snID)
		if err != nil {
			return err
		}
		if len(sneps) != 1 {
			t.Fatalf("live junctions = %d, want 1", len(sneps))
		}
		if sneps[0].Number != 1 {
			t.Errorf("restored ep Number = %d, want 1", sneps[0].Number)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashRestoreSiblingsCollectionMovies covers section 3.3: two
// Movies in a Collection, both trashed, restore one — other stays
// trashed.
func TestTrashRestoreSiblingsCollectionMovies(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	colID := createCollectionRow(t, m, "sib-col", "Sib Col")
	mA, _ := createMovieRow(t, m, "sib-col-a", nil)
	mB, _ := createMovieRow(t, m, "sib-col-b", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.q.CollectionMovieAdd(ctx, schema.CollectionMovieAddParams{
			CollectionID: colID, MovieID: mA,
		}); err != nil {
			return err
		}
		return tx.q.CollectionMovieAdd(ctx, schema.CollectionMovieAddParams{
			CollectionID: colID, MovieID: mB,
		})
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, mA); err != nil {
			return err
		}
		return tx.Trash(ctx, mB)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, mA)
	}); err != nil {
		t.Fatalf("restore mA: %v", err)
	}
	assertLive(t, m, mA)
	assertTrashed(t, m, mB, "")
	assertLive(t, m, colID)
}

// TestTrashRestoreJunctionSharedEpisodeEditionTrash exercises the
// "both sides live" junction-restore invariant: an Episode shared by
// two editions stays live when one edition is trashed (the other
// still references it), and restoring that edition must bring back
// the soft-deleted (season, episode) junction even though the
// Episode itself was never trashed.
func TestTrashRestoreJunctionSharedEpisodeEditionTrash(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	fx := newTrashTestFixture(t, m)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, fx.edition1)
	}); err != nil {
		t.Fatalf("trash edition 1: %v", err)
	}
	assertLive(t, m, fx.episodeID)
	if n := countRows(t, m, "SeasonEpisode",
		"SeasonID = ? AND EpisodeID = ? AND DeletedAt IS NOT NULL",
		fx.season1, fx.episodeID); n != 1 {
		t.Errorf("(season1, ep) trashed junction rows = %d, want 1", n)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, fx.edition1)
	}); err != nil {
		t.Fatalf("restore edition 1: %v", err)
	}
	assertLive(t, m, fx.edition1)
	assertLive(t, m, fx.season1)
	if n := countRows(t, m, "SeasonEpisode",
		"SeasonID = ? AND EpisodeID = ? AND DeletedAt IS NULL",
		fx.season1, fx.episodeID); n != 1 {
		t.Errorf("(season1, ep) live junction rows = %d, want 1", n)
	}
}

// TestTrashRestoreJunctionSharedVideoMEdTrash is the movie analogue
// of the shared-episode test: a Video in two non-default
// MovieEditions stays live when one MEd is trashed; restoring that
// MEd must restore the soft-deleted (med, video) junction.
func TestTrashRestoreJunctionSharedVideoMEdTrash(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, _ := createMovieRow(t, m, "junc-two-med", nil)
	var med1, med2 string
	if err := m.WithTxRW(func(tx *TxRW) error {
		a, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "A", Label: "A", Slug: "a", MovieID: movieID,
		})
		if err != nil {
			return err
		}
		med1 = a.ID
		b, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "B", Label: "B", Slug: "b", MovieID: movieID,
		})
		if err != nil {
			return err
		}
		med2 = b.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	vidID := createVideoRow(t, m, "shared.mkv", "", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: med1, VideoID: vidID,
		}); err != nil {
			return err
		}
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: med2, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, med1)
	}); err != nil {
		t.Fatalf("trash med1: %v", err)
	}
	assertLive(t, m, vidID)
	if n := countRows(t, m, "MovieVideo",
		"MovieEditionID = ? AND VideoID = ? AND DeletedAt IS NOT NULL",
		med1, vidID); n != 1 {
		t.Errorf("(med1, vid) trashed junction rows = %d, want 1", n)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, med1)
	}); err != nil {
		t.Fatalf("restore med1: %v", err)
	}
	assertLive(t, m, med1)
	if n := countRows(t, m, "MovieVideo",
		"MovieEditionID = ? AND VideoID = ? AND DeletedAt IS NULL",
		med1, vidID); n != 1 {
		t.Errorf("(med1, vid) live junction rows = %d, want 1", n)
	}
}

// TestTrashRestoreJunctionOtherSideStillTrashed locks in the "both
// sides live" guard: restoring one cascade root must NOT restore a
// junction whose other side is still trashed as cascade of a
// different root. Video is referenced by Episode-ep (Series sr) and
// MEd-med (Movie mo). Trashing sr soft-deletes (ep,v) but v stays
// live via (med,v). Trashing mo then orphan-reaps v. Restoring sr
// must leave (ep,v) soft-deleted because v is still trashed.
func TestTrashRestoreJunctionOtherSideStillTrashed(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID, _, _, _, epID, vidID := createSeriesRow(t, m, "junc-other-side", "Junc")
	movieID, medID := createMovieRow(t, m, "junc-other-side-mv", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medID, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, srID)
	}); err != nil {
		t.Fatalf("trash series: %v", err)
	}
	assertLive(t, m, vidID)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieID)
	}); err != nil {
		t.Fatalf("trash movie: %v", err)
	}
	assertTrashed(t, m, vidID, movieID)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, srID)
	}); err != nil {
		t.Fatalf("restore series: %v", err)
	}
	assertLive(t, m, epID)
	assertTrashed(t, m, vidID, movieID)
	if n := countRows(t, m, "EpisodeVideo",
		"EpisodeID = ? AND VideoID = ? AND DeletedAt IS NOT NULL",
		epID, vidID); n != 1 {
		t.Errorf("(ep, vid) junction must stay trashed while vid is trashed; got %d soft-deleted rows, want 1", n)
	}
}

// TestTrashRestoreJunctionDanglingAutoRestore locks in the behavior
// that a junction soft-deleted under cascade root R1 is auto-
// restored when a LATER restore of cascade root R2 brings its other
// side live, since the query scope ("one side in R2 subtree") plus
// "both sides live" is now satisfied. This is new-system behavior
// the old per-row DeletedAsCascadeOf bookkeeping could not express.
func TestTrashRestoreJunctionDanglingAutoRestore(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID, _, _, _, epID, vidID := createSeriesRow(t, m, "junc-dangling", "Dangling")
	movieID, medID := createMovieRow(t, m, "junc-dangling-mv", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: medID, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, srID); err != nil {
			return err
		}
		return tx.Trash(ctx, movieID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Restore(ctx, srID); err != nil {
			return err
		}
		return tx.Restore(ctx, movieID)
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	assertLive(t, m, srID)
	assertLive(t, m, movieID)
	assertLive(t, m, vidID)
	// (ep, vid) was soft-deleted under series cascade; the movie
	// restore scope includes vidID (was in movie cascade) so the
	// junction's scope clause matches. Both sides are now live, so
	// the junction is auto-restored.
	if n := countRows(t, m, "EpisodeVideo",
		"EpisodeID = ? AND VideoID = ? AND DeletedAt IS NULL",
		epID, vidID); n != 1 {
		t.Errorf("(ep, vid) junction should be auto-restored; got %d live rows, want 1", n)
	}
}

// TestTrashOrphanReapAcrossSeries covers section 4.1: two Series
// sharing one Episode; trashing series1 keeps Episode live (still
// ref'd by series2); trashing series2 reaps Episode; restoring
// series2 un-reaps it.
func TestTrashOrphanReapAcrossSeries(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	var sr1, sr2, epID string
	if err := m.WithTxRW(func(tx *TxRW) error {
		for i, slug := range []string{"reap-sr1", "reap-sr2"} {
			id := "sr" + flurry.NewID()
			if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
				ID: id, Slug: slug, Title: slug,
				Status: "Running", PremieredOn: "2020-01-01",
			}); err != nil {
				return err
			}
			if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
				Slug: slug, Kind: "series", Target: id,
			}); err != nil {
				return err
			}
			if i == 0 {
				sr1 = id
			} else {
				sr2 = id
			}
		}
		ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "Shared", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		epID = ep.ID
		for _, srID := range []string{sr1, sr2} {
			sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
				Label: "Default", Slug: "", SeriesID: srID, Summary: "",
			})
			if err != nil {
				return err
			}
			sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
				EditionID: sed.ID, SortKey: "01", Title: "S1", Number: 1,
			})
			if err != nil {
				return err
			}
			if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
				EditionID: sed.ID, SeasonID: sn.ID, EpisodeID: epID,
				SortKey: 1, Label: "1", Number: 1, Slug: "shared",
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, sr1)
	}); err != nil {
		t.Fatalf("trash sr1: %v", err)
	}
	assertLive(t, m, epID)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, sr2)
	}); err != nil {
		t.Fatalf("trash sr2: %v", err)
	}
	assertTrashed(t, m, epID, sr2)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, sr2)
	}); err != nil {
		t.Fatalf("restore sr2: %v", err)
	}
	assertLive(t, m, epID)
}

// TestTrashCascadeEventListsOrphans covers section 4.2: restoring a
// cascade root fires an EventRestore whose TrashItems include the
// reaped Episode.
func TestTrashCascadeEventListsOrphans(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID, _, _, _, epID, _ := createSeriesRow(t, m, "cascade-orphan-ev", "Cascade Orphan")

	sink := subscribeEvents(ctx, m)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, srID)
	}); err != nil {
		t.Fatal(err)
	}
	trashEvents := sink.drain()
	var cascade *Event
	for _, ev := range trashEvents {
		if ev.Type == EventTrashCascade && ev.ID == srID {
			cascade = ev
			break
		}
	}
	if cascade == nil {
		t.Fatal("no EventTrashCascade for series")
	}
	var foundEp bool
	for _, it := range cascade.TrashItems {
		if it.ID == epID {
			foundEp = true
			break
		}
	}
	if !foundEp {
		t.Errorf("trash-cascade items missing Episode %q; got %+v", epID, cascade.TrashItems)
	}

	sink = subscribeEvents(ctx, m)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, srID)
	}); err != nil {
		t.Fatal(err)
	}
	var restore *Event
	for _, ev := range sink.drain() {
		if ev.Type == EventRestore && ev.ID == srID {
			restore = ev
			break
		}
	}
	if restore == nil {
		t.Fatal("no EventRestore for series")
	}
	foundEp = false
	for _, it := range restore.TrashItems {
		if it.ID == epID {
			foundEp = true
			break
		}
	}
	if !foundEp {
		t.Errorf("restore items missing Episode %q; got %+v", epID, restore.TrashItems)
	}
}

// TestTrashVideoAcrossTwoMEds covers section 4.3: a Video in two
// non-default MovieEditions of the same Movie stays live when only
// one MEd is trashed, and is reaped when the second MEd is trashed.
func TestTrashVideoAcrossTwoMEds(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, _ := createMovieRow(t, m, "two-med-vid", nil)
	var med1, med2 string
	if err := m.WithTxRW(func(tx *TxRW) error {
		a, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "A", Label: "A", Slug: "a",
			MovieID: movieID, ReleaseDate: "",
		})
		if err != nil {
			return err
		}
		med1 = a.ID
		b, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
			Title: "B", Label: "B", Slug: "b",
			MovieID: movieID, ReleaseDate: "",
		})
		if err != nil {
			return err
		}
		med2 = b.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	vidID := createVideoRow(t, m, "shared.mkv", "", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: med1, VideoID: vidID,
		}); err != nil {
			return err
		}
		_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
			MovieEditionID: med2, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, med2)
	}); err != nil {
		t.Fatalf("trash med2: %v", err)
	}
	assertLive(t, m, vidID)

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, med1)
	}); err != nil {
		t.Fatalf("trash med1: %v", err)
	}
	assertTrashed(t, m, vidID, med1)
}

// TestTrashVideoAcrossTwoSeasons covers section 4.4: a Video linked
// via Episode A (Season 1) and Episode B (Season 2) of the same
// edition stays live when only Season 1 is trashed.
func TestTrashVideoAcrossTwoSeasons(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	var sedID, sn1, sn2, epA, epB, vidID string
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "vid-2-sn-" + srID, Title: "Vid2Sn",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		a, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "S1", Number: 1,
		})
		if err != nil {
			return err
		}
		sn1 = a.ID
		b, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "02", Title: "S2", Number: 2,
		})
		if err != nil {
			return err
		}
		sn2 = b.ID
		ea, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "A", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		epA = ea.ID
		eb, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "B", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		epB = eb.ID
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: sn1, EpisodeID: epA,
			SortKey: 1, Label: "1", Number: 1, Slug: "ep-a",
		}); err != nil {
			return err
		}
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: sn2, EpisodeID: epB,
			SortKey: 1, Label: "1", Number: 1, Slug: "ep-b",
		}); err != nil {
			return err
		}
		v, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{Name: "v.mkv"})
		if err != nil {
			return err
		}
		vidID = v.ID
		if _, err := tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: epA, VideoID: vidID,
		}); err != nil {
			return err
		}
		_, err = tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: epB, VideoID: vidID,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, sn1)
	}); err != nil {
		t.Fatalf("trash sn1: %v", err)
	}
	assertLive(t, m, vidID)
	assertLive(t, m, epB)
	_ = sn2
}

// TestTrashSeriesSlugReuse covers section 5.1: a trashed Series'
// slug is freed by the partial unique index, so a fresh Series can
// claim it.
func TestTrashSeriesSlugReuse(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	sr1, _, _, _, _, _ := createSeriesRow(t, m, "foo", "Foo")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, sr1)
	}); err != nil {
		t.Fatalf("trash sr1: %v", err)
	}
	sr2, _, _, _, _, _ := createSeriesRow(t, m, "foo", "Foo2")
	if sr2 == sr1 {
		t.Fatal("second series got the same ID")
	}
	if err := m.WithTxR(func(tx *TxR) error {
		sr, err := tx.q.SeriesGet(ctx, sr2)
		if err != nil {
			return err
		}
		if sr.Slug != "foo" {
			t.Errorf("sr2 slug = %q, want foo", sr.Slug)
		}
		if sr.DeletedAt != nil {
			t.Error("sr2 is trashed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashCollectionSlugReuse covers section 5.2.
func TestTrashCollectionSlugReuse(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	c1 := createCollectionRow(t, m, "foo", "Foo")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, c1)
	}); err != nil {
		t.Fatalf("trash c1: %v", err)
	}
	c2 := createCollectionRow(t, m, "foo", "Foo2")
	if c2 == c1 {
		t.Fatal("second collection got the same ID")
	}
	if err := m.WithTxR(func(tx *TxR) error {
		col, err := tx.q.CollectionGet(ctx, c2)
		if err != nil {
			return err
		}
		if col.Slug != "foo" {
			t.Errorf("c2 slug = %q, want foo", col.Slug)
		}
		if col.DeletedAt != nil {
			t.Error("c2 is trashed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashSlugGetAfterTrash covers section 5.5: after Trash(Movie),
// SlugGet("<slug>") returns sql.ErrNoRows.
func TestTrashSlugGetAfterTrash(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	mID, _ := createMovieRow(t, m, "gone", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, mID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxR(func(tx *TxR) error {
		_, err := tx.q.SlugGet(ctx, "gone")
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("SlugGet after trash: err = %v, want sql.ErrNoRows", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashSlugCollisionOnRestoreSeries covers section 5.3: trashed
// Series' slug taken by a new Series; restoring the original must
// auto-rename. Uses MovieCreate-analog path via the sr* public
// helpers by way of direct q calls + seriesEnsureSlug.
func TestTrashSlugCollisionOnRestoreSeries(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	events := subscribeEvents(ctx, m)

	srA, _, _, _, _, _ := createSeriesRow(t, m, "breaking-bad", "Breaking Bad")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, srA)
	}); err != nil {
		t.Fatal(err)
	}
	srB, _, _, _, _, _ := createSeriesRow(t, m, "breaking-bad", "Breaking Bad")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, srA)
	}); err != nil {
		t.Fatalf("restore srA: %v", err)
	}
	var slugA, slugB string
	if err := m.WithTxR(func(tx *TxR) error {
		a, err := tx.q.SeriesGet(ctx, srA)
		if err != nil {
			return err
		}
		slugA = a.Slug
		b, err := tx.q.SeriesGet(ctx, srB)
		if err != nil {
			return err
		}
		slugB = b.Slug
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if slugA == "breaking-bad" {
		t.Errorf("srA slug = %q, want auto-renamed", slugA)
	}
	if slugB != "breaking-bad" {
		t.Errorf("srB slug = %q, want breaking-bad", slugB)
	}
	// No slug-change announcement on restore; see
	// TestTrashSlugCollisionOnRestore.
	if hasSlugEvent(events.drain(), EventSeriesSetSlug, srA, slugA) {
		t.Errorf("EventSeriesSetSlug for srA on restore (NewText=%q)", slugA)
	}
}

// TestTrashSlugCollisionOnRestoreCollection covers section 5.4.
func TestTrashSlugCollisionOnRestoreCollection(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	events := subscribeEvents(ctx, m)

	cA := createCollectionRow(t, m, "classics", "Classics")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, cA)
	}); err != nil {
		t.Fatal(err)
	}
	cB := createCollectionRow(t, m, "classics", "Classics")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, cA)
	}); err != nil {
		t.Fatalf("restore cA: %v", err)
	}
	var slugA, slugB string
	if err := m.WithTxR(func(tx *TxR) error {
		a, err := tx.q.CollectionGet(ctx, cA)
		if err != nil {
			return err
		}
		slugA = a.Slug
		b, err := tx.q.CollectionGet(ctx, cB)
		if err != nil {
			return err
		}
		slugB = b.Slug
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if slugA == "classics" {
		t.Errorf("cA slug = %q, want auto-renamed", slugA)
	}
	if slugB != "classics" {
		t.Errorf("cB slug = %q, want classics", slugB)
	}
	// No slug-change announcement on restore; see
	// TestTrashSlugCollisionOnRestore.
	if hasSlugEvent(events.drain(), EventCollectionSetSlug, cA, slugA) {
		t.Errorf("EventCollectionSetSlug for cA on restore (NewText=%q)", slugA)
	}
}

// TestTrashDefaultSeriesEditionPromotesSuccessor covers section 6.1:
// SeriesEdition analog of TestTrashDefaultMovieEditionPromotesSuccessor.
func TestTrashDefaultSeriesEditionPromotesSuccessor(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	var srID, ed1, ed2 string
	if err := m.WithTxRW(func(tx *TxRW) error {
		srID = "sr" + flurry.NewID()
		_, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "def-" + srID, Title: "D",
			Status: "Running", PremieredOn: "2020-01-01",
		})
		if err != nil {
			return err
		}
		def, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		ed1 = def.ID
		nondef, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Extended", Slug: "extended", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		ed2 = nondef.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	events := collectEvents(t, m, func() {
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, ed1)
		}); err != nil {
			t.Fatal(err)
		}
	})
	if err := m.WithTxR(func(tx *TxR) error {
		e2, err := tx.q.SeriesEditionGet(ctx, ed2)
		if err != nil {
			return err
		}
		if e2.Slug != "" {
			t.Errorf("ed2 slug after promotion = %q, want \"\"", e2.Slug)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !hasSlugEvent(events, EventSeriesEditionSetSlug, ed2, "") {
		t.Errorf("expected EventSeriesEditionSetSlug promoting ed2 to \"\"")
	}
}

// TestTrashDefaultMovieEditionSole covers section 6.2: Trashing the
// only MovieEdition fails with sql.ErrNoRows (no successor).
func TestTrashDefaultMovieEditionSole(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	_, medID := createMovieRow(t, m, "sole-med", nil)
	err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, medID)
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("trash sole default MEd: err = %v, want wraps sql.ErrNoRows", err)
	}
}

// TestTrashDefaultSeriesEditionSole covers section 6.3.
func TestTrashDefaultSeriesEditionSole(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	var srID, sedID string
	if err := m.WithTxRW(func(tx *TxRW) error {
		srID = "sr" + flurry.NewID()
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "sole-sed-" + srID, Title: "Sole",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, sedID)
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("trash sole default SEd: err = %v, want wraps sql.ErrNoRows", err)
	}
}

// TestTrashDefaultSeriesEditionPromotesLexSmallest covers section
// 6.4: when there are multiple non-default candidates, the lex-
// smallest ID wins.
func TestTrashDefaultSeriesEditionPromotesLexSmallest(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	var srID, edDefault, edZZZ, edAAA string
	if err := m.WithTxRW(func(tx *TxRW) error {
		srID = "sr" + flurry.NewID()
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "lex-sed-" + srID, Title: "Lex",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		def, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		edDefault = def.ID
		zzz, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "zzz", Slug: "zzz", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		edZZZ = zzz.ID
		aaa, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "aaa", Slug: "aaa", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		edAAA = aaa.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// edZZZ's ID was created before edAAA's, so edZZZ has the lex-
	// smaller flurry ID regardless of the label. The default
	// successor query orders by ID, so edZZZ wins.
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, edDefault)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxR(func(tx *TxR) error {
		z, err := tx.q.SeriesEditionGet(ctx, edZZZ)
		if err != nil {
			return err
		}
		if z.Slug != "" {
			t.Errorf("edZZZ (lex-smallest ID) slug = %q, want \"\" (promoted)", z.Slug)
		}
		a, err := tx.q.SeriesEditionGet(ctx, edAAA)
		if err != nil {
			return err
		}
		if a.Slug == "" {
			t.Errorf("edAAA slug = \"\" (should not be promoted)")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestPurgeByKind covers section 7.1-7.6: for each kind, Trash then
// Purge, assert the target and cascade subtree rows are hard-deleted.
func TestPurgeByKind(t *testing.T) {
	t.Run("movie", func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		movieID, medID := createMovieRow(t, m, "purge-movie", nil)
		vidID := createVideoRow(t, m, "pm.mkv", "", nil)
		if err := m.WithTxRW(func(tx *TxRW) error {
			_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
				MovieEditionID: medID, VideoID: vidID,
			})
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, movieID)
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Purge(ctx, movieID)
		}); err != nil {
			t.Fatalf("purge: %v", err)
		}
		if n := countRows(t, m, "Movie", "ID = ?", movieID); n != 0 {
			t.Errorf("Movie rows = %d, want 0", n)
		}
		if n := countRows(t, m, "MovieEdition", "ID = ?", medID); n != 0 {
			t.Errorf("MovieEdition rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Video", "ID = ?", vidID); n != 0 {
			t.Errorf("Video rows = %d, want 0", n)
		}
		if n := countRows(t, m, "MovieVideo", "VideoID = ?", vidID); n != 0 {
			t.Errorf("MovieVideo rows = %d, want 0", n)
		}
	})

	t.Run("movie-edition", func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		movieID, _ := createMovieRow(t, m, "purge-med", nil)
		var medExtra string
		if err := m.WithTxRW(func(tx *TxRW) error {
			med, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
				Title: "x", Label: "Extended", Slug: "ext",
				MovieID: movieID, ReleaseDate: "",
			})
			if err != nil {
				return err
			}
			medExtra = med.ID
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		vidID := createVideoRow(t, m, "pmed.mkv", "", nil)
		if err := m.WithTxRW(func(tx *TxRW) error {
			_, err := tx.q.MovieVideoCreate(ctx, schema.MovieVideoCreateParams{
				MovieEditionID: medExtra, VideoID: vidID,
			})
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, medExtra)
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Purge(ctx, medExtra)
		}); err != nil {
			t.Fatalf("purge: %v", err)
		}
		if n := countRows(t, m, "MovieEdition", "ID = ?", medExtra); n != 0 {
			t.Errorf("MovieEdition rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Video", "ID = ?", vidID); n != 0 {
			t.Errorf("Video rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Movie", "ID = ?", movieID); n != 1 {
			t.Errorf("Movie row count = %d, want 1 (parent alive)", n)
		}
	})

	t.Run("series-edition", func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		srID, _, sedExtra, snID, epID, vidID := createSeriesRow(t, m, "p-sed", "PSed")
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, sedExtra)
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Purge(ctx, sedExtra)
		}); err != nil {
			t.Fatalf("purge: %v", err)
		}
		if n := countRows(t, m, "SeriesEdition", "ID = ?", sedExtra); n != 0 {
			t.Errorf("SeriesEdition rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Season", "ID = ?", snID); n != 0 {
			t.Errorf("Season rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Episode", "ID = ?", epID); n != 0 {
			t.Errorf("Episode rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Video", "ID = ?", vidID); n != 0 {
			t.Errorf("Video rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Series", "ID = ?", srID); n != 1 {
			t.Errorf("Series rows = %d, want 1", n)
		}
	})

	t.Run("season", func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		_, _, sedExtra, snID, epID, vidID := createSeriesRow(t, m, "p-season", "PSeason")
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, snID)
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Purge(ctx, snID)
		}); err != nil {
			t.Fatalf("purge: %v", err)
		}
		if n := countRows(t, m, "Season", "ID = ?", snID); n != 0 {
			t.Errorf("Season rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Episode", "ID = ?", epID); n != 0 {
			t.Errorf("Episode rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Video", "ID = ?", vidID); n != 0 {
			t.Errorf("Video rows = %d, want 0", n)
		}
		if n := countRows(t, m, "SeriesEdition", "ID = ?", sedExtra); n != 1 {
			t.Errorf("SeriesEdition rows = %d, want 1", n)
		}
	})

	t.Run("episode", func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		_, _, _, snID, epID, vidID := createSeriesRow(t, m, "p-ep", "PEp")
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, epID)
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Purge(ctx, epID)
		}); err != nil {
			t.Fatalf("purge: %v", err)
		}
		if n := countRows(t, m, "Episode", "ID = ?", epID); n != 0 {
			t.Errorf("Episode rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Video", "ID = ?", vidID); n != 0 {
			t.Errorf("Video rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Season", "ID = ?", snID); n != 1 {
			t.Errorf("Season rows = %d, want 1", n)
		}
	})

	t.Run("collection", func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)
		colID := createCollectionRow(t, m, "p-col", "PCol")
		movieID, _ := createMovieRow(t, m, "p-col-mv", nil)
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.q.CollectionMovieAdd(ctx, schema.CollectionMovieAddParams{
				CollectionID: colID, MovieID: movieID,
			})
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, colID)
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Purge(ctx, colID)
		}); err != nil {
			t.Fatalf("purge: %v", err)
		}
		if n := countRows(t, m, "Collection", "ID = ?", colID); n != 0 {
			t.Errorf("Collection rows = %d, want 0", n)
		}
		if n := countRows(t, m, "CollectionMovie", "CollectionID = ?", colID); n != 0 {
			t.Errorf("CollectionMovie rows = %d, want 0", n)
		}
		if n := countRows(t, m, "Movie", "ID = ?", movieID); n != 1 {
			t.Errorf("Movie rows = %d, want 1 (member preserved)", n)
		}
	})
}

// TestPurgeEmitsPurgeEvent covers section 7.7.
func TestPurgeEmitsPurgeEvent(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	movieID, _ := createMovieRow(t, m, "purge-ev", nil)
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, movieID)
	}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, m, func() {
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Purge(ctx, movieID)
		}); err != nil {
			t.Fatal(err)
		}
	})
	var count int
	for _, ev := range events {
		if ev.Type == EventPurge {
			count++
			if ev.ID != movieID {
				t.Errorf("EventPurge ID = %q, want %q", ev.ID, movieID)
			}
			if ev.TrashKind != TrashKindMovie {
				t.Errorf("EventPurge TrashKind = %v, want TrashKindMovie", ev.TrashKind)
			}
		}
	}
	if count != 1 {
		t.Errorf("EventPurge count = %d, want 1", count)
	}
}

// TestPurgeOnCascadeTrashed covers Purge on a cascade-trashed ID.
// The Trash table tracks CascadeOf, so Purge rejects cascade-trashed
// items the same way Restore does.
func TestPurgeOnCascadeTrashed(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID, _, _, _, epID, _ := createSeriesRow(t, m, "purge-cascade", "PCascade")
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, srID)
	}); err != nil {
		t.Fatal(err)
	}
	err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Purge(ctx, epID)
	})
	if !errors.Is(err, ErrCascadeTrashed) {
		t.Fatalf("purge cascade-trashed episode: err = %v, want ErrCascadeTrashed", err)
	}
}

// TestPurgeSiblingSEd covers section 7.10: Purge one of two
// trashed SeriesEditions; the other SEd's rows still exist.
func TestPurgeSiblingSEd(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	var sed1, sed2 string
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "sib-purge-" + srID, Title: "SibPurge",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		a, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "A", Slug: "a", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sed1 = a.ID
		b, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "B", Slug: "b", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sed2 = b.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := m.WithTxRW(func(tx *TxRW) error {
		if err := tx.Trash(ctx, sed1); err != nil {
			return err
		}
		return tx.Trash(ctx, sed2)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Purge(ctx, sed1)
	}); err != nil {
		t.Fatalf("purge sed1: %v", err)
	}
	if n := countRows(t, m, "SeriesEdition", "ID = ?", sed1); n != 0 {
		t.Errorf("sed1 rows = %d, want 0", n)
	}
	if n := countRows(t, m, "SeriesEdition", "ID = ?", sed2); n != 1 {
		t.Errorf("sed2 rows = %d, want 1 (still trashed)", n)
	}
}

// TestTrashPurgeLoopAgedTrashPurged covers section 8.1: trashed
// rows older than trashRetention are hard-deleted when the periodic
// purge fires.
//
// Note: synctest bubbles require every spawned goroutine to exit
// before the bubble returns, and purgeTrashLoop has no shutdown
// signal. Rather than spawn the real loop, these tests advance
// virtual time and call purgeTrashOnce directly — the threshold
// computation (time.Now - trashRetention) is the interesting bit.
func TestTrashPurgeLoopAgedTrashPurged(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)

		vidID := createVideoRow(t, m, "aged.mkv", "", nil)
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, vidID)
		}); err != nil {
			t.Fatal(err)
		}

		time.Sleep(trashRetention + 2*time.Hour)
		if err := m.purgeTrashOnce(ctx); err != nil {
			t.Fatal(err)
		}

		if n := countRows(t, m, "Video", "ID = ?", vidID); n != 0 {
			t.Errorf("Video rows = %d, want 0 (aged video purged)", n)
		}
	})
}

// TestTrashPurgeLoopRecentTrashSkipped covers section 8.2.
func TestTrashPurgeLoopRecentTrashSkipped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)

		vidID := createVideoRow(t, m, "fresh.mkv", "", nil)
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, vidID)
		}); err != nil {
			t.Fatal(err)
		}

		time.Sleep(trashRetention - 1*time.Hour)
		if err := m.purgeTrashOnce(ctx); err != nil {
			t.Fatal(err)
		}

		if n := countRows(t, m, "Video", "ID = ?", vidID); n != 1 {
			t.Errorf("Video rows = %d, want 1 (still within retention)", n)
		}
	})
}

// TestTrashPurgeLoopMixedAges covers section 8.3: two videos trashed
// at different wall-clock times; only the aged one purges.
func TestTrashPurgeLoopMixedAges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		m := newTestModel(t)

		vidOld := createVideoRow(t, m, "old.mkv", "", nil)
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, vidOld)
		}); err != nil {
			t.Fatal(err)
		}

		time.Sleep(trashRetention + 2*time.Hour)
		vidNew := createVideoRow(t, m, "new.mkv", "", nil)
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, vidNew)
		}); err != nil {
			t.Fatal(err)
		}
		if err := m.purgeTrashOnce(ctx); err != nil {
			t.Fatal(err)
		}

		if n := countRows(t, m, "Video", "ID = ?", vidOld); n != 0 {
			t.Errorf("old video rows = %d, want 0 (should purge)", n)
		}
		if n := countRows(t, m, "Video", "ID = ?", vidNew); n != 1 {
			t.Errorf("new video rows = %d, want 1 (still within retention)", n)
		}
	})
}

// TestTrashSortKeyBumpCollision covers section 10.1: trash middle ep,
// add a new ep taking the SortKey slot, then restore the trashed ep —
// the two-phase negate trick must handle the collision.
func TestTrashSortKeyBumpCollision(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	var sedID, snID string
	var epIDs []string
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "sk-coll-" + srID, Title: "SkColl",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "S1", Number: 1,
		})
		if err != nil {
			return err
		}
		snID = sn.ID
		for i := int64(1); i <= 3; i++ {
			ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
				Title: fmt.Sprintf("Ep %d", i), Type: "regular", Runtime: 30,
			})
			if err != nil {
				return err
			}
			epIDs = append(epIDs, ep.ID)
			if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
				EditionID: sedID, SeasonID: snID, EpisodeID: ep.ID,
				SortKey: i, Label: fmt.Sprintf("%d", i), Number: i,
				Slug: fmt.Sprintf("ep-%d", i),
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Trash middle episode (SortKey 2).
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, epIDs[1])
	}); err != nil {
		t.Fatal(err)
	}

	// Add a new episode that occupies SortKey 2 (renumberSeason).
	var newEp string
	if err := m.WithTxRW(func(tx *TxRW) error {
		ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "New", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		newEp = ep.ID
		return tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: snID, EpisodeID: newEp,
			SortKey: 2, Label: "2", Number: 2, Slug: "ep-new",
		})
	}); err != nil {
		t.Fatal(err)
	}

	// Restore the trashed episode. Its junction had SortKey 2, so
	// the bump must shift the new ep (and ep 3) out of the way.
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Restore(ctx, epIDs[1])
	}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := m.WithTxR(func(tx *TxR) error {
		sneps, err := tx.q.SeasonEpisodeListBySeasonID(ctx, snID)
		if err != nil {
			return err
		}
		if len(sneps) != 4 {
			t.Fatalf("live junctions = %d, want 4", len(sneps))
		}
		order := []string{epIDs[0], epIDs[1], newEp, epIDs[2]}
		for i, want := range order {
			if sneps[i].EpisodeID != want {
				t.Errorf("position %d: got %s, want %s", i, sneps[i].EpisodeID, want)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashEventCounts covers section 11.1: trashing a Series with 2
// episodes + videos emits exactly one EventTrash and one
// EventTrashCascade (listing all descendants).
func TestTrashEventCounts(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	srID := "sr" + flurry.NewID()
	var sedID, snID string
	var epIDs, vidIDs []string
	if err := m.WithTxRW(func(tx *TxRW) error {
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "evt-" + srID, Title: "Evt",
			Status: "Running", PremieredOn: "2020-01-01",
		}); err != nil {
			return err
		}
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "S1", Number: 1,
		})
		if err != nil {
			return err
		}
		snID = sn.ID
		for i := int64(1); i <= 2; i++ {
			ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
				Title: fmt.Sprintf("Ep %d", i), Type: "regular", Runtime: 30,
			})
			if err != nil {
				return err
			}
			epIDs = append(epIDs, ep.ID)
			if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
				EditionID: sedID, SeasonID: snID, EpisodeID: ep.ID,
				SortKey: i, Label: fmt.Sprintf("%d", i), Number: i,
				Slug: fmt.Sprintf("ep-%d", i),
			}); err != nil {
				return err
			}
			v, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{
				Name: fmt.Sprintf("v%d.mkv", i),
			})
			if err != nil {
				return err
			}
			vidIDs = append(vidIDs, v.ID)
			if _, err := tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
				EpisodeID: ep.ID, VideoID: v.ID,
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	events := collectEvents(t, m, func() {
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Trash(ctx, srID)
		}); err != nil {
			t.Fatal(err)
		}
	})
	var trashCount, cascadeCount int
	var cascade *Event
	for _, ev := range events {
		switch ev.Type {
		case EventTrash:
			if ev.ID == srID {
				trashCount++
			}
		case EventTrashCascade:
			if ev.ID == srID {
				cascadeCount++
				cascade = ev
			}
		}
	}
	if trashCount != 1 {
		t.Errorf("EventTrash count = %d, want 1", trashCount)
	}
	if cascadeCount != 1 {
		t.Fatalf("EventTrashCascade count = %d, want 1", cascadeCount)
	}
	ids := map[string]bool{}
	for _, it := range cascade.TrashItems {
		ids[it.ID] = true
	}
	for _, want := range append(append([]string{}, epIDs...), vidIDs...) {
		if !ids[want] {
			t.Errorf("cascade items missing %q", want)
		}
	}

	events = collectEvents(t, m, func() {
		if err := m.WithTxRW(func(tx *TxRW) error {
			return tx.Restore(ctx, srID)
		}); err != nil {
			t.Fatal(err)
		}
	})
	var restoreCount int
	var restoreEv *Event
	for _, ev := range events {
		if ev.Type == EventRestore && ev.ID == srID {
			restoreCount++
			restoreEv = ev
		}
	}
	if restoreCount != 1 {
		t.Fatalf("EventRestore count = %d, want 1", restoreCount)
	}
	if len(restoreEv.TrashItems) == 0 {
		t.Errorf("restore TrashItems is empty; want populated with cascade items")
	}
}

// TestTrashCASBlobsRemovedByPurge covers explicit-Purge CAS cleanup:
// Trash a Video with an OriginalKey and a rendition, Purge, assert
// both blobs are removed.
func TestTrashCASBlobsRemovedByPurge(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)

	origKey, err := m.store.Copy(strings.NewReader("orig"))
	if err != nil {
		t.Fatal(err)
	}
	rendKey, err := m.store.Copy(strings.NewReader("rend"))
	if err != nil {
		t.Fatal(err)
	}
	vidID := createVideoRow(t, m, "p-cas.mkv", origKey, []string{rendKey})

	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Trash(ctx, vidID)
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.WithTxRW(func(tx *TxRW) error {
		return tx.Purge(ctx, vidID)
	}); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if casBlobExists(t, m, origKey) {
		t.Errorf("original blob %q still exists", origKey)
	}
	if casBlobExists(t, m, rendKey) {
		t.Errorf("rendition blob %q still exists", rendKey)
	}
}

// createSeriesRow builds a full Series chain: Series (with Slug),
// default SeriesEdition, one non-default SeriesEdition, one Season
// (under the non-default edition), one Episode, one Video, and the
// linking SeasonEpisode / EpisodeVideo junctions.
func createSeriesRow(t *testing.T, m *Model, slug, title string) (
	srID, sedDefaultID, sedID, snID, epID, vidID string,
) {
	t.Helper()
	ctx := context.Background()
	err := m.WithTxRW(func(tx *TxRW) error {
		srID = "sr" + flurry.NewID()
		sr, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: slug, Title: title,
			Status: "Running", PremieredOn: "2020-01-01",
		})
		if err != nil {
			return err
		}
		srID = sr.ID
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: slug, Kind: "series", Target: srID,
		}); err != nil {
			return err
		}
		sedDefault, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Default", Slug: "", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedDefaultID = sedDefault.ID
		sed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{
			Label: "Original", Slug: "original", SeriesID: srID, Summary: "",
		})
		if err != nil {
			return err
		}
		sedID = sed.ID
		sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{
			EditionID: sedID, SortKey: "01", Title: "Season 1", Number: 1,
		})
		if err != nil {
			return err
		}
		snID = sn.ID
		ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Title: "Pilot", Type: "regular", Runtime: 30,
		})
		if err != nil {
			return err
		}
		epID = ep.ID
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: sedID, SeasonID: snID, EpisodeID: epID,
			SortKey: 1, Label: "1", Number: 1, Slug: "s1e1-pilot",
		}); err != nil {
			return err
		}
		vid, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{Name: "pilot.mkv"})
		if err != nil {
			return err
		}
		vidID = vid.ID
		_, err = tx.q.EpisodeVideoCreate(ctx, schema.EpisodeVideoCreateParams{
			EpisodeID: epID, VideoID: vidID,
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return srID, sedDefaultID, sedID, snID, epID, vidID
}

// createCollectionRow creates a Collection with the given slug/title
// plus matching Slug table row.
func createCollectionRow(t *testing.T, m *Model, slug, title string) string {
	t.Helper()
	ctx := context.Background()
	var colID string
	err := m.WithTxRW(func(tx *TxRW) error {
		col, err := tx.q.CollectionCreate(ctx, schema.CollectionCreateParams{
			Slug: slug, Title: title,
		})
		if err != nil {
			return err
		}
		colID = col.ID
		return tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: slug, Kind: "collection", Target: colID,
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	return colID
}

// assertLive fails if the row identified by id is soft-deleted.
func assertLive(t *testing.T, m *Model, id string) {
	t.Helper()
	ctx := context.Background()
	if err := m.WithTxR(func(tx *TxR) error {
		st, err := tx.trashState(ctx, id)
		if err != nil {
			return err
		}
		if !st.live() {
			t.Errorf("%s still trashed (cascadeOf=%q)", id, st.cascadeOf)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// assertTrashed fails if the row is live, or (when wantCascadeOf is
// non-empty) if its CascadeOf doesn't match. Pass "" to assert a
// direct trash (CascadeOf empty).
func assertTrashed(t *testing.T, m *Model, id, wantCascadeOf string) {
	t.Helper()
	ctx := context.Background()
	if err := m.WithTxR(func(tx *TxR) error {
		st, err := tx.trashState(ctx, id)
		if err != nil {
			return err
		}
		if st.live() {
			t.Errorf("%s live, want trashed", id)
			return nil
		}
		if st.cascadeOf != wantCascadeOf {
			t.Errorf("%s cascadeOf = %q, want %q", id, st.cascadeOf, wantCascadeOf)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// collectEvents subscribes, runs fn, then returns every event emitted.
func collectEvents(t *testing.T, m *Model, fn func()) []*Event {
	t.Helper()
	sink := subscribeEvents(context.Background(), m)
	fn()
	return sink.drain()
}

// countRows returns the count of rows matching `where` (may be empty)
// in the given table.
func countRows(t *testing.T, m *Model, table, where string, args ...any) int {
	t.Helper()
	ctx := context.Background()
	q := "SELECT COUNT(*) FROM " + table
	if where != "" {
		q += " WHERE " + where
	}
	var n int
	if err := m.WithTxR(func(tx *TxR) error {
		return tx.tx.QueryRowContext(ctx, q, args...).Scan(&n)
	}); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// casBlobExists reports whether the CAS store still has the given key.
func casBlobExists(t *testing.T, m *Model, key string) bool {
	t.Helper()
	if key == "" {
		return false
	}
	f, err := m.store.Open(key)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
