package video

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Eyevinn/hls-m3u8/m3u8"
)

// FixupMediaPlaylist replaces all occurrences of oldName in the
// HLS media playlist text with newName.
// This is used to replace the temporary media filename produced
// by ffmpeg with the content-addressed storage hash.
func FixupMediaPlaylist(playlist, oldName, newName string) string {
	return strings.ReplaceAll(playlist, oldName, newName)
}

// MVEntry describes one variant in a multivariant HLS playlist.
type MVEntry struct {
	URI        string // rendition identifier (resolved to a URL by the HTTP layer)
	Bandwidth  int64  // peak bitrate in bits per second
	Resolution string // e.g. "1920x1080"; empty to omit
	Codecs     string // e.g. "hvc1.1.6.L150.90,mp4a.40.2"; empty to omit
}

// GenerateMVPlaylist builds a multivariant (master) HLS playlist
// from the given entries and returns it as a string.
func GenerateMVPlaylist(entries []MVEntry) string {
	master := m3u8.NewMasterPlaylist()
	for _, e := range entries {
		p := m3u8.VariantParams{
			Bandwidth: uint32(e.Bandwidth),
		}
		if e.Resolution != "" {
			p.Resolution = e.Resolution
		}
		if e.Codecs != "" {
			p.Codecs = e.Codecs
		}
		master.Append(e.URI, nil, p)
	}
	return master.Encode().String()
}

// PeakBitrate computes the peak segment bitrate from an HLS
// media playlist, in bits per second. It parses #EXTINF durations
// and #EXT-X-BYTERANGE sizes, returning the maximum
// (segment_bytes * 8 / segment_duration) across all segments.
// Returns 0 if no segments are found.
func PeakBitrate(playlist string) int64 {
	var peak int64
	var dur float64
	for line := range strings.SplitSeq(playlist, "\n") {
		switch {
		case strings.HasPrefix(line, "#EXTINF:"):
			// #EXTINF:6.131125,
			s := strings.TrimPrefix(line, "#EXTINF:")
			s, _, _ = strings.Cut(s, ",")
			dur, _ = strconv.ParseFloat(s, 64)
		case strings.HasPrefix(line, "#EXT-X-BYTERANGE:"):
			// #EXT-X-BYTERANGE:4792012@3868
			s := strings.TrimPrefix(line, "#EXT-X-BYTERANGE:")
			s, _, _ = strings.Cut(s, "@")
			bytes, _ := strconv.ParseInt(s, 10, 64)
			if dur > 0 && bytes > 0 {
				bps := int64(float64(bytes*8) / dur)
				if bps > peak {
					peak = bps
				}
			}
		}
	}
	return peak
}

// ResolutionString formats a width and height as "WxH".
func ResolutionString(w, h int) string {
	return fmt.Sprintf("%dx%d", w, h)
}

// ScaleResolution computes the output dimensions when capping the
// height to maxH while maintaining the source aspect ratio.
// The output width is rounded down to an even number (required by
// most video codecs). If maxH is 0 or ≥ srcH, the source
// dimensions are returned unchanged.
func ScaleResolution(srcW, srcH, maxH int) (w, h int) {
	if maxH <= 0 || maxH >= srcH {
		return srcW, srcH
	}
	w = srcW * maxH / srcH
	w = w &^ 1 // round down to even
	return w, maxH
}
