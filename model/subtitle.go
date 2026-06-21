package model

import (
	"fmt"
	"os"

	"kr.dev/errorfmt"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/video/ffmpeg"
)

type SubtitleTrack struct {
	st schema.SubtitleTrack
}

// Name returns a human-readable name without any forced-narrative
// suffix. Use this when an out-of-band signal already conveys the
// forced flag (e.g. HLS FORCED=YES).
func (s *SubtitleTrack) Name() string {
	name := s.st.Title
	if name == "" {
		name = langName(s.st.Language)
	}
	if name == "" {
		name = fmt.Sprintf("Track %d", s.st.StreamIndex+1)
	}
	return name
}

// Label returns Name with " (Forced)" appended for forced tracks.
// Use this for in-app UI (menus) where there's no other forced-flag
// indicator.
func (s *SubtitleTrack) Label() string {
	name := s.Name()
	if s.st.Forced != 0 {
		name += " (Forced)"
	}
	return name
}

// SubtitleOption describes one entry in the player captions menu.
type SubtitleOption struct {
	ID            string // SubtitleTrack ID
	Label         string // human-readable for the menu
	Language      string // ISO 639-2 code as stored
	OriginalCodec string // browser routes by this: "ass"/"ssa" → ASS renderer, others → native <track>
	WebVTTPath    string // URL for the .vtt
	OriginalPath  string // URL for the original-codec file; empty when codec was mov_text
	Forced        bool
}

// SubtitleOptions returns the captions menu entries for a video.
// Tracks whose extraction has not completed (no WebVTTKey) are skipped.
func (tx *TxR) SubtitleOptions(v *Video) ([]SubtitleOption, error) {
	tracks, err := tx.q.SubtitleTrackListByVideoID(v.ID())
	if err != nil {
		return nil, err
	}
	var opts []SubtitleOption
	for _, st := range tracks {
		if st.WebVTTKey == "" {
			continue
		}
		opt := SubtitleOption{
			ID:            st.ID,
			Label:         (&SubtitleTrack{st: st}).Label(),
			Language:      st.Language,
			OriginalCodec: st.OriginalCodec,
			WebVTTPath:    "/-/sub/" + st.ID + ".vtt",
			Forced:        st.Forced != 0,
		}
		if st.OriginalKey != "" {
			opt.OriginalPath = "/-/sub/" + st.ID + "." + subtitleOriginalExt(st.OriginalCodec)
		}
		opts = append(opts, opt)
	}
	return opts, nil
}

// subtitleOriginalExt maps an original-codec name to the URL extension
// served by the web layer. Mirrors ffmpeg's standalone muxer choices in
// ExtractSubtitleOriginal: ssa shares the ass muxer, subrip is .srt.
// mov_text has no original artifact and never reaches this function.
func subtitleOriginalExt(codec string) string {
	switch codec {
	case "ass", "ssa":
		return "ass"
	case "subrip":
		return "srt"
	case "webvtt":
		return "vtt"
	}
	return codec
}

// SubtitleOriginalExt is the exported counterpart to subtitleOriginalExt
// for use by the web layer when matching a request URL suffix to a track.
func SubtitleOriginalExt(codec string) string {
	return subtitleOriginalExt(codec)
}

// SubtitleContentType returns the response Content-Type to pin when
// serving the original-format subtitle blob for a given codec. WebVTT
// is text/vtt; ASS/SSA share text/x-ssa; SubRip is application/x-subrip.
// All formats are UTF-8 (we transcode at extraction time).
func SubtitleContentType(codec string) string {
	switch codec {
	case "ass", "ssa":
		return "text/x-ssa; charset=utf-8"
	case "subrip":
		return "application/x-subrip; charset=utf-8"
	case "webvtt":
		return "text/vtt; charset=utf-8"
	}
	return "application/octet-stream"
}

// Subtitle returns the SubtitleTrack row for the given ID.
func (tx *TxR) Subtitle(id string) (schema.SubtitleTrack, error) {
	return tx.q.SubtitleTrackGet(id)
}

// taskIngestExtractSubs extracts each subtitle track of a video to CAS:
// always a WebVTT artifact, plus an original-codec artifact for codecs
// that have a standalone on-disk format (everything except mov_text).
// Idempotent per-track: a track whose WebVTTKey is already set is
// skipped, so retries after a partial failure resume cleanly.
func (tx *TxR) taskIngestExtractSubs(args []string) (err error) {
	defer errorfmt.Handlef("extract subtitles for video %s: %w", args[0], &err)

	vid, err := tx.q.VideoGet(args[0])
	if err != nil {
		return err
	}
	tracks, err := tx.q.SubtitleTrackListByVideoID(vid.ID)
	if err != nil {
		return err
	}
	if len(tracks) == 0 {
		return nil
	}

	src, err := tx.m.store.Open(vid.OriginalKey)
	if err != nil {
		return err
	}
	defer src.Close()

	var updated bool
	for _, track := range tracks {
		if track.WebVTTKey != "" {
			continue
		}

		webvttKey, err := tx.m.store.CreateFunc(func(dst *os.File) error {
			return ffmpeg.ExtractSubtitleWebVTT(tx.ctx, src, vid.Format, int(track.StreamIndex), dst)
		})
		if err != nil {
			return err
		}

		var originalKey string
		if track.OriginalCodec != "mov_text" {
			originalKey, err = tx.m.store.CreateFunc(func(dst *os.File) error {
				return ffmpeg.ExtractSubtitleOriginal(tx.ctx, src, vid.Format, int(track.StreamIndex), track.OriginalCodec, dst)
			})
			if err != nil {
				return err
			}
		}

		err = tx.m.WithTxRW(tx.ctx, func(txw *TxRW) error {
			_, err := txw.q.SubtitleTrackUpdateKeys(schema.SubtitleTrackUpdateKeysParams{
				ID:          track.ID,
				OriginalKey: originalKey,
				WebVTTKey:   webvttKey,
			})

			return err
		})
		if err != nil {
			return err
		}
		updated = true
	}

	// Rebuild the MV playlist once, after the per-track loop, so the
	// new subtitle entries appear alongside the existing variants.
	// One rebuild per task keeps DB writes minimal; for text-only
	// extraction the per-track latency cost is microseconds.
	if updated {
		err = tx.m.WithTxRW(tx.ctx, func(txw *TxRW) error {
			return txw.recomputePlayable(vid)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
