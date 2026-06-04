package model

import (
	"database/sql"
	"encoding/json/v2"
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
func (tx *TxR) planAndCreateRenditions(ctx Context, vid schema.Video) (err error) {
	defer func() { tx.m.prog.Close(vid.ID, err) }()
	existing, err := tx.q.RenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	tx.m.prog.UpdateStatus(vid.ID, "Probing")
	f, err := tx.m.store.Open(vid.OriginalKey)
	if err != nil {
		return err
	}
	defer f.Close()
	probe, err := ffmpeg.Probe(ctx, f)
	if err != nil {
		return err
	}

	// When the source might be remuxable, the planner falls back to
	// re-encoding if GOPs are long enough that aligned segments would
	// overshoot Apple's recommended ceiling. Probe.Video.Keyframes is
	// already populated; we only need to convert the largest gap to a
	// duration. Use the coded frame rate (the rate the encoder
	// actually sees under -fps_mode passthrough) so soft-telecine
	// sources don't get measured at the inflated display rate.
	var maxKeyframeGap time.Duration
	if probe.Video != nil && (probe.Video.CodecName == "h264" || probe.Video.CodecName == "hevc") {
		gapFrames := ffmpeg.MaxKeyframeGap(probe.Video.Keyframes)
		fps := probe.Video.CodedFrameRate()
		if fps.Positive() {
			maxKeyframeGap = time.Duration(gapFrames) * time.Second *
				time.Duration(fps.Den) / time.Duration(fps.Num)
		}
	}

	planned, err := video.PlanVideoRenditions(probe, maxKeyframeGap)
	if err != nil {
		return err
	}
	if len(planned) == 0 {
		return fmt.Errorf("no video stream found in source")
	}
	audioPlanned := video.PlanAudioRenditions(probe)

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
			SampleRate:    int64(as.SampleRate),
			Codec:         as.CodecName,
			Profile:       as.Profile,
		})
	}

	var stParams []schema.SubtitleTrackCreateParams
	for _, ss := range probe.Subtitles {
		stParams = append(stParams, schema.SubtitleTrackCreateParams{
			VideoID:       vid.ID,
			StreamIndex:   int64(ss.Index),
			Language:      ss.Language,
			Title:         ss.Title,
			OriginalCodec: ss.CodecName,
			Forced:        boolToInt64(ss.Forced),
		})
	}

	keyframesJSON, err := json.Marshal(probe.Video.Keyframes)
	if err != nil {
		return err
	}

	tx.m.prog.UpdateStatus(vid.ID, "Queued")
	return tx.m.WithTxRW(func(txw *TxRW) error {
		err = txw.q.VideoUpdateProbe(ctx, schema.VideoUpdateProbeParams{
			ID:                 vid.ID,
			Duration:           probe.Duration.Milliseconds(),
			OriginalType:       probe.ContentType(),
			Format:             probe.FormatName,
			Width:              int64(probe.Video.Width),
			Height:             int64(probe.Video.Height),
			FrameRateNum:       int64(probe.Video.FrameRate.Num),
			FrameRateDen:       int64(probe.Video.FrameRate.Den),
			VideoPacketCount:   probe.Video.PacketCount,
			VideoDurationTicks: probe.Video.DurationTicks,
			VideoTimebaseNum:   int64(probe.Video.TimebaseNum),
			VideoTimebaseDen:   int64(probe.Video.TimebaseDen),
			VideoKeyframes:     string(keyframesJSON),
			DolbyVisionProfile: int64(probe.Video.DolbyVisionProfile),
			ColorTransfer:      probe.Video.ColorTransfer,
		})
		if err != nil {
			return err
		}
		streamIndexToTrackID := make(map[int]string, len(atParams))
		for _, at := range atParams {
			created, err := txw.q.AudioTrackCreate(ctx, at)
			if err != nil {
				return err
			}
			streamIndexToTrackID[int(created.StreamIndex)] = created.ID
		}
		for _, st := range stParams {
			_, err = txw.q.SubtitleTrackCreate(ctx, st)
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
		for _, ar := range audioPlanned {
			trackID, ok := streamIndexToTrackID[ar.SourceStreamIndex]
			if !ok {
				return fmt.Errorf("audio rendition: no AudioTrack for source stream %d", ar.SourceStreamIndex)
			}
			_, err = txw.q.AudioRenditionCreate(ctx, schema.AudioRenditionCreateParams{
				VideoID:      vid.ID,
				AudioTrackID: trackID,
				Channels:     int64(ar.Channels),
				Bitrate:      ar.Bitrate,
				Codec:        ar.Codec,
				Priority:     int64(ar.Priority),
				SortKey:      int64(ar.SortKey),
			})
			if err != nil {
				return err
			}
		}
		txw.addTaskWithPriority(ctx, priority.Pass1, taskIngestPass1, vid.ID)
		for range audioPlanned {
			txw.addTaskWithPriority(ctx, priority.EncodeAudio, taskIngestEncodeAudio, vid.ID)
		}
		if len(stParams) > 0 {
			txw.addTask(ctx, taskIngestExtractSubs, vid.ID)
		}
		return nil
	})
}

