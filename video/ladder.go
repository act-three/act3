// Package video contains the video processing pipeline.
// See doc.go for an overview of the approach.
package video

import "ily.dev/act3/video/ffmpeg"

// Rendition describes one output variant in the bitrate ladder.
type Rendition struct {
	Remux         bool   // true: copy video stream; false: reencode
	Codec         string // output codec: "h264" or "hevc"
	TargetBitrate int64  // kbit/s (actual source bitrate for remux)
	MaxHeight     int    // output height in pixels; 0 = use source
	MaxFPS        int    // output frame rate cap; 0 = use source
	CopyAudio     bool   // true if source audio is AAC
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
var ladder = []struct {
	Bitrate   int64 // kbit/s
	MaxHeight int   // 0 = source
	MaxFPS    int   // 0 = source
}{
	{20_000, 0, 0},
	{5_000, 1080, 0},
	{3_000, 720, 0},
	{1_500, 540, 30},
	{420, 540, 30},
}

const (
	// topTierCeiling is the maximum source bitrate (kbit/s) below
	// which we remux as the best rendition.
	topTierCeiling = 500_000

	// reencodeThreshold is ~90 % of topTierCeiling.
	// For non-native codecs with source bitrate below this we
	// reencode at ~110 % of the source bitrate.
	reencodeThreshold = 450_000
)

// PlanRenditions determines which renditions to produce based on
// the probed source media. The returned list is ordered from
// highest to lowest bitrate; the first entry is the best rendition.
//
// MaxHeight and MaxFPS are pre-resolved against the source
// properties: they are set to 0 (meaning "use source") when the
// source already satisfies the constraint, so the encoding layer
// does not need access to the source metadata.
//
// Returns nil if the source has no video stream.
func PlanRenditions(probe *ffmpeg.ProbeResult) []Rendition {
	if probe.Video == nil {
		return nil
	}

	vs := probe.Video
	srcBitrateKbps := vs.BitRate / 1000 // bits/s → kbit/s

	canRemux := vs.CodecName == "h264" || vs.CodecName == "hevc"

	// Determine output codec: use source codec if native,
	// otherwise reencode everything to HEVC.
	outCodec := "hevc"
	if vs.CodecName == "h264" {
		outCodec = "h264"
	}

	copyAudio := false
	if probe.Audio != nil && probe.Audio.CodecName == "aac" {
		copyAudio = true
	}

	// Plan best rendition.
	var best Rendition
	switch {
	case canRemux && srcBitrateKbps <= topTierCeiling:
		// Remux source as-is.
		best = Rendition{
			Remux:         true,
			Codec:         outCodec,
			TargetBitrate: srcBitrateKbps,
			CopyAudio:     copyAudio,
		}
	case !canRemux && srcBitrateKbps <= reencodeThreshold:
		// Reencode at ~110 % of source bitrate.
		best = Rendition{
			Codec:         "hevc",
			TargetBitrate: srcBitrateKbps * 11 / 10,
			CopyAudio:     copyAudio,
		}
	default:
		// Reencode at top-tier ceiling.
		best = Rendition{
			Codec:         "hevc",
			TargetBitrate: topTierCeiling,
			CopyAudio:     copyAudio,
		}
	}

	renditions := []Rendition{best}

	// Add lower-bitrate renditions from the ladder.
	for _, entry := range ladder {
		bitrate := entry.Bitrate

		// Apply 20 % bitrate reduction for ≤ 25 fps content
		// (per the HLS authoring spec note on low-frame-rate video).
		if vs.FrameRate.Positive() && vs.FrameRate.Le(25) {
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
		if entry.MaxFPS > 0 && vs.FrameRate.Gt(entry.MaxFPS) {
			maxFPS = entry.MaxFPS
		}

		renditions = append(renditions, Rendition{
			Codec:         outCodec,
			TargetBitrate: bitrate,
			MaxHeight:     maxH,
			MaxFPS:        maxFPS,
			CopyAudio:     copyAudio,
		})
	}

	return renditions
}
