package model

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"kr.dev/errorfmt"
	"lukechampine.com/blake3"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/priority"
	"ily.dev/act3/video"
	"ily.dev/act3/video/ffmpeg"
)

// copyToStoreHashed streams r into the CAS store and returns the
// storage key alongside a blake3-256 digest of the bytes. Used by
// ingest-style paths to enable duplicate-content detection.
func (m *Model) copyToStoreHashed(r io.Reader) (key string, sum []byte, err error) {
	h := blake3.New(32, nil)
	key, err = m.store.Copy(io.TeeReader(r, h))
	if err != nil {
		return "", nil, err
	}
	return key, h.Sum(nil), nil
}

// openIngestSource opens name inside localDir for reading, rejecting
// symlinks that escape the root, absolute or ".." paths, and any
// source that is not a regular file. The returned file's Stat and
// read operations refer to the same inode, so there is no Stat/Open
// TOCTOU. Callers are responsible for Close.
func openIngestSource(localDir, name string) (*os.File, error) {
	f, err := os.OpenInRoot(localDir, name)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		f.Close()
		return nil, fmt.Errorf("ingest: %q is not a regular file (mode %s)", name, fi.Mode())
	}
	return f, nil
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func (tx *TxR) taskIngest(ctx Context, args []string) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}
	if vid.OriginalKey != "" {
		return nil
	}

	// Resolve the download directory.
	// Transmission may still be copying the file to its final
	// location after reporting the download as complete.
	// If the probe fails or the file doesn't exist yet,
	// re-queue and check again shortly.
	remoteDir := args[1]
	name := args[2]
	if !filepath.IsLocal(name) {
		return fmt.Errorf("refusing ingest of non-local path %q", name)
	}
	retryIngest := func() error {
		slog.InfoContext(ctx, "ingest: file not yet available, retrying",
			"vid", vid.ID, "dir", remoteDir, "name", name)
		return tx.m.WithTxRW(func(txw *TxRW) error {
			return txw.addTaskAfter(ctx, 5*time.Second, taskIngest, args...)
		})
	}
	localDir, err := tx.m.resolveDownloadDir(remoteDir, name)
	if err != nil {
		return retryIngest()
	}
	src, err := openIngestSource(localDir, name)
	if errors.Is(err, os.ErrNotExist) {
		return retryIngest()
	}
	if err != nil {
		return err
	}
	defer src.Close()

	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	tx.m.prog.Open(vid.ID, vid.Name, "Copying")
	key, sum, err := tx.m.copyToStoreHashed(src)
	if err != nil {
		tx.m.prog.Close(vid.ID, err)
		return err
	}

	merged := false
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		// If another live Video already has these exact bytes, merge
		// the current (freshly-copied) Video into it: re-point the
		// junctions, drop the duplicate bytes, hard-delete our row.
		dups, err := txw.q.VideoListByContentHash(ctx, sum)
		if err != nil {
			return err
		}
		for _, winner := range dups {
			if winner.ID == vid.ID {
				continue
			}
			merged = true
			return txw.mergeDuplicateVideo(ctx, vid, winner, key)
		}
		vid, err = txw.q.VideoUpdateOriginalKey(ctx, schema.VideoUpdateOriginalKeyParams{
			ID:          vid.ID,
			OriginalKey: key,
			ContentHash: sum,
		})
		return err
	})
	if err != nil {
		tx.m.prog.Close(vid.ID, err)
		return err
	}
	if merged {
		tx.m.prog.Close(vid.ID, nil)
		return nil
	}

	return tx.planAndCreateRenditions(ctx, vid)
}

