package model

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"time"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/video"
	"ily.dev/act3/video/ffmpeg"
)

// renditionParams is serialized to JSON in RenditionForStreaming.Params.
type renditionParams struct {
	Remux         bool   `json:"remux"`
	Codec         string `json:"codec"`               // "h264" or "hevc"
	TargetBitrate int64  `json:"targetBitrate"`       // kbit/s
	MaxHeight     int    `json:"maxHeight,omitempty"` // 0 = source
	MaxFPS        int    `json:"maxFps,omitempty"`    // 0 = source
	CopyAudio     bool   `json:"copyAudio,omitempty"` // copy audio (source is AAC)
}

func (tx *TxR) taskIngest(ctx Context, args []string) func(*TxRW) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return taskError(err)
	}
	hash, err := tx.m.store.Link(args[1])
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

	// Serialize rendition params for DB storage.
	var rfsParams []schema.RenditionForStreamingCreateParams
	for _, r := range planned {
		b, err := json.Marshal(renditionParams{
			Remux:         r.Remux,
			Codec:         r.Codec,
			TargetBitrate: r.TargetBitrate,
			MaxHeight:     r.MaxHeight,
			MaxFPS:        r.MaxFPS,
			CopyAudio:     r.CopyAudio,
		})
		if err != nil {
			return taskError(err)
		}
		rfsParams = append(rfsParams, schema.RenditionForStreamingCreateParams{
			VideoID: vid.ID,
			Params:  string(b),
		})
	}

	// Compute total encoding work for progress tracking.
	// The 3-phase approach means: remux=1×dur, pass1+pass2=2×dur
	// (regardless of the number of reencode renditions).
	hasRemux, hasEncode := false, false
	for _, r := range planned {
		if r.Remux {
			hasRemux = true
		} else {
			hasEncode = true
		}
	}
	totalWork := time.Duration(0)
	if hasRemux {
		totalWork += probe.Duration
	}
	if hasEncode {
		totalWork += 2 * probe.Duration
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

		tx.m.prog.addVideo(vid.ID, "Encoding", totalWork)
		tx.addTask(ctx, taskIngestDemo, vid.ID)
		return nil
	}
}

func (tx *TxR) taskIngestDemo(ctx Context, args []string) func(*TxRW) error {
	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return taskError(err)
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

	// Parse the rendition params stored in the DB.
	params := make([]renditionParams, len(rfsList))
	for i, rfs := range rfsList {
		if err := json.Unmarshal([]byte(rfs.Params), &params[i]); err != nil {
			return taskError(fmt.Errorf("parse rendition params %s: %w", rfs.ID, err))
		}
	}

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
		for i, p := range params {
			dsts[i] = ffmpeg.EncodeParams{
				File:      dstFiles[i],
				Remux:     p.Remux,
				Codec:     toFFmpegCodec(p.Codec),
				Bitrate:   p.TargetBitrate,
				MaxHeight: p.MaxHeight,
				MaxFPS:    p.MaxFPS,
				Tag:       toVideoTag(p.Codec),
				CopyAudio: p.CopyAudio,
			}
		}
		playlists, err = ffmpeg.Encode(ctx, src, dsts, probe.Duration,
			func(d time.Duration) {
				tx.m.prog.updateVideo(vid.ID, d)
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
			playlists[i], ffmpeg.MediaName(i), hashes[i],
		)
	}

	// Generate the multivariant (master) HLS playlist.
	var mvEntries []video.MVEntry
	for i, p := range params {
		bandwidth := p.TargetBitrate * 1000 // kbit/s → bit/s

		var resolution string
		if probe.Video != nil {
			w, h := video.ScaleResolution(
				probe.Video.Width, probe.Video.Height, p.MaxHeight,
			)
			resolution = video.ResolutionString(w, h)
		}

		r := video.Rendition{Codec: p.Codec}
		mvEntries = append(mvEntries, video.MVEntry{
			URI:        rfsList[i].ID,
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
			Mvplaylist: mvPlaylist,
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