// taskIngestPass1 runs combined first-pass analysis for all reencode
// renditions of a video, saves the stats blobs to the DB, then queues
// per-rendition encode tasks.
func (tx *TxR) taskIngestPass1(ctx Context, args []string) (err error) {
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
	defer func() { tx.m.prog.Close(vid.ID, err) }()

	renditions, err := tx.q.RenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	if len(renditions) == 0 {
		return fmt.Errorf("no renditions planned for video %s", vid.ID)
	}

	// Count renditions needing reencode to decide whether to run pass 1.
	var numReencode int
	var hasRemux bool
	for _, r := range renditions {
		if r.Remux == 0 {
			numReencode++
		} else {
			hasRemux = true
		}
	}

	if numReencode > 0 {
		src, err := tx.m.store.Open(vid.OriginalKey)
		if err != nil {
			return err
		}
		defer src.Close()

		boundaries, fps, err := sourceSegmentBoundaries(vid, hasRemux)
		if err != nil {
			return fmt.Errorf("source segment boundaries: %w", err)
		}

		dolbyVision := ffmpeg.DolbyVisionNeedsConversion(int(vid.DolbyVisionProfile))
		dsts := make([]ffmpeg.EncodeParams, len(renditions))
		for i, r := range renditions {
			dsts[i] = ffmpeg.EncodeParams{
				Remux:       r.Remux != 0,
				Codec:       toFFmpegCodec(r.Codec),
				Bitrate:     r.TargetBitrate,
				MaxHeight:   int(r.MaxHeight),
				MaxFPS:      int(r.MaxFPS),
				Tag:         toVideoTag(r.Codec),
				StatsID:     r.ID,
				DolbyVision: dolbyVision && r.Remux == 0,
			}
			// Forced keyframes (and the keyint=99999:scenecut=0
			// extras that go with them) only matter for streaming
			// renditions, which Pass2Single also passes through
			// keyframeArgs. The download rendition's Pass2ToMP4 runs
			// without those extras, so giving it the same extras in
			// pass 1 would write stats x265 then rejects in pass 2
			// with "different keyint setting than first pass".
			if r.Purpose != "download" {
				dsts[i].SegmentBoundaries = boundaries
				dsts[i].SegmentBoundaryRate = fps
			}
		}

		statsDir := tx.m.pass1Dir(vid.ID)
		if err := os.MkdirAll(statsDir, 0o755); err != nil {
			return err
		}

		tx.m.prog.UpdateStatus(vid.ID, fmt.Sprintf("Pass 1 (%d renditions)", numReencode))
		err = ffmpeg.Pass1Combined(ctx, src, vid.Format, dsts, statsDir, time.Duration(vid.Duration)*time.Millisecond,
			func(v float64) { tx.m.prog.Update(vid.ID, v) },
		)
		if err != nil {
			return err
		}
	}

	// Queue one encode task per rendition.
	tx.m.prog.UpdateStatus(vid.ID, "Queuing renditions")
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

// taskIngestEncodeAudio picks the highest-priority unencoded audio
// rendition for a video, encodes it, and rebuilds the MV playlist.
func (tx *TxR) taskIngestEncodeAudio(ctx Context, args []string) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}

	ar, err := tx.q.AudioRenditionNextUnencoded(ctx, vid.ID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	at, err := tx.q.AudioTrackGet(ctx, ar.AudioTrackID)
	if err != nil {
		return err
	}

	progKey := "aud-" + ar.ID
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	tx.m.prog.AddEdge(vid.ID, progKey)
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	desc := audioRendDesc(ar)
	tx.m.prog.Open(progKey, vid.Name, desc+": starting")

	src, err := tx.m.store.Open(vid.OriginalKey)
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}
	defer src.Close()

	streamCopy := at.Codec == "aac" &&
		at.Profile == "LC" &&
		int(at.Channels) == int(ar.Channels) &&
		ar.Channels <= 2 &&
		(at.SampleRate == 44100 || at.SampleRate == 48000)

	dst := ffmpeg.AudioEncodeParams{
		SourceStreamIndex: int(at.StreamIndex),
		Channels:          int(ar.Channels),
		SourceLayout:      at.ChannelLayout,
		Bitrate:           ar.Bitrate,
		StreamCopy:        streamCopy,
	}
	duration := time.Duration(vid.Duration) * time.Millisecond

	var playlist string
	hash, err := tx.m.store.CreateFunc(func(dstFile *os.File) error {
		dst.File = dstFile
		tx.m.prog.UpdateStatus(progKey, desc+": encoding")
		var err error
		playlist, err = ffmpeg.EncodeAudio(ctx, src, vid.Format, dst, duration,
			func(v float64) { tx.m.prog.Update(progKey, v) })
		return err
	})
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}

	playlist = video.FixupMediaPlaylist(playlist, ffmpeg.MediaName(0), "/-/aud/"+ar.ID+".mp4")

	tx.m.prog.UpdateStatus(progKey, desc+": saving")
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		_, err := txw.q.AudioRenditionUpdateEncode(ctx,
			schema.AudioRenditionUpdateEncodeParams{
				ID:       ar.ID,
				Key:      hash,
				Playlist: playlist,
			})
		if err != nil {
			return err
		}
		return txw.recomputePlayable(ctx, vid)
	})
	tx.m.prog.Close(progKey, err)
	return err
}