// mergeDuplicateVideo is called when a freshly-copied video's content
// hash matches an already-live `winner` Video. It re-points all live
// junctions from `loser` to `winner`, hard-deletes the loser row, and
// schedules the newly-copied bytes (`loserKey`) for removal on commit.
func (tx *TxRW) mergeDuplicateVideo(ctx Context, loser schema.Video, winner schema.Video, loserKey string) (err error) {
	defer errorfmt.Handlef("merge %s into %s: %w", loser.ID, winner.ID, &err)
	// Revive any soft-deleted winner junctions that would otherwise
	// collide with live loser junctions on reassign. The loser's
	// live junction represents current user intent and supersedes
	// the older detach.
	if err := tx.q.EpisodeVideoRestoreForReassign(ctx, schema.EpisodeVideoRestoreForReassignParams{
		FromVideoID: loser.ID,
		ToVideoID:   winner.ID,
	}); err != nil {
		return err
	}
	if err := tx.q.EpisodeVideoReassign(ctx, schema.EpisodeVideoReassignParams{
		FromVideoID: loser.ID,
		ToVideoID:   winner.ID,
	}); err != nil {
		return err
	}
	if err := tx.q.EpisodeVideoDeleteByVideoID(ctx, loser.ID); err != nil {
		return err
	}
	if err := tx.q.MovieVideoRestoreForReassign(ctx, schema.MovieVideoRestoreForReassignParams{
		FromVideoID: loser.ID,
		ToVideoID:   winner.ID,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieVideoReassign(ctx, schema.MovieVideoReassignParams{
		FromVideoID: loser.ID,
		ToVideoID:   winner.ID,
	}); err != nil {
		return err
	}
	if err := tx.q.MovieVideoDeleteByVideoID(ctx, loser.ID); err != nil {
		return err
	}
	if loser.InfoHash != nil {
		if err := tx.bumpDownloadActivity(ctx, *loser.InfoHash); err != nil {
			return err
		}
	}
	if winner.InfoHash != nil && (loser.InfoHash == nil || *winner.InfoHash != *loser.InfoHash) {
		if err := tx.bumpDownloadActivity(ctx, *winner.InfoHash); err != nil {
			return err
		}
	}
	if err := tx.q.VideoHardDelete(ctx, loser.ID); err != nil {
		return err
	}
	tx.onCommit(func() {
		if err := tx.m.store.Remove(loserKey); err != nil {
			slog.Error("remove merged duplicate bytes",
				"key", loserKey, "err", err)
		}
	})
	return nil
}

// planAndCreateRenditions probes the source in CAS, plans a
// rendition ladder, creates the rendition DB records, and queues
// pass1. The caller must have already set vid.OriginalKey and
// opened progress tracking for vid.ID.
func (tx *TxR) planAndCreateRenditions(ctx Context, vid schema.Video) error {
	existing, err := tx.q.RenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		tx.m.prog.Close(vid.ID, nil)
		return nil
	}

	tx.m.prog.UpdateStatus(vid.ID, "Probing")
	f, err := tx.m.store.Open(vid.OriginalKey)
	if err != nil {
		tx.m.prog.Close(vid.ID, err)
		return err
	}
	defer f.Close()
	probe, err := ffmpeg.Probe(ctx, f)
	if err != nil {
		tx.m.prog.Close(vid.ID, err)
		return err
	}

	planned, err := video.PlanRenditions(probe)
	if err != nil {
		return err
	}
	if len(planned) == 0 {
		return fmt.Errorf("no video stream found in source")
	}

	var rendParams []schema.RenditionCreateParams
	for _, r := range planned {
		rendParams = append(rendParams, schema.RenditionCreateParams{
			VideoID:       vid.ID,
			Purpose:       "streaming",
			Remux:         boolToInt64(r.Remux),
			Codec:         r.Codec,
			TargetBitrate: r.TargetBitrate,
			MaxHeight:     int64(r.MaxHeight),
			MaxFPS:        int64(r.MaxFPS),
			CopyAudio:     boolToInt64(r.CopyAudio),
			SurroundAudio: boolToInt64(r.SurroundAudio),
			Priority:      int64(r.Priority),
		})
	}

	// Download rendition uses the same parameters as the best
	// streaming rendition, packaged as a plain MP4.
	best := planned[0]
	rendParams = append(rendParams, schema.RenditionCreateParams{
		VideoID:       vid.ID,
		Purpose:       "download",
		Remux:         boolToInt64(best.Remux),
		Codec:         best.Codec,
		TargetBitrate: best.TargetBitrate,
		MaxHeight:     int64(best.MaxHeight),
		MaxFPS:        int64(best.MaxFPS),
		CopyAudio:     boolToInt64(best.CopyAudio),
		SurroundAudio: boolToInt64(best.SurroundAudio),
		Priority:      int64(priority.EncodeDownload),
	})

	var atParams []schema.AudioTrackCreateParams
	for _, as := range probe.Audio {
		atParams = append(atParams, schema.AudioTrackCreateParams{
			VideoID:       vid.ID,
			StreamIndex:   int64(as.Index),
			Language:      as.Language,
			Title:         as.Title,
			Channels:      int64(as.Channels),
			ChannelLayout: as.ChannelLayout,
			Codec:         as.CodecName,
		})
	}

	tx.m.prog.UpdateStatus(vid.ID, "Queued")
	return tx.m.WithTxRW(func(txw *TxRW) error {
		err = txw.q.VideoUpdateProbe(ctx, schema.VideoUpdateProbeParams{
			ID:           vid.ID,
			Duration:     probe.Duration.Milliseconds(),
			OriginalType: probe.ContentType(),
		})
		if err != nil {
			return err
		}
		for _, at := range atParams {
			_, err = txw.q.AudioTrackCreate(ctx, at)
			if err != nil {
				return err
			}
		}
		for _, rp := range rendParams {
			_, err = txw.q.RenditionCreate(ctx, rp)
			if err != nil {
				return err
			}
		}
		txw.addTaskWithPriority(ctx, priority.Pass1, taskIngestPass1, vid.ID)
		return nil
	})
}

// taskIngestPass1 runs combined first-pass analysis for all reencode
// renditions of a video, saves the stats blobs to the DB, then queues
// per-rendition encode tasks.
func (tx *TxR) taskIngestPass1(ctx Context, args []string) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}

	// Re-derive episode→video edges for progress tracking.
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	tx.m.prog.Open(vid.ID, vid.Name, "Starting")

	renditions, err := tx.q.RenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	if len(renditions) == 0 {
		return fmt.Errorf("no renditions planned for video %s", vid.ID)
	}

	// Classify renditions: collect indices of those needing reencode.
	var encodeIdxs []int
	for i, r := range renditions {
		if r.Remux == 0 {
			encodeIdxs = append(encodeIdxs, i)
		}
	}

	// If there are reencode renditions, run pass 1.
	if len(encodeIdxs) > 0 {
		src, err := tx.m.store.Open(vid.OriginalKey)
		if err != nil {
			return err
		}
		defer src.Close()

		tx.m.prog.UpdateStatus(vid.ID, "Probing")
		probe, err := ffmpeg.Probe(ctx, src)
		if err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}

		dsts := make([]ffmpeg.EncodeParams, len(renditions))
		for _, i := range encodeIdxs {
			r := renditions[i]
			dsts[i] = ffmpeg.EncodeParams{
				Codec:         toFFmpegCodec(r.Codec),
				Bitrate:       r.TargetBitrate,
				MaxHeight:     int(r.MaxHeight),
				MaxFPS:        int(r.MaxFPS),
				Tag:           toVideoTag(r.Codec),
				CopyAudio:     r.CopyAudio != 0,
				SurroundAudio: r.SurroundAudio != 0,
			}
		}

		statsDir := tx.m.pass1StatsDir(vid.ID)
		if err := os.MkdirAll(statsDir, 0o777); err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}
		passlogs := make(map[int]string, len(encodeIdxs))
		for _, i := range encodeIdxs {
			passlogs[i] = filepath.Join(statsDir, renditions[i].ID)
		}

		tx.m.prog.UpdateStatus(vid.ID, fmt.Sprintf("Pass 1 (%d renditions)", len(encodeIdxs)))
		preset, err := ffmpeg.Pass1Combined(ctx, src, dsts, encodeIdxs, passlogs, probe.Duration,
			func(v float64) { tx.m.prog.Update(vid.ID, v) },
		)
		if err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}

		err = os.WriteFile(filepath.Join(statsDir, "preset"), []byte(preset), 0o666)
		if err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}
	}

	// Queue one encode task per rendition.
	tx.m.prog.UpdateStatus(vid.ID, "Queuing renditions")
	tx.m.prog.Close(vid.ID, nil)

	return tx.m.WithTxRW(func(txw *TxRW) error {
		for _, r := range renditions {
			ttype := taskIngestEncodeRend
			if r.Purpose == "download" {
				ttype = taskIngestEncodeDownloadRend
			}
			txw.addTaskWithPriority(ctx, r.Priority, ttype, vid.ID)
		}
		return nil
	})
}

