package video

import (
	"fmt"
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
