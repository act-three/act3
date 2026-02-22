package model

import (
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

func (tx *TxR) taskIngest(ctx Context, args []string) func(*TxRW) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return taskError(err)
	}
	hash, err := tx.m.store.Copy(args[1])
	if err != nil {
		return taskError(err)
	}

	// Probe source to get codec info, resolution, bitrate, etc.
	f, err := tx.m.store.Open(hash)
	if err != nil {
		return taskError(err)
	}
	defer f.Close()
	probe, err := ffmpeg.Probe(ctx, f)
	if err != nil {
		return taskError(err)
	}

	// Plan renditions based on the source properties.
	planned, err := video.PlanRenditions(probe)
	if err != nil {
		return taskError(err)
	}
	if len(planned) == 0 {
		return taskError(fmt.Errorf("no video stream found in source"))
	}

	// Build rendition params for DB storage.
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
		})
	}

	return func(tx *TxRW) error {
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

		tx.m.prog.addVideo(vid.ID, "Encoding")
		tx.addTask(ctx, taskIngestEncode, vid.ID)
		return nil
	}
}

func (tx *TxR) taskIngestEncode(ctx Context, args []string) func(*TxRW) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return taskError(err)
	}

	// Register progress tracking. This re-derives the episode→video
	// mappings from the database so progress bars survive restarts.
	evs, err := tx.q.EpisodeVideoListByVideoID(ctx, vid.ID)
	if err != nil {
		return taskError(err)
	}
	tx.m.prog.addVideo(vid.ID, "Encoding")
	for _, ev := range evs {
		tx.m.prog.addEpisodeVideo(ev.EpisodeID, vid.ID)
	}

	rfsList, err := tx.q.RenditionForStreamingListDirectByVideoID(ctx, vid.ID)
	if err != nil {
		return taskError(err)
	}
	if len(rfsList) == 0 {
		return taskError(fmt.Errorf("no renditions planned for video %s", vid.ID))
	}

	src, err := tx.m.store.Open(vid.OriginalHash)
	if err != nil {
		return taskError(err)
	}
	defer src.Close()

	// Probe source for duration (needed for progress tracking).
	probe, err := ffmpeg.Probe(ctx, src)
	if err != nil {
		return taskError(err)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return taskError(err)
	}

	defer tx.m.prog.clearVideo(vid.ID)

	// Encode all renditions.
	var playlists []string
	hashes, err := tx.m.store.CreateNFunc(len(rfsList), func(dstFiles []*os.File) error {
		dsts := make([]ffmpeg.EncodeParams, len(rfsList))
		for i, rfs := range rfsList {
			dsts[i] = ffmpeg.EncodeParams{
				File:          dstFiles[i],
				Remux:         rfs.Remux != 0,
				Codec:         toFFmpegCodec(rfs.Codec),
				Bitrate:       rfs.TargetBitrate,
				MaxHeight:     int(rfs.MaxHeight),
				MaxFPS:        int(rfs.MaxFPS),
				Tag:           toVideoTag(rfs.Codec),
				CopyAudio:     rfs.CopyAudio != 0,
				SurroundAudio: rfs.SurroundAudio != 0,
			}
		}
		playlists, err = ffmpeg.Encode(ctx, src, dsts, probe.Duration,
			func(v float64) {
				tx.m.prog.updateVideo(vid.ID, v)
			},
		)
		return err
	})
	if err != nil {
		return taskError(err)
	}

	// Fixup media playlists: replace temp media filenames with
	// content-addressed storage hashes.
	for i := range playlists {
		playlists[i] = video.FixupMediaPlaylist(
			playlists[i], ffmpeg.MediaName(i), "/vids/"+hashes[i]+".mp4",
		)
	}

	// Generate the multivariant (master) HLS playlist.
	var mvEntries []video.MVEntry
	for i, rfs := range rfsList {
		bandwidth := rfs.TargetBitrate * 1000 // kbit/s → bit/s

		var resolution string
		if probe.Video != nil {
			w, h := video.ScaleResolution(
				probe.Video.Width, probe.Video.Height, int(rfs.MaxHeight),
			)
			resolution = video.ResolutionString(w, h)
		}

		r := video.Rendition{Codec: rfs.Codec}
		mvEntries = append(mvEntries, video.MVEntry{
			URI:        "/vidr/" + rfsList[i].ID + ".m3u8",
			Bandwidth:  bandwidth,
			Resolution: resolution,
			Codecs:     r.HLSCodecs(),
		})
	}
	mvPlaylist := video.GenerateMVPlaylist(mvEntries)

	return func(tx *TxRW) error {
		for i, rfs := range rfsList {
			_, err := tx.q.RenditionForStreamingUpdateEncode(ctx,
				schema.RenditionForStreamingUpdateEncodeParams{
					ID:       rfs.ID,
					Hash:     hashes[i],
					Playlist: playlists[i],
				},
			)
			if err != nil {
				return err
			}
		}
		_, err := tx.q.VideoUpdateMVPlaylist(ctx, schema.VideoUpdateMVPlaylistParams{
			ID:         vid.ID,
			MVPlaylist: mvPlaylist,
		})
		return err
	}
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