func audioRendDesc(ar schema.AudioRendition) string {
	return fmt.Sprintf("audio %dch %dk", ar.Channels, ar.Bitrate)
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

	// Determine whether any rendition is a stream-copy remux: that
	// decides whether segment boundaries must follow source keyframes
	// or can be on a synthetic grid.
	all, err := tx.q.RenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	var hasRemux bool
	for _, r := range all {
		if r.Remux != 0 {
			hasRemux = true
			break
		}
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

	duration := time.Duration(vid.Duration) * time.Millisecond

	boundaries, fps, err := sourceSegmentBoundaries(vid, hasRemux)
	if err != nil {
		return fmt.Errorf("source segment boundaries: %w", err)
	}

	dst := ffmpeg.EncodeParams{
		Remux:               rfs.Remux != 0,
		Codec:               toFFmpegCodec(rfs.Codec),
		Bitrate:             rfs.TargetBitrate,
		MaxHeight:           int(rfs.MaxHeight),
		MaxFPS:              int(rfs.MaxFPS),
		Tag:                 toVideoTag(rfs.Codec),
		StatsID:             rfs.ID,
		SegmentBoundaries:   boundaries,
		SegmentBoundaryRate: fps,
		DolbyVision:         rfs.Remux == 0 && ffmpeg.DolbyVisionNeedsConversion(int(vid.DolbyVisionProfile)),
	}

	var playlist string
	hash, err := tx.m.store.CreateFunc(func(dstFile *os.File) error {
		dst.File = dstFile
		onProgress := func(v float64) { tx.m.prog.Update(progKey, v) }

		if rfs.Remux != 0 {
			tx.m.prog.UpdateStatus(progKey, desc+": remuxing")
			var err error
			playlist, err = ffmpeg.RemuxSingle(ctx, src, vid.Format, dst, duration, onProgress)
			return err
		}

		tx.m.prog.UpdateStatus(progKey, desc+": encoding")
		var err error
		playlist, err = ffmpeg.Pass2Single(ctx, src, vid.Format, dst,
			tx.m.pass1Dir(vid.ID), duration, onProgress)
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
	var allDone bool
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
		allDone, err = txw.allRenditionsEncoded(ctx, vid.ID)
		if err != nil {
			return err
		}

		// Rebuild MV playlist from all completed renditions.
		return txw.recomputePlayable(ctx, vid)
	})
	if err == nil && allDone {
		os.RemoveAll(tx.m.pass1Dir(vid.ID))
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
		Remux:       rfd.Remux != 0,
		Codec:       toFFmpegCodec(rfd.Codec),
		Bitrate:     rfd.TargetBitrate,
		MaxHeight:   int(rfd.MaxHeight),
		MaxFPS:      int(rfd.MaxFPS),
		Tag:         toVideoTag(rfd.Codec),
		StatsID:     rfd.ID,
		DolbyVision: rfd.Remux == 0 && ffmpeg.DolbyVisionNeedsConversion(int(vid.DolbyVisionProfile)),
	}

	hash, err := tx.m.store.CreateFunc(func(dstFile *os.File) error {
		dst.File = dstFile
		onProgress := func(v float64) { tx.m.prog.Update(progKey, v) }

		if rfd.Remux != 0 {
			tx.m.prog.UpdateStatus(progKey, "download mp4: remuxing")
			return ffmpeg.RemuxToMP4(ctx, src, vid.Format, dst, duration, onProgress)
		}

		tx.m.prog.UpdateStatus(progKey, "download mp4: encoding")
		return ffmpeg.Pass2ToMP4(ctx, src, vid.Format, dst,
			tx.m.pass1Dir(vid.ID), duration, onProgress)
	})
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}

	tx.m.prog.UpdateStatus(progKey, "download mp4: saving")
	var allDone bool
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		_, err := txw.q.RenditionUpdateEncode(ctx,
			schema.RenditionUpdateEncodeParams{
				ID:  rfd.ID,
				Key: hash,
			},
		)
		if err != nil {
			return err
		}
		allDone, err = txw.allRenditionsEncoded(ctx, vid.ID)
		return err
	})
	if err == nil && allDone {
		os.RemoveAll(tx.m.pass1Dir(vid.ID))
	}
	tx.m.prog.Close(progKey, err)
	return err
}

