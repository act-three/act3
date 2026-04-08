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

	"ily.dev/act3/database/schema"
	"ily.dev/act3/priority"
	"ily.dev/act3/video"
	"ily.dev/act3/video/ffmpeg"
)

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

	// Transmission may still be copying the file to its final
	// location. If the file doesn't exist yet, re-queue and
	// check again shortly.
	diskPath := args[1]
	if _, err := os.Stat(diskPath); errors.Is(err, os.ErrNotExist) {
		slog.InfoContext(ctx, "ingest: file not yet available, retrying",
			"vid", vid.ID, "path", diskPath)
		return tx.m.WithTxRW(func(txw *TxRW) error {
			return txw.addTaskAfter(ctx, 5*time.Second, taskIngest, args...)
		})
	}

	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	tx.m.prog.Open(vid.ID, vid.Name, "Copying")
	hash, err := tx.m.store.CopyFile(diskPath)
	if err != nil {
		tx.m.prog.Close(vid.ID, err)
		return err
	}

	err = tx.m.WithTxRW(func(txw *TxRW) error {
		vid, err = txw.q.VideoUpdateOriginalKey(ctx, schema.VideoUpdateOriginalKeyParams{
			ID:          vid.ID,
			OriginalKey: hash,
		})
		return err
	})
	if err != nil {
		tx.m.prog.Close(vid.ID, err)
		return err
	}

	return tx.planAndCreateRenditions(ctx, vid)
}

// planAndCreateRenditions probes the source in CAS, plans a
// rendition ladder, creates the rendition DB records, and queues
// pass1. The caller must have already set vid.OriginalKey and
// opened progress tracking for vid.ID.
func (tx *TxR) planAndCreateRenditions(ctx Context, vid schema.Video) error {
	existing, err := tx.q.RenditionForStreamingListDirectByVideoID(ctx, vid.ID)
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

	var rfsParams []schema.RenditionForStreamingCreateParams
	for _, r := range planned {
		rfsParams = append(rfsParams, schema.RenditionForStreamingCreateParams{
			VideoID:       vid.ID,
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
		for _, at := range atParams {
			_, err = txw.q.AudioTrackCreate(ctx, at)
			if err != nil {
				return err
			}
		}
		for _, rfs := range rfsParams {
			_, err = txw.q.RenditionForStreamingCreate(ctx, rfs)
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

	rfsList, err := tx.q.RenditionForStreamingListDirectByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	if len(rfsList) == 0 {
		return fmt.Errorf("no renditions planned for video %s", vid.ID)
	}

	// Classify renditions: collect indices of those needing reencode.
	var encodeIdxs []int
	for i, rfs := range rfsList {
		if rfs.Remux == 0 {
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

		// Build EncodeParams for the reencode renditions.
		dsts := make([]ffmpeg.EncodeParams, len(rfsList))
		for _, i := range encodeIdxs {
			rfs := rfsList[i]
			dsts[i] = ffmpeg.EncodeParams{
				Codec:         toFFmpegCodec(rfs.Codec),
				Bitrate:       rfs.TargetBitrate,
				MaxHeight:     int(rfs.MaxHeight),
				MaxFPS:        int(rfs.MaxFPS),
				Tag:           toVideoTag(rfs.Codec),
				CopyAudio:     rfs.CopyAudio != 0,
				SurroundAudio: rfs.SurroundAudio != 0,
			}
		}

		// Create stats directory and build passlog paths.
		statsDir := tx.m.pass1StatsDir(vid.ID)
		if err := os.MkdirAll(statsDir, 0o777); err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}
		passlogs := make(map[int]string, len(encodeIdxs))
		for _, i := range encodeIdxs {
			passlogs[i] = filepath.Join(statsDir, rfsList[i].ID)
		}

		tx.m.prog.UpdateStatus(vid.ID, fmt.Sprintf("Pass 1 (%d renditions)", len(encodeIdxs)))
		preset, err := ffmpeg.Pass1Combined(ctx, src, dsts, encodeIdxs, passlogs, probe.Duration,
			func(v float64) { tx.m.prog.Update(vid.ID, v) },
		)
		if err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}

		// Save preset so pass 2 uses the same one.
		err = os.WriteFile(filepath.Join(statsDir, "preset"), []byte(preset), 0o666)
		if err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}
	}

	// Queue one taskIngestEncodeRend per rendition.
	// Rendition priorities are set by video.PlanRenditions
	// using constants from the priority package.
	tx.m.prog.UpdateStatus(vid.ID, "Queuing renditions")
	tx.m.prog.Close(vid.ID, nil)

	return tx.m.WithTxRW(func(txw *TxRW) error {
		for i := range rfsList {
			txw.addTaskWithPriority(ctx, rfsList[i].Priority, taskIngestEncodeRend, vid.ID)
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

	rfs, err := tx.q.RenditionForStreamingNextUnencoded(ctx, vid.ID)
	if err == sql.ErrNoRows {
		// All renditions already encoded (race or duplicate task).
		return nil
	}
	if err != nil {
		return err
	}

	// Progress tracking: use rfs ID as key, link to video.
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

	// Fixup media playlist: replace temp media filename with storage hash.
	playlist = video.FixupMediaPlaylist(playlist, ffmpeg.MediaName(0), "/-/vid/"+hash+".mp4")

	tx.m.prog.UpdateStatus(progKey, desc+": saving")
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		_, err := txw.q.RenditionForStreamingUpdateEncode(ctx,
			schema.RenditionForStreamingUpdateEncodeParams{
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
	srcPath, err := tx.transmissionDiskPath(ctx, &ts[0], vid.Name)
	if err != nil {
		return err
	}

	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	tx.m.prog.Open(vid.ID, vid.Name, "Copying")
	defer func() { tx.m.prog.Close(vid.ID, err) }()

	newKey, err := tx.m.store.CopyFile(srcPath)
	if err != nil {
		return err
	}
	oldKey := vid.OriginalKey

	return tx.m.WithTxRW(func(txw *TxRW) error {
		txw.onCommit(func() { tx.m.store.Remove(oldKey) })
		_, err := txw.q.VideoUpdateOriginalKey(ctx, schema.VideoUpdateOriginalKeyParams{
			ID:          vid.ID,
			OriginalKey: newKey,
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
	rfsList, err := tx.q.RenditionForStreamingListDirectByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	for _, rfs := range rfsList {
		if rfs.Key != "" {
			tx.m.store.Remove(rfs.Key)
		}
		tx.m.removePass1Stats(vid.ID, rfs.ID)
	}

	// Delete rendition and audio track DB records, clear MV playlist.
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		err := txw.q.AudioTrackDeleteByVideoID(ctx, vid.ID)
		if err != nil {
			return err
		}
		err = txw.q.RenditionForStreamingDeleteByVideoID(ctx, vid.ID)
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
	encoded, err := tx.q.RenditionForStreamingListEncodedByVideoID(ctx, vid.ID)
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
func rendDesc(rfs schema.RenditionForStreaming) string {
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