// taskIngestEncodeRend picks the highest-priority unencoded rendition
// for a video, encodes it, and rebuilds the MV playlist.
func (tx *TxR) taskIngestEncodeRend(ctx Context, args []string) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}

	rfs, err := tx.q.RenditionNextUnencodedStreaming(ctx, vid.ID)
	if err == sql.ErrNoRows {
		// All renditions already encoded (race or duplicate task).
		return nil
	}
	if err != nil {
		return err
	}

	// Progress tracking: use rendition ID as key, link to video.
	progKey := "rfs-" + rfs.ID
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	tx.m.prog.AddEdge(vid.ID, progKey)
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	desc := rendDesc(rfs)
	tx.m.prog.Open(progKey, vid.Name, desc+": starting")

	src, err := tx.m.store.Open(vid.OriginalKey)
	if err != nil {
		return err
	}
	defer src.Close()

	tx.m.prog.UpdateStatus(progKey, desc+": probing")
	probe, err := ffmpeg.Probe(ctx, src)
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}

	dst := ffmpeg.EncodeParams{
		Remux:         rfs.Remux != 0,
		Codec:         toFFmpegCodec(rfs.Codec),
		Bitrate:       rfs.TargetBitrate,
		MaxHeight:     int(rfs.MaxHeight),
		MaxFPS:        int(rfs.MaxFPS),
		Tag:           toVideoTag(rfs.Codec),
		CopyAudio:     rfs.CopyAudio != 0,
		SurroundAudio: rfs.SurroundAudio != 0,
	}

	var playlist string
	hash, err := tx.m.store.CreateFunc(func(dstFile *os.File) error {
		dst.File = dstFile
		onProgress := func(v float64) { tx.m.prog.Update(progKey, v) }

		if rfs.Remux != 0 {
			tx.m.prog.UpdateStatus(progKey, desc+": remuxing")
			var err error
			playlist, err = ffmpeg.RemuxSingle(ctx, src, dst, probe.Duration, onProgress)
			return err
		}

		passlog := tx.m.pass1Passlog(vid.ID, rfs.ID)
		preset, err := tx.m.loadPass1Preset(vid.ID)
		if err != nil {
			return err
		}
		tx.m.prog.UpdateStatus(progKey, desc+": encoding")
		playlist, err = ffmpeg.Pass2Single(ctx, src, dst, passlog, preset, probe.Duration, onProgress)
		return err
	})
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}

	// Fixup media playlist: point segment references at the
	// rendition's stream URL. The URL carries the rendition ID
	// rather than the storage key so the handler can do a DB
	// lookup and serve with a pinned Content-Type.
	playlist = video.FixupMediaPlaylist(playlist, ffmpeg.MediaName(0), "/-/vid/"+rfs.ID+".mp4")

	tx.m.prog.UpdateStatus(progKey, desc+": saving")
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		_, err := txw.q.RenditionUpdateEncode(ctx,
			schema.RenditionUpdateEncodeParams{
				ID:       rfs.ID,
				Key:      hash,
				Playlist: playlist,
			},
		)
		if err != nil {
			return err
		}

		// Rebuild MV playlist from all completed renditions.
		return rebuildMVPlaylist(ctx, txw, vid, probe)
	})
	if err == nil && rfs.Remux == 0 {
		tx.m.removePass1Stats(vid.ID, rfs.ID)
	}
	tx.m.prog.Close(progKey, err)
	return err
}

