package model

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	"ily.dev/act3/database/schema"
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
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	tx.m.prog.Open(vid.ID, vid.ReleasePath, "Copying")
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}
	hash, err := tx.m.store.Copy(args[1])
	if err != nil {
		tx.m.prog.Close(vid.ID, err)
		return err
	}

	// Probe source to get codec info, resolution, bitrate, etc.
	tx.m.prog.UpdateStatus(vid.ID, "Probing")
	f, err := tx.m.store.Open(hash)
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

	// Plan renditions based on the source properties.
	planned, err := video.PlanRenditions(probe)
	if err != nil {
		return err
	}
	if len(planned) == 0 {
		return fmt.Errorf("no video stream found in source")
	}

	// Build rendition params for DB storage.
	// Index = priority (0 = best rendition, from PlanRenditions order).
	var rfsParams []schema.RenditionForStreamingCreateParams
	for i, r := range planned {
		rfsParams = append(rfsParams, schema.RenditionForStreamingCreateParams{
			VideoID:       vid.ID,
			Remux:         boolToInt64(r.Remux),
			Codec:         r.Codec,
			TargetBitrate: r.TargetBitrate,
			MaxHeight:     int64(r.MaxHeight),
			MaxFPS:        int64(r.MaxFPS),
			CopyAudio:     boolToInt64(r.CopyAudio),
			SurroundAudio: boolToInt64(r.SurroundAudio),
			Priority:      int64(i),
		})
	}

	return tx.m.WithTxRW(func(tx *TxRW) error {
		vid, err = tx.q.VideoUpdateOriginalHash(ctx, schema.VideoUpdateOriginalHashParams{
			ID:           vid.ID,
			OriginalHash: hash,
		})
		if err != nil {
			return err
		}

		for _, rfs := range rfsParams {
			_, err = tx.q.RenditionForStreamingCreate(ctx, rfs)
			if err != nil {
				return err
			}
		}

		tx.m.prog.Open(vid.ID, vid.ReleasePath, "Queued")
		tx.addTask(ctx, taskIngestPass1, vid.ID)
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
	tx.m.prog.Open(vid.ID, vid.ReleasePath, "Starting")
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}

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
		src, err := tx.m.store.Open(vid.OriginalHash)
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

		tx.m.prog.UpdateStatus(vid.ID, fmt.Sprintf("Pass 1 (%d renditions)", len(encodeIdxs)))
		results, err := ffmpeg.Pass1Combined(ctx, src, dsts, encodeIdxs, probe.Duration,
			func(v float64) { tx.m.prog.Update(vid.ID, v) },
		)
		if err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}

		// Save stats to DB.
		err = tx.m.WithTxRW(func(txw *TxRW) error {
			for _, r := range results {
				_, err := txw.q.RenditionForStreamingUpdatePass1Stats(ctx,
					schema.RenditionForStreamingUpdatePass1StatsParams{
						ID:         rfsList[r.Index].ID,
						Pass1Stats: r.Stats,
						Preset:     r.Preset,
					},
				)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			tx.m.prog.Close(vid.ID, err)
			return err
		}
	}

	// Queue one taskIngestEncodeRend per rendition, with ascending
	// priority so that best renditions from all videos run first.
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
	progKey := "rfs/" + rfs.ID
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	desc := rendDesc(rfs)
	tx.m.prog.Open(progKey, vid.ReleasePath, desc+": starting")
	tx.m.prog.AddEdge(vid.ID, progKey)
	for _, ev := range evs {
		tx.m.prog.AddEdge(ev.EpisodeID, vid.ID)
	}

	src, err := tx.m.store.Open(vid.OriginalHash)
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
	hashes, err := tx.m.store.CreateNFunc(1, func(dstFiles []*os.File) error {
		dst.File = dstFiles[0]
		onProgress := func(v float64) { tx.m.prog.Update(progKey, v) }

		if rfs.Remux != 0 {
			tx.m.prog.UpdateStatus(progKey, desc+": remuxing")
			var err error
			playlist, err = ffmpeg.RemuxSingle(ctx, src, dst, probe.Duration, onProgress)
			return err
		}

		tx.m.prog.UpdateStatus(progKey, desc+": encoding")
		var err error
		playlist, err = ffmpeg.Pass2Single(ctx, src, dst, rfs.Pass1Stats, rfs.Preset, probe.Duration, onProgress)
		return err
	})
	if err != nil {
		tx.m.prog.Close(progKey, err)
		return err
	}

	// Fixup media playlist: replace temp media filename with storage hash.
	playlist = video.FixupMediaPlaylist(
		playlist, ffmpeg.MediaName(0), "/vids/"+hashes[0]+".mp4",
	)

	tx.m.prog.UpdateStatus(progKey, desc+": saving")
	err = tx.m.WithTxRW(func(txw *TxRW) error {
		_, err := txw.q.RenditionForStreamingUpdateEncode(ctx,
			schema.RenditionForStreamingUpdateEncodeParams{
				ID:       rfs.ID,
				Hash:     hashes[0],
				Playlist: playlist,
			},
		)
		if err != nil {
			return err
		}

		// Rebuild MV playlist from all completed renditions.
		return rebuildMVPlaylist(ctx, txw, vid, probe)
	})
	tx.m.prog.Close(progKey, err)
	return err
}

// rebuildMVPlaylist regenerates the multivariant HLS playlist from
// all currently encoded renditions for the video.
func rebuildMVPlaylist(ctx Context, tx *TxRW, vid schema.Video, probe *ffmpeg.ProbeResult) error {
	encoded, err := tx.q.RenditionForStreamingListEncodedByVideoID(ctx, vid.ID)
	if err != nil {
		return err
	}
	if len(encoded) == 0 {
		return nil
	}

	var mvEntries []video.MVEntry
	for _, rfs := range encoded {
		bandwidth := rfs.TargetBitrate * 1000

		var resolution string
		if probe.Video != nil {
			w, h := video.ScaleResolution(
				probe.Video.Width, probe.Video.Height, int(rfs.MaxHeight),
			)
			resolution = video.ResolutionString(w, h)
		}

		r := video.Rendition{Codec: rfs.Codec}
		mvEntries = append(mvEntries, video.MVEntry{
			URI:        "/vidr/" + rfs.ID + ".m3u8",
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
