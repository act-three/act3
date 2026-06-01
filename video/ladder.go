// Package video contains the video processing pipeline.
// See doc.go for an overview of the approach.
package video

import (
	"fmt"
	"time"

	"ily.dev/act3/priority"
	"ily.dev/act3/video/ffmpeg"
)

// Rendition describes one output variant in the bitrate ladder.
type Rendition struct {
	Remux         bool   // true: copy video stream; false: reencode
	Codec         string // output codec: "h264" or "hevc"
	TargetBitrate int64  // kbit/s (actual source bitrate for remux)
	MaxHeight     int    // output height in pixels; 0 = use source
	MaxFPS        int    // output frame rate cap; 0 = use source
	Priority      int    // encoding order: 0 = best, then worst, middle, rest
}

// FFmpegCodec returns the ffmpeg encoder name for the rendition codec.
func (r *Rendition) FFmpegCodec() string {
	switch r.Codec {
	case "h264":
		return "libx264"
	case "hevc":
		return "libx265"
	}
	return "libx265"
}

// VideoTag returns the fMP4 video tag for the codec ("hvc1" for HEVC).
func (r *Rendition) VideoTag() string {
	if r.Codec == "hevc" {
		return "hvc1"
	}
	return ""
}

// HLSCodecs returns a CODECS string suitable for an HLS
// multivariant playlist EXT-X-STREAM-INF tag.
func (r *Rendition) HLSCodecs() string {
	var vc string
	switch r.Codec {
	case "h264":
		vc = "avc1.640028" // High Profile, Level 4.0
	case "hevc":
		vc = "hvc1.1.6.L150.90" // Main Profile, Main Tier, Level 5.0
	default:
		vc = "hvc1.1.6.L150.90"
	}
	return vc + ",mp4a.40.2" // + AAC-LC
}

// ladder defines the bitrate tiers below the best rendition.
// MaxHeight and MaxFPS are caps: only applied when the source
// exceeds them (resolved at planning time, not encoding time).
// Priority constants are defined in the priority package.
var ladder = []struct {
	Bitrate   int64 // kbit/s
	MaxHeight int   // 0 = source
	MaxFPS    int   // 0 = source
	Priority  int
}{
	{20_000, 0, 0, priority.Encode6th},
	{5_000, 1080, 0, priority.Encode4th},
	{3_000, 720, 0, priority.Encode3rd},
	{1_500, 540, 30, priority.Encode5th},
	{420, 540, 30, priority.Encode2nd},
}

const (
	// topTierCeiling is the maximum source bitrate (kbit/s) below
	// which we remux as the best rendition.
	topTierCeiling = 500_000

	// reencodeThreshold is ~90 % of topTierCeiling.
	// For non-native codecs with source bitrate below this we
	// reencode at ~110 % of the source bitrate.
	reencodeThreshold = 450_000

	// MaxRemuxKeyframeGap is the largest source-keyframe gap we'll
	// tolerate when remuxing the top tier. A stream-copy variant
	// inherits the source's keyframe layout, and HLS segments can
	// only end on keyframes — so a source with a longer gap forces
	// every aligned variant past Apple's recommended 6s ceiling and
	// degrades ABR responsiveness. When the source's max gap exceeds
	// this, the planner falls back to re-encoding the top tier so
	// segments can use the synthetic 4s grid.
	MaxRemuxKeyframeGap = 10 * time.Second
)

