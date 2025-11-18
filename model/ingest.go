package model

import (
	"log/slog"
	"os"
	"time"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/video/ffmpeg"
)

func (tx *TxR) taskIngest(ctx Context, args []string) func(*TxRW) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return taskError(err)
	}
	hash, err := tx.m.store.Link(args[1])
	if err != nil {
		return taskError(err)
	}

	// Get source duration for progress calculation.
	// Open a second handle so ffprobe can seek independently of src.
	f, err := tx.m.store.Open(hash)
	if err != nil {
		return taskError(err)
	}
	defer f.Close()
	dur, err := ffmpeg.ProbeDuration(ctx, f)
	if err != nil {
		return taskError(err)
	}

	return func(tx *TxRW) error {
		vid, err = tx.q.VideoUpdateOriginalHash(ctx, schema.VideoUpdateOriginalHashParams{
			ID:           vid.ID,
			OriginalHash: hash,
		})
		if err != nil {
			return err
		}
		rendID := "demorendition" + vid.ID
		tx.m.prog.addRendition(vid.ID, rendID, "Transcoding Demo", dur)
		tx.addTask(ctx, taskIngestDemo, vid.ID, rendID)
		// TODO(april): tx.addTask for generating renditions
		return nil
	}
}

func (tx *TxR) taskIngestDemo(ctx Context, args []string) func(*TxRW) error {
	rendID := args[1]
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return taskError(err)
	}

	src, err := tx.m.store.Open(vid.OriginalHash)
	if err != nil {
		return taskError(err)
	}

	defer tx.m.prog.clearRendition(vid.ID, rendID)
	hash, err := tx.m.store.CreateFunc(func(dst *os.File) error {
		slog.DebugContext(ctx, "create", "name", dst.Name())
		return ffmpeg.Remux(ctx, src, dst, func(d time.Duration) {
			tx.m.prog.updateRendition(rendID, d)
		})
	})
	if err != nil {
		return taskError(err)
	}

	return func(tx *TxRW) error {
		_, err := tx.q.RenditionForStreamingCreate(ctx, schema.RenditionForStreamingCreateParams{
			VideoID: vid.ID,
			Hash:    hash,
		})
		return err
	}
}