// taskIngestEncodeDownloadRend encodes the download rendition for a
// video as a plain MP4 with faststart.
func (tx *TxR) taskIngestEncodeDownloadRend(ctx Context, args []string) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}

	rfd, err := tx.q.RenditionGetDownloadByVideoID(ctx, vid.ID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if rfd.Key != "" {
		return nil // already encoded
	}

	progKey := "rfd-" + rfd.ID
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	tx.m.prog.AddEdge(vid.ID, progKey)
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	tx.m.prog.Open(progKey, vid.Name, "download mp4: starting")

	src, err := tx.m.store.Open(vid.OriginalKey)
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}
	defer src.Close()

	duration := time.Duration(vid.Duration) * time.Millisecond
	dst := ffmpeg.EncodeParams{
		Remux:         rfd.Remux != 0,
		Codec:         toFFmpegCodec(rfd.Codec),
		Bitrate:       rfd.TargetBitrate,
		MaxHeight:     int(rfd.MaxHeight),
		MaxFPS:        int(rfd.MaxFPS),
		Tag:           toVideoTag(rfd.Codec),
		CopyAudio:     rfd.CopyAudio != 0,
		SurroundAudio: rfd.SurroundAudio != 0,
	}

	hash, err := tx.m.store.CreateFunc(func(dstFile *os.File) error {
		dst.File = dstFile
		onProgress := func(v float64) { tx.m.prog.Update(progKey, v) }

		if rfd.Remux != 0 {
			tx.m.prog.UpdateStatus(progKey, "download mp4: remuxing")
			return ffmpeg.RemuxToMP4(ctx, src, dst, duration, onProgress)
		}

		passlog := tx.m.pass1Passlog(vid.ID, rfd.ID)
		preset, err := tx.m.loadPass1Preset(vid.ID)
		if err != nil {
			return err
		}
		tx.m.prog.UpdateStatus(progKey, "download mp4: encoding")
		return ffmpeg.Pass2ToMP4(ctx, src, dst, passlog, preset, duration, onProgress)
	})
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}

	tx.m.prog.UpdateStatus(progKey, "download mp4: saving")
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		_, err := txw.q.RenditionUpdateEncode(ctx,
			schema.RenditionUpdateEncodeParams{
				ID:  rfd.ID,
				Key: hash,
			},
		)
		return err
	})
	if err == nil && rfd.Remux == 0 {
		tx.m.removePass1Stats(vid.ID, rfd.ID)
	}
	tx.m.prog.Close(progKey, err)
	return err
}

