package model

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"slices"
	"sync"
	"time"

	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
)

// An Upload describes a video upload in flight.
type Upload struct {
	TargetID string  // the episode or movie edition the upload attaches to
	Name     string  // the uploaded file's name
	Frac     float64 // the fraction received so far, in [0,1]
}

// Uploads lists the video uploads currently in flight,
// oldest first.
func (m *Model) Uploads() []Upload {
	m.uploadMu.Lock()
	defer m.uploadMu.Unlock()
	var a []Upload
	for _, u := range m.uploads {
		a = append(a, Upload{
			TargetID: u.targetID,
			Name:     u.name,
			Frac:     u.frac(),
		})
	}
	return a
}

// Uploads lists the video uploads currently in flight,
// oldest first.
func (tx *TxR) Uploads() []Upload { return tx.m.Uploads() }

// An upload tracks one in-flight video upload's received bytes.
// It emits events periodically as bytes arrive.
type upload struct {
	m        *Model
	targetID string
	name     string
	size     int64
	r        io.Reader

	mu      sync.Mutex
	count   int64
	lastAnn time.Time // last announcement
}

func (u *upload) frac() float64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.size <= 0 {
		return 0
	}
	return min(1, float64(u.count)/float64(u.size))
}

func (u *upload) Read(p []byte) (int, error) {
	n, err := u.r.Read(p)
	u.mu.Lock()
	u.count += int64(n)
	announce := time.Since(u.lastAnn) >= 500*time.Millisecond
	if announce {
		u.lastAnn = time.Now()
	}
	u.mu.Unlock()
	if announce {
		u.m.emit()
	}
	return n, err
}

// uploadBegin registers an upload around r and announces it.
func (m *Model) uploadBegin(targetID, name string, size int64, r io.Reader) *upload {
	u := &upload{m: m, targetID: targetID, name: name, size: size, r: r}
	m.uploadMu.Lock()
	m.uploads = append(m.uploads, u)
	m.uploadMu.Unlock()
	m.emit()
	return u
}

// uploadEnd removes u from the registry and announces its end.
func (m *Model) uploadEnd(u *upload) {
	m.uploadMu.Lock()
	m.uploads = slices.DeleteFunc(m.uploads, func(v *upload) bool { return v == u })
	m.uploadMu.Unlock()
	m.emit()
}

// VideoUploadCreate streams r into the blob store and registers it as
// a new Video not associated with any Download. Exactly one target
// must be specified: an episode (epID) or a movie edition (medID).
// While the upload is in flight it is listed by Uploads, with its
// progress measured against size (the caller's estimate of the
// total; non-positive means unknown).
//
// If the uploaded bytes match an existing live Video's ContentHash,
// the freshly-copied blob is discarded and the existing Video is
// attached to the target junction instead — mirroring the
// duplicate-merge path in taskIngest.
//
// On a fresh upload the source is probed before the write tx, then the
// video create, target attach, and rendition rows commit together,
// joining the same downstream encoding path used by torrent ingest.
func (m *Model) VideoUploadCreate(
	ctx context.Context,
	r io.Reader,
	name string,
	size int64,
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
	var targetID string
	switch {
	case medID != nil && epID == nil:
		targetID = *medID
	case medID == nil && epID != nil:
		targetID = *epID
	default:
		return vid, &ValidationError{
			Op:  "target",
			Err: fmt.Errorf("must specify exactly one of med-id or ep-id"),
		}
	}

	u := m.uploadBegin(targetID, base, size, r)
	defer m.uploadEnd(u)
	key, sum, err := m.copyToStoreHashed(u)
	if err != nil {
		return vid, err
	}

	// Probe before the write tx so the video create, the target attach,
	// and the rendition rows commit together; a duplicate just wastes
	// the probe.
	probe, err := m.probeVideo(ctx, key)
	if err != nil {
		return vid, err
	}

	var merged bool
	err = m.WithTxRW(ctx, func(tx *TxRW) error {
		dups, err := tx.q.VideoListByContentHash(sum)
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
			return tx.attachUploadedVideo(vid.ID, medID, epID)
		}
		v, err := tx.q.VideoCreate(schema.VideoCreateParams{Name: base})
		if err != nil {
			return err
		}
		v, err = tx.q.VideoUpdateOriginalKey(schema.VideoUpdateOriginalKeyParams{
			ID:          v.ID,
			OriginalKey: key,
			ContentHash: sum,
		})

		if err != nil {
			return err
		}
		vid = v
		if err := tx.attachUploadedVideo(v.ID, medID, epID); err != nil {
			return err
		}
		return tx.planAndCreateRenditions(v, probe)
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
	return vid, nil
}

// attachUploadedVideo wires a Video into either an Episode or a
// MovieEdition. Uses ensure-style inserts so a re-upload of an
// already-attached file is idempotent.
func (tx *TxRW) attachUploadedVideo(videoID string, medID, epID *string) error {
	switch {
	case epID != nil:
		return tx.q.EpisodeVideoEnsure(schema.EpisodeVideoEnsureParams{
			EpisodeID: *epID,
			VideoID:   videoID,
		})

	case medID != nil:
		return tx.q.MovieVideoEnsure(schema.MovieVideoEnsureParams{
			MovieEditionID: *medID,
			VideoID:        videoID,
		})

	}
	return fmt.Errorf("attachUploadedVideo: no target")
}