// PlanVideoRenditions determines which video renditions to produce
// based on the probed source media. The returned list is ordered
// from highest to lowest bitrate; the first entry is the best
// rendition. Audio output variants are planned separately by
// PlanAudioRenditions.
//
// maxKeyframeGap is the largest distance between consecutive source
// keyframes (0 = unknown). When known and beyond MaxRemuxKeyframeGap,
// the top-tier remux candidate is re-encoded instead, so HLS segment
// length isn't bound by the source's GOP cadence. Pass 0 when the
// source's keyframe layout doesn't matter (e.g. the codec can't be
// remuxed anyway).
//
// Sources whose first packet has no explicit DTS (typically MKV
// h264 with B-frames, since MKV doesn't carry DTS) are also
// excluded from remux. The mp4 muxer would otherwise synthesize a
// DTS shift on stream-copy and write an empty edit-list entry to
// compensate, leaving the resulting fmp4 init segment on a
// different timeline than re-encodes — Chrome MSE refuses to
// bridge the gap on ABR switch (ACT-186).
//
// MaxHeight and MaxFPS are pre-resolved against the source
// properties: they are set to 0 (meaning "use source") when the
// source already satisfies the constraint, so the encoding layer
// does not need access to the source metadata.
//
// Returns nil if the source has no video stream.
func PlanVideoRenditions(probe *ffmpeg.ProbeResult, maxKeyframeGap time.Duration) ([]Rendition, error) {
	if probe.Video == nil {
		return nil, nil
	}

	vs := probe.Video
	srcBitrateKbps := vs.BitRate / 1000 // bits/s → kbit/s
	if srcBitrateKbps == 0 {
		return nil, fmt.Errorf("source video bitrate is unknown")
	}

	canRemux := (vs.CodecName == "h264" || vs.CodecName == "hevc") &&
		(maxKeyframeGap == 0 || maxKeyframeGap <= MaxRemuxKeyframeGap) &&
		vs.HasExplicitDTS

	// Determine output codec: use source codec if remuxing,
	// otherwise reencode everything to HEVC.
	outCodec := "hevc"
	if canRemux && vs.CodecName == "h264" {
		outCodec = "h264"
	}

	var best Rendition
	switch {
	case canRemux && srcBitrateKbps <= topTierCeiling:
		// Remux source as-is.
		best = Rendition{
			Remux:         true,
			Codec:         outCodec,
			TargetBitrate: srcBitrateKbps,
			Priority:      priority.Encode1st,
		}
	case !canRemux && srcBitrateKbps <= reencodeThreshold:
		// Reencode at ~110 % of source bitrate.
		best = Rendition{
			Codec:         outCodec,
			TargetBitrate: srcBitrateKbps * 11 / 10,
			Priority:      priority.Encode1st,
		}
	default:
		// Reencode at top-tier ceiling.
		best = Rendition{
			Codec:         outCodec,
			TargetBitrate: topTierCeiling,
			Priority:      priority.Encode1st,
		}
	}

	renditions := []Rendition{best}

	// FrameRate caps and the low-fps bitrate adjustment compare
	// against the rate the encoder actually receives — coded fps
	// under -fps_mode passthrough, not the container's r_frame_rate.
	// On soft-telecine and other VFR sources these differ (display
	// rate ~60, coded ~24); using display rate would falsely trigger
	// the fps filter and produce variants whose encoder frame counter
	// ticks at a different rate than passthrough variants, leaving
	// SegmentBoundaries no longer aligned across the ladder.
	encFPS := vs.CodedFrameRate()
	if !encFPS.Positive() {
		encFPS = vs.FrameRate
	}

	// Add lower-bitrate renditions from the ladder.
	for _, entry := range ladder {
		bitrate := entry.Bitrate

		// Apply 20 % bitrate reduction for ≤ 25 fps content
		// (per the HLS authoring spec note on low-frame-rate video).
		if encFPS.Positive() && encFPS.Le(25) {
			bitrate = bitrate * 4 / 5
		}

		if bitrate >= best.TargetBitrate {
			continue
		}

		// Resolve MaxHeight: only set when the source exceeds the cap.
		maxH := 0
		if entry.MaxHeight > 0 && vs.Height > entry.MaxHeight {
			maxH = entry.MaxHeight
		}

		// Resolve MaxFPS: only set when the source exceeds the cap.
		maxFPS := 0
		if entry.MaxFPS > 0 && encFPS.Gt(entry.MaxFPS) {
			maxFPS = entry.MaxFPS
		}

		renditions = append(renditions, Rendition{
			Codec:         outCodec,
			TargetBitrate: bitrate,
			MaxHeight:     maxH,
			MaxFPS:        maxFPS,
			Priority:      entry.Priority,
		})
	}

	return renditions, nil
}

// AudioRendition describes one audio output variant.
// SourceStreamIndex matches ffmpeg.AudioStream.Index, so the
// caller can resolve it to a database AudioTrack row.
type AudioRendition struct {
	SourceStreamIndex int
	Channels          int    // 1 (mono), 2 (stereo) or 6 (5.1)
	Bitrate           int64  // kbit/s
	Codec             string // "aac"
	Priority          int
}

// PlanAudioRenditions returns the output variants for each source
// audio stream. Every stream is encoded to AAC as it is, at 64 kbit/s
// per channel:
//   - mono (1 channel) → 64 kbit/s.
//   - stereo (2 channels) → 128 kbit/s.
//   - surround (more than 2 channels) → 5.1 at 384 kbit/s.
//
// As a special case, if there is no stereo track and there is a
// surround source, we generate a stereo downmix, so clients
// without surround output have a stereo option.
//
// Returns nil if the source has no audio.
func PlanAudioRenditions(probe *ffmpeg.ProbeResult) []AudioRendition {
	if len(probe.Audio) == 0 {
		return nil
	}
	// Source tracks that already provide stereo, keyed by title and
	// language, so a surround track matching one can skip its extra
	// downmix.
	type key struct{ title, language string }
	haveStereo := make(map[key]bool)
	for _, as := range probe.Audio {
		if as.Channels == 2 {
			haveStereo[key{as.Title, as.Language}] = true
		}
	}
	var out []AudioRendition
	for _, as := range probe.Audio {
		// Emit the track as it is: mono, stereo, or 5.1.
		asis := AudioRendition{
			SourceStreamIndex: as.Index,
			Codec:             "aac",
			Priority:          priority.EncodeAudio,
		}
		switch {
		case as.Channels <= 1:
			asis.Channels, asis.Bitrate = 1, 64
		case as.Channels == 2:
			asis.Channels, asis.Bitrate = 2, 128
		default:
			asis.Channels, asis.Bitrate = 6, 384
		}
		out = append(out, asis)

		// Add an extra stereo downmix for surround, unless the file
		// already carries an equivalent stereo track.
		if as.Channels > 2 && !haveStereo[key{as.Title, as.Language}] {
			out = append(out, AudioRendition{
				SourceStreamIndex: as.Index,
				Channels:          2,
				Bitrate:           128,
				Codec:             "aac",
				Priority:          priority.EncodeAudio,
			})
		}
	}
	return out
}
