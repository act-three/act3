package model

import (
	"fmt"
	"io"
	"log/slog"
	"path"

	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
)

// VideoUploadCreate streams r into the CAS store and registers it as
// a new Video not associated with any Download. Exactly one target
// must be specified: an episode (epID) or a movie edition (medID).
//
// If the uploaded bytes match an existing live Video's ContentHash,
// the freshly-copied blob is discarded and the existing Video is
// attached to the target junction instead — mirroring the
// duplicate-merge path in taskIngest.
//
// On a fresh upload, planAndCreateRenditions is called outside the
// write transaction to kick off the HLS rendition pipeline, joining
// the same downstream encoding path used by torrent ingest.
func (m *Model) VideoUploadCreate(
	ctx Context,
	r io.Reader,
	name string,
	medID, epID *string,
) (vid schema.Video, err error) {
	defer errorfmt.Handlef("VideoUploadCreate: %w", &err)

	base := path.Base(name)
	if !hasVideoExtension(base) {
		return vid, &ValidationError{
			Op:  "filename",
			Err: fmt.Errorf("%q lacks a recognized video extension", base),
		}
	}
	switch {
	case medID != nil && epID == nil:
	case medID == nil && epID != nil:
	default:
		return vid, &ValidationError{
			Op:  "target",
			Err: fmt.Errorf("must specify exactly one of med-id or ep-id"),
		}
	}

	key, sum, err := m.copyToStoreHashed(r)
	if err != nil {
		return vid, err
	}

	var merged bool
	err = m.WithTxRW(func(tx *TxRW) error {
		dups, err := tx.q.VideoListByContentHash(ctx, sum)
		if err != nil {
			return err
		}
		if len(dups) > 0 {
			merged = true
			vid = dups[0]
			tx.onCommit(func() {
				if err := m.store.Remove(key); err != nil {
					slog.Error("remove upload duplicate bytes",
						"key", key, "err", err)
				}
			})
			return tx.attachUploadedVideo(ctx, vid.ID, medID, epID)
		}
		v, err := tx.q.VideoCreate(ctx, schema.VideoCreateParams{Name: base})
		if err != nil {
			return err
		}
		v, err = tx.q.VideoUpdateOriginalKey(ctx, schema.VideoUpdateOriginalKeyParams{
			ID:          v.ID,
			OriginalKey: key,
			ContentHash: sum,
		})
		if err != nil {
			return err
		}
		vid = v
		return tx.attachUploadedVideo(ctx, v.ID, medID, epID)
	})
	if err != nil {
		return vid, err
	}
	if merged {
		return vid, nil
	}

	if epID != nil {
		m.prog.AddEdge(*epID, vid.ID)
	}
	if medID != nil {
		m.prog.AddEdge(*medID, vid.ID)
	}
	m.prog.Open(vid.ID, vid.Name, "Probing")
	err = m.WithTxR(func(tx *TxR) error {
		return tx.planAndCreateRenditions(ctx, vid)
	})
	return vid, err
}

// attachUploadedVideo wires a Video into either an Episode or a
// MovieEdition. Uses ensure-style inserts so a re-upload of an
// already-attached file is idempotent.
func (tx *TxRW) attachUploadedVideo(ctx Context, videoID string, medID, epID *string) error {
	switch {
	case epID != nil:
		return tx.q.EpisodeVideoEnsure(ctx, schema.EpisodeVideoEnsureParams{
			EpisodeID: *epID,
			VideoID:   videoID,
		})
	case medID != nil:
		return tx.q.MovieVideoEnsure(ctx, schema.MovieVideoEnsureParams{
			MovieEditionID: *medID,
			VideoID:        videoID,
		})
	}
	return fmt.Errorf("attachUploadedVideo: no target")
}