// ReImportVideo queues a reimport task for the given video.
// The task will delete all existing renditions and the original
// CAS blob, re-copy the source from the download path, and
// restart the full ingestion pipeline.
func (tx *TxRW) ReImportVideo(ctx Context, videoID string) error {
	if err := tx.guardActiveVideo(ctx, videoID); err != nil {
		return err
	}
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
		if err := tx.guardActiveVideo(ctx, videoID); err != nil {
			return err
		}
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

	// Delete existing rendition CAS blobs and any stale pass1 data
	// left behind by a prior cycle.
	renditions, err := tx.q.RenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, r := range renditions {
		if r.Key != "" {
			tx.m.store.Remove(r.Key)
		}
	}
	audioRends, err := tx.q.AudioRenditionListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, ar := range audioRends {
		if ar.Key != "" {
			tx.m.store.Remove(ar.Key)
		}
	}
	subTracks, err := tx.q.SubtitleTrackListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, st := range subTracks {
		if st.OriginalKey != "" {
			tx.m.store.Remove(st.OriginalKey)
		}
		if st.WebVTTKey != "" {
			tx.m.store.Remove(st.WebVTTKey)
		}
	}
	os.RemoveAll(tx.m.pass1Dir(vid.ID))

	// Delete rendition, audio track, and subtitle track DB records;
	// clear the playable flag.
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		err := txw.q.AudioRenditionDeleteByVideoIDList(ctx, []string{vid.ID})
		if err != nil {
			return err
		}
		err = txw.q.AudioTrackDeleteByVideoID(ctx, vid.ID)
		if err != nil {
			return err
		}
		err = txw.q.SubtitleTrackDeleteByVideoID(ctx, vid.ID)
		if err != nil {
			return err
		}
		err = txw.q.RenditionDeleteByVideoID(ctx, vid.ID)
		if err != nil {
			return err
		}
		_, err = txw.q.VideoUpdatePlayable(ctx, schema.VideoUpdatePlayableParams{
			ID:       vid.ID,
			Playable: 0,
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

	// Open a fresh read transaction so planAndCreateRenditions's
	// existing-rendition guard observes the post-delete state. Reusing
	// the outer tx would still see the pre-delete snapshot pinned by
	// our earlier reads, and the guard would short-circuit.
	return tx.m.WithTxR(func(tx *TxR) error {
		return tx.planAndCreateRenditions(ctx, vid)
	})
}

// recomputePlayable updates Video.Playable based on whether the
// renditions needed for an MV playlist are all present. Only flips
// the flag to 1; callers that need to clear it (e.g. re-encode) do
// so explicitly. The MV playlist itself is built on demand by
// (TxR).MVPlaylist.
func (tx *TxRW) recomputePlayable(ctx Context, vid schema.Video) error {
	encoded, err := tx.q.RenditionListEncodedStreamingByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	encodedAudio, err := tx.q.AudioRenditionListEncodedForMV(ctx, vid.ID)
	if err != nil {
		return err
	}
	tracks, err := tx.q.AudioTrackListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}

	if !isPlayableMV(encoded, encodedAudio, tracks) {
		return nil
	}

	_, err = tx.q.VideoUpdatePlayable(ctx, schema.VideoUpdatePlayableParams{
		ID:       vid.ID,
		Playable: 1,
	})
	if err != nil {
		return err
	}
	return tx.ensureActiveVideoForVideoID(ctx, vid.ID)
}

// isPlayableMV reports whether a video has all the renditions
// needed to assemble a multivariant playlist: at least one encoded
// streaming video rendition, and an encoded audio rendition for
// every source audio track. A partially-encoded MV would either
// omit audio entries (silent playback) or expose only a subset of
// the publisher's intended tracks. Inputs are the unfiltered sets
// — callers must check this before applying any per-rendition
// filter, otherwise the readiness signal is lost.
func isPlayableMV(
	encoded []schema.Rendition,
	encodedAudio []schema.AudioRendition,
	tracks []schema.AudioTrack,
) bool {
	if len(encoded) == 0 {
		return false
	}
	if len(tracks) == 0 {
		return true
	}
	ready := make(map[string]bool, len(encodedAudio))
	for _, ar := range encodedAudio {
		ready[ar.AudioTrackID] = true
	}
	for _, t := range tracks {
		if !ready[t.ID] {
			return false
		}
	}
	return true
}

// buildMVPlaylist generates a multivariant HLS playlist from the
// given inputs. The set of videoRends controls whether this is a
// full MV (all renditions) or a single-variant MV (one rendition).
// Callers must verify playability via isPlayableMV before filtering
// the inputs and calling this function; an empty videoRends slice
// (e.g. from a filter that didn't match) yields "".
func buildMVPlaylist(
	vid schema.Video,
	videoRends []schema.Rendition,
	encodedAudio []schema.AudioRendition,
	tracks []schema.AudioTrack,
	subTracks []schema.SubtitleTrack,
) string {
	if len(videoRends) == 0 {
		return ""
	}

	hdr := ffmpeg.HDRFormat(int(vid.DolbyVisionProfile), vid.ColorTransfer)
	srcFPS := codedFrameRate(vid)
	var mvEntries []video.MVEntry
	for _, rfs := range videoRends {
		bandwidth := video.PeakBitrate(rfs.Playlist)
		if bandwidth == 0 {
			bandwidth = rfs.TargetBitrate * 1000
		}
		w, h := video.ScaleResolution(
			int(vid.Width), int(vid.Height), int(rfs.MaxHeight),
		)
		// The variant's peak frame rate: the coded source rate (what
		// -fps_mode passthrough emits), capped by the rendition's fps
		// filter when present.
		var fps float64
		if srcFPS.Positive() {
			fps = float64(srcFPS.Num) / float64(srcFPS.Den)
			if rfs.MaxFPS > 0 && float64(rfs.MaxFPS) < fps {
				fps = float64(rfs.MaxFPS)
			}
		}
		r := video.Rendition{Codec: rfs.Codec, HDR: hdr}
		mvEntries = append(mvEntries, video.MVEntry{
			URI:        "/-/pls/" + rfs.ID + ".m3u8",
			Bandwidth:  bandwidth,
			Resolution: video.ResolutionString(w, h),
			Codecs:     r.HLSCodecs(),
			VideoRange: r.HLSVideoRange(),
			FrameRate:  fps,
		})
	}

	tracksByID := make(map[string]schema.AudioTrack, len(tracks))
	for _, at := range tracks {
		tracksByID[at.ID] = at
	}
	var mvAudios []video.MVAudio
	for i, ar := range encodedAudio {
		at := tracksByID[ar.AudioTrackID]
		// NAME carries the AudioRendition ID, not a human label. The
		// browser exposes it as audioTracks[i].label; the player JS
		// matches by that opaque ID. Display strings live in the
		// player menu's visible text, so labels need not be globally
		// unique, escaped, or generated identically across two paths.
		mvAudios = append(mvAudios, video.MVAudio{
			URI:       "/-/audpls/" + ar.ID + ".m3u8",
			Name:      ar.ID,
			Language:  at.Language,
			Channels:  int(ar.Channels),
			Default:   i == 0,
			IsDownmix: ar.Channels < at.Channels,
		})
	}

	var mvSubs []video.MVSubtitle
	// Default-track rule: prefer the first non-forced English track,
	// else the first non-forced track, else nothing. A single linear
	// scan picks the index; only that one entry sets Default=true.
	defaultIdx := -1
	for i, st := range subTracks {
		if st.WebVTTKey == "" {
			continue
		}
		if st.Forced != 0 {
			continue
		}
		if st.Language == "eng" {
			defaultIdx = i
			break
		}
		if defaultIdx == -1 {
			defaultIdx = i
		}
	}
	for i, st := range subTracks {
		if st.WebVTTKey == "" {
			continue
		}
		// NAME carries the SubtitleTrack ID, not a human label —
		// same shape as audio (ACT-168). Display text lives in the
		// player menu's visible text.
		mvSubs = append(mvSubs, video.MVSubtitle{
			URI:      "/-/subpls/" + st.ID + ".m3u8",
			Name:     st.ID,
			Language: st.Language,
			Default:  i == defaultIdx,
			Forced:   st.Forced != 0,
		})
	}

	return video.GenerateMVPlaylist(mvEntries, mvAudios, mvSubs)
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

// pass1Dir is the directory ffmpeg owns for storing pass-1 stats for
// every rendition of a video. Callers treat its contents as opaque:
// creation and removal are this package's responsibility, but the
// files inside belong to video/ffmpeg.
func (m *Model) pass1Dir(vidID string) string {
	return filepath.Join(m.pass1Root, vidID)
}

// sourceSegmentBoundaries returns the HLS segment cut points every
// rendition of this source should share, as source-frame indices in
// the encoder's input order (i.e. coded-rate frames, what
// -force_key_frames "expr:eq(n,N)" indexes).
//
// When hasRemux is true the cuts must land on source keyframes, so
// we pick a subset of the persisted keyframe list spaced at least
// MinSegmentDuration apart. Re-encodes are then forced to keyframe
// at exactly those source frames, keeping every variant aligned to
// within one frame.
//
// When hasRemux is false every variant is under our encoder's
// control, so we skip the keyframe constraint and emit a uniform
// MinSegmentDuration grid the encoders can hit exactly. Sources
// with long GOPs would otherwise inherit those long GOPs as segment
// lengths needlessly.
//
// Coded rate (rather than the display rate carried in
// FrameRateNum/Den) is used because -fps_mode passthrough hands the
// encoder one frame per source coded picture; on soft-telecine and
// VFR sources the display rate is higher and would produce too few
// frames per segment.
func sourceSegmentBoundaries(vid schema.Video, hasRemux bool) ([]int64, ffmpeg.FrameRate, error) {
	fps := codedFrameRate(vid)
	if !hasRemux {
		duration := time.Duration(vid.Duration) * time.Millisecond
		minFrames := ffmpeg.MinFramesPerSegment(fps, ffmpeg.MinSegmentDuration)
		return ffmpeg.UniformSegmentBoundaries(fps, duration, minFrames), fps, nil
	}
	var keyframes []int64
	if err := json.Unmarshal([]byte(vid.VideoKeyframes), &keyframes); err != nil {
		return nil, fps, fmt.Errorf("unmarshal video keyframes: %w", err)
	}
	return ffmpeg.SegmentBoundaries(keyframes, fps, ffmpeg.MinSegmentDuration), fps, nil
}

// codedFrameRate returns the source's coded picture rate computed
// from the persisted probe measurements. Falls back to the display
// rate when the new fields aren't populated (e.g. for legacy Video
// records ingested before the coded-rate columns existed).
func codedFrameRate(vid schema.Video) ffmpeg.FrameRate {
	v := &ffmpeg.VideoStream{
		PacketCount:   vid.VideoPacketCount,
		DurationTicks: vid.VideoDurationTicks,
		TimebaseNum:   int(vid.VideoTimebaseNum),
		TimebaseDen:   int(vid.VideoTimebaseDen),
	}
	if fps := v.CodedFrameRate(); fps.Positive() {
		return fps
	}
	return ffmpeg.FrameRate{Num: int(vid.FrameRateNum), Den: int(vid.FrameRateDen)}
}

// allRenditionsEncoded reports whether every rendition for videoID
// has a storage key, meaning no further encode task will need the
// pass-1 stats directory.
func (tx *TxRW) allRenditionsEncoded(ctx Context, videoID string) (bool, error) {
	rends, err := tx.q.RenditionListByVideoID(ctx, videoID)
	if err != nil {
		return false, err
	}
	for _, r := range rends {
		if r.Key == "" {
			return false, nil
		}
	}
	return true, nil
}