// ReImportVideo queues a reimport task for the given video.
// The task will delete all existing renditions and the original
// CAS blob, re-copy the source from the download path, and
// restart the full ingestion pipeline.
func (tx *TxRW) ReImportVideo(ctx Context, videoID string) error {
	return tx.addTask(ctx, taskReimport, videoID)
}

// taskReimport copies a fresh source file from Transmission into the
// store, repoints the video's OriginalKey at it, and queues a reencode.
// Existing renditions remain playable until taskReencode tears them
// down, minimizing the window during which the video is unwatchable.
func (tx *TxR) taskReimport(ctx Context, args []string) (err error) {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}
	if vid.InfoHash == nil {
		return fmt.Errorf("video %s has no info hash", vid.ID)
	}

	tm := tx.m.transmission.Load()
	if tm == nil {
		return fmt.Errorf("no transmission client available")
	}
	ts, err := tm.TorrentGetAllForHashes(ctx, []string{*vid.InfoHash})
	if err != nil {
		return err
	}
	if len(ts) != 1 {
		return fmt.Errorf("torrent %s: got %d results, wanted 1", *vid.InfoHash, len(ts))
	}
	dl, err := tx.q.DownloadGet(ctx, *vid.InfoHash)
	if err != nil {
		return err
	}
	name := torrentRelPath(dl.Title, vid.Name)
	if !filepath.IsLocal(name) {
		return fmt.Errorf("refusing reimport of non-local path %q", name)
	}
	localDir, err := tx.m.resolveDownloadDir(*ts[0].DownloadDir, name)
	if err != nil {
		return err
	}
	src, err := openIngestSource(localDir, name)
	if err != nil {
		return err
	}
	defer src.Close()

	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	tx.m.prog.Open(vid.ID, vid.Name, "Copying")
	defer func() { tx.m.prog.Close(vid.ID, err) }()

	newKey, newSum, err := tx.m.copyToStoreHashed(src)
	if err != nil {
		return err
	}
	oldKey := vid.OriginalKey

	return tx.m.WithTxRW(func(txw *TxRW) error {
		txw.onCommit(func() { tx.m.store.Remove(oldKey) })
		_, err := txw.q.VideoUpdateOriginalKey(ctx, schema.VideoUpdateOriginalKeyParams{
			ID:          vid.ID,
			OriginalKey: newKey,
			ContentHash: newSum,
		})
		if err != nil {
			return err
		}
		return txw.addTask(ctx, taskReencode, vid.ID)
	})
}

// ReencodeVideo queues a reencode task for the given video.
// The task will delete all existing renditions (CAS blobs and DB
// records), re-probe the original source, plan new renditions,
// and restart the encoding pipeline.
func (m *Model) ReencodeVideo(ctx Context, videoID string) (err error) {
	defer errorfmt.Handlef("reencode video %s: %w", videoID, &err)
	vid, err := schema.New(m.dbr).VideoGet(ctx, videoID)
	if err != nil {
		return err
	}
	if vid.OriginalKey == "" {
		return fmt.Errorf("video has no original hash")
	}
	return m.WithTxRW(func(tx *TxRW) error {
		return tx.addTask(ctx, taskReencode, videoID)
	})
}

