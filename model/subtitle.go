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

func (s *SubtitleTrack) ID() string            { return s.st.ID }
func (s *SubtitleTrack) StreamIndex() int      { return int(s.st.StreamIndex) }
func (s *SubtitleTrack) Language() string      { return s.st.Language }
func (s *SubtitleTrack) Title() string         { return s.st.Title }
func (s *SubtitleTrack) OriginalCodec() string { return s.st.OriginalCodec }
func (s *SubtitleTrack) OriginalKey() string   { return s.st.OriginalKey }
func (s *SubtitleTrack) WebVTTKey() string     { return s.st.WebVTTKey }
func (s *SubtitleTrack) Forced() bool          { return s.st.Forced != 0 }

// Label returns a human-readable label like
// "English" or "English (Forced)" or "Track 2".
func (s *SubtitleTrack) Label() string {
	name := s.st.Title
	if name == "" {
		name = langName(s.st.Language)
	}
	if name == "" {
		name = fmt.Sprintf("Track %d", s.st.StreamIndex+1)
	}
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
func (tx *TxR) SubtitleOptions(ctx Context, v *Video) ([]SubtitleOption, error) {
	tracks, err := tx.q.SubtitleTrackListByVideoID(ctx, v.ID())
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

// taskIngestExtractSubs extracts each subtitle track of a video to CAS:
// always a WebVTT artifact, plus an original-codec artifact for codecs
// that have a standalone on-disk format (everything except mov_text).
// Idempotent per-track: a track whose WebVTTKey is already set is
// skipped, so retries after a partial failure resume cleanly.
func (tx *TxR) taskIngestExtractSubs(ctx Context, args []string) (err error) {
	defer errorfmt.Handlef("extract subtitles for video %s: %w", args[0], &err)

	vid, err := tx.q.VideoGet(ctx, args[0])
	if err != nil {
		return err
	}
	tracks, err := tx.q.SubtitleTrackListByVideoID(ctx, vid.ID)
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

	for _, track := range tracks {
		if track.WebVTTKey != "" {
			continue
		}

		webvttKey, err := tx.m.store.CreateFunc(func(dst *os.File) error {
			return ffmpeg.ExtractSubtitleWebVTT(ctx, src, vid.Format, int(track.StreamIndex), dst)
		})
		if err != nil {
			return err
		}

		var originalKey string
		if track.OriginalCodec != "mov_text" {
			originalKey, err = tx.m.store.CreateFunc(func(dst *os.File) error {
				return ffmpeg.ExtractSubtitleOriginal(ctx, src, vid.Format, int(track.StreamIndex), track.OriginalCodec, dst)
			})
			if err != nil {
				return err
			}
		}

		err = tx.m.WithTxRW(func(txw *TxRW) error {
			_, err := txw.q.SubtitleTrackUpdateKeys(ctx, schema.SubtitleTrackUpdateKeysParams{
				ID:          track.ID,
				OriginalKey: originalKey,
				WebVTTKey:   webvttKey,
			})
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}
