package model

import (
	"errors"

	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
)

// ErrActiveVideoLocked is returned when an admin attempts a destructive
// op (delete, re-import, re-encode) on the active video of a work that
// has another playable video. The admin must first switch the active
// video.
var ErrActiveVideoLocked = errors.New("active video is locked while another playable video exists; switch active first")

// ensureActiveVideoForVideoID promotes a playable video to Active for
// every work it's attached to that doesn't already have an Active
// junction. Run after a video transitions to playable (MVPlaylist set)
// or after a junction set changes.
func (tx *TxRW) ensureActiveVideoForVideoID(ctx Context, videoID string) (err error) {
	defer errorfmt.Handlef("ensureActiveVideoForVideoID: %w", &err)

	epIDs, err := tx.q.EpisodeVideoDistinctEpisodesByVideo(ctx, videoID)
	if err != nil {
		return err
	}
	for _, epID := range epIDs {
		if err := tx.q.EpisodeVideoActivePromote(ctx, epID); err != nil {
			return err
		}
	}
	medIDs, err := tx.q.MovieVideoDistinctEditionsByVideo(ctx, videoID)
	if err != nil {
		return err
	}
	for _, medID := range medIDs {
		if err := tx.q.MovieVideoActivePromote(ctx, medID); err != nil {
			return err
		}
	}
	return nil
}

// EpisodeVideoSetActive marks (episodeID, videoID) as the active
// junction for the episode. The target video must be live, attached,
// and playable (MVPlaylist non-empty).
func (tx *TxRW) EpisodeVideoSetActive(ctx Context, episodeID, videoID string) (err error) {
	defer errorfmt.Handlef("EpisodeVideoSetActive: %w", &err)

	if err := tx.q.EpisodeVideoSetInactiveByEpisodeID(ctx, episodeID); err != nil {
		return err
	}
	n, err := tx.q.EpisodeVideoMarkActive(ctx, schema.EpisodeVideoMarkActiveParams{
		EpisodeID: episodeID,
		VideoID:   videoID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("video is not attached, not live, or not playable")
	}
	return nil
}

// MovieVideoSetActive marks (medID, videoID) as the active junction
// for the movie edition. The target video must be live, attached, and
// playable.
func (tx *TxRW) MovieVideoSetActive(ctx Context, medID, videoID string) (err error) {
	defer errorfmt.Handlef("MovieVideoSetActive: %w", &err)

	if err := tx.q.MovieVideoSetInactiveByMovieEditionID(ctx, medID); err != nil {
		return err
	}
	n, err := tx.q.MovieVideoMarkActive(ctx, schema.MovieVideoMarkActiveParams{
		MovieEditionID: medID,
		VideoID:        videoID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("video is not attached, not live, or not playable")
	}
	return nil
}

// guardActiveVideo rejects a destructive op on videoID when it is the
// Active junction for any work that has another playable video. Used
// by delete, re-import, and re-encode flows.
func (tx *TxRW) guardActiveVideo(ctx Context, videoID string) (err error) {
	defer errorfmt.Handlef("guardActiveVideo: %w", &err)

	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, videoID)
	if err != nil {
		return err
	}
	for _, ev := range evs {
		if ev.DeletedAt != nil || ev.Active == 0 {
			continue
		}
		n, err := tx.q.EpisodeVideoCountPlayable(ctx, ev.EpisodeID)
		if err != nil {
			return err
		}
		if n > 1 {
			return ErrActiveVideoLocked
		}
	}
	mvs, err := tx.q.MovieVideoListByVideoID(ctx, videoID)
	if err != nil {
		return err
	}
	for _, mv := range mvs {
		if mv.DeletedAt != nil || mv.Active == 0 {
			continue
		}
		n, err := tx.q.MovieVideoCountPlayable(ctx, mv.MovieEditionID)
		if err != nil {
			return err
		}
		if n > 1 {
			return ErrActiveVideoLocked
		}
	}
	return nil
}