// taskReencode deletes all existing renditions for a video
// and restarts the ingestion pipeline from the probe step.
func (tx *TxR) taskReencode(ctx Context, args []string) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}
	if vid.OriginalKey == "" {
		return fmt.Errorf("video %s has no original hash", vid.ID)
	}

	// Delete existing rendition CAS blobs and pass1 stats.
	renditions, err := tx.q.RenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, r := range renditions {
		if r.Key != "" {
			tx.m.store.Remove(r.Key)
		}
		tx.m.removePass1Stats(vid.ID, r.ID)
	}

	// Delete rendition and audio track DB records, clear MV playlist.
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		err := txw.q.AudioTrackDeleteByVideoID(ctx, vid.ID)
		if err != nil {
			return err
		}
		err = txw.q.RenditionDeleteByVideoID(ctx, vid.ID)
		if err != nil {
			return err
		}
		_, err = txw.q.VideoUpdateMVPlaylist(ctx, schema.VideoUpdateMVPlaylistParams{
			ID:         vid.ID,
			MVPlaylist: "",
		})
		return err
	})
	if err != nil {
		return err
	}

	// Set up progress tracking.
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	tx.m.prog.Open(vid.ID, vid.Name, "Re-encoding")

	return tx.planAndCreateRenditions(ctx, vid)
}

// rebuildMVPlaylist regenerates the multivariant HLS playlist from
// all currently encoded renditions for the video. If probe is
// non-nil, resolutions are computed from source dimensions;
// otherwise they are preserved from the existing MV playlist.
func rebuildMVPlaylist(ctx Context, tx *TxRW, vid schema.Video, probe *ffmpeg.ProbeResult) error {
	encoded, err := tx.q.RenditionListEncodedStreamingByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	if len(encoded) == 0 {
		return nil
	}

	// Parse existing MV playlist so we can preserve resolution
	// when probe data isn't available.
	existing := video.ParseMVPlaylist(vid.MVPlaylist)

	var mvEntries []video.MVEntry
	for _, rfs := range encoded {
		bandwidth := video.PeakBitrate(rfs.Playlist)
		if bandwidth == 0 {
			bandwidth = rfs.TargetBitrate * 1000
		}

		uri := "/-/pls/" + rfs.ID + ".m3u8"

		var resolution string
		if probe != nil && probe.Video != nil {
			w, h := video.ScaleResolution(
				probe.Video.Width, probe.Video.Height, int(rfs.MaxHeight),
			)
			resolution = video.ResolutionString(w, h)
		} else if prev, ok := existing[uri]; ok {
			resolution = prev.Resolution
		}

		r := video.Rendition{Codec: rfs.Codec}
		mvEntries = append(mvEntries, video.MVEntry{
			URI:        uri,
			Bandwidth:  bandwidth,
			Resolution: resolution,
			Codecs:     r.HLSCodecs(),
		})
	}
	mvPlaylist := video.GenerateMVPlaylist(mvEntries)

	_, err = tx.q.VideoUpdateMVPlaylist(ctx, schema.VideoUpdateMVPlaylistParams{
		ID:         vid.ID,
		MVPlaylist: mvPlaylist,
	})
	return err
}

// rendDesc returns a short human-readable description of a rendition,
// e.g. "remux 8000k", "hevc 5000k 1080p", "h264 3000k 720p".
func rendDesc(rfs schema.Rendition) string {
	if rfs.Remux != 0 {
		return fmt.Sprintf("remux %dk", rfs.TargetBitrate)
	}
	s := fmt.Sprintf("%s %dk", rfs.Codec, rfs.TargetBitrate)
	if rfs.MaxHeight > 0 {
		s += fmt.Sprintf(" %dp", rfs.MaxHeight)
	}
	return s
}

func toFFmpegCodec(codec string) string {
	switch codec {
	case "h264":
		return "libx264"
	default:
		return "libx265"
	}
}

func toVideoTag(codec string) string {
	if codec == "hevc" {
		return "hvc1"
	}
	return ""
}

func (m *Model) pass1Dir() string {
	return filepath.Join(m.persistentTmp, "pass1")
}

func (m *Model) pass1StatsDir(vidID string) string {
	return filepath.Join(m.pass1Dir(), vidID)
}

func (m *Model) pass1Passlog(vidID, rfsID string) string {
	return filepath.Join(m.pass1StatsDir(vidID), rfsID)
}

func (m *Model) loadPass1Preset(vidID string) (string, error) {
	b, err := os.ReadFile(filepath.Join(m.pass1StatsDir(vidID), "preset"))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (m *Model) removePass1Stats(vidID, rfsID string) {
	passlog := m.pass1Passlog(vidID, rfsID)
	os.Remove(passlog)
	os.Remove(passlog + ".mbtree")
	os.Remove(passlog + ".cutree")
}
