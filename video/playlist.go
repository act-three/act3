package video

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Eyevinn/hls-m3u8/m3u8"
)

// subtitleGroupID is the GROUP-ID used for the subtitle alternative
// rendition group referenced by every variant in the MV playlist.
const subtitleGroupID = "subs"

// audioGroupID is the GROUP-ID used for the audio alternative
// rendition group referenced by every variant in the MV playlist.
const audioGroupID = "aud"

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

// MVSubtitle describes one subtitle media entry referenced from a
// multivariant playlist via #EXT-X-MEDIA:TYPE=SUBTITLES.
type MVSubtitle struct {
	URI      string // points at a per-track media playlist (.m3u8)
	Name     string // opaque identifier (emitted as NAME); the player JS matches on it
	Language string // BCP47-ish code (we store ISO 639-2 like "eng")
	Default  bool   // first track of preferred language; client may auto-select
	Forced   bool   // forced narrative
}

// MVAudio describes one audio media entry referenced from a
// multivariant playlist via #EXT-X-MEDIA:TYPE=AUDIO.
type MVAudio struct {
	URI      string // points at a per-track audio media playlist (.m3u8)
	Name     string // opaque identifier (emitted as NAME); the player JS matches on it
	Language string // BCP47-ish code (we store ISO 639-2 like "eng")
	Channels int    // CHANNELS attribute (1, 2, or 6)
	Default  bool   // exactly one Default per group
}

// GenerateMVPlaylist builds a multivariant (root) HLS playlist from
// the given variants, audio alternatives, and subtitle alternatives,
// and returns it as a string. When audios is non-empty, every variant
// gains an AUDIO="aud" group reference; likewise for subtitles via
// SUBTITLES="subs".
//
// EXT-X-MEDIA emission uses the m3u8 library's Alternative type
// (writeExtXMedia in github.com/Eyevinn/hls-m3u8/m3u8/writer.go);
// alternatives are attached to the first variant, since
// MasterPlaylist.GetAllAlternatives dedups across variants and sorts
// by GROUP-ID, TYPE, NAME, LANGUAGE before emitting.
func GenerateMVPlaylist(variants []MVEntry, audios []MVAudio, subtitles []MVSubtitle) string {
	root := m3u8.NewMasterPlaylist()
	var alts []*m3u8.Alternative
	for _, a := range audios {
		alts = append(alts, &m3u8.Alternative{
			Type:       "AUDIO",
			GroupId:    audioGroupID,
			Name:       a.Name,
			Language:   a.Language,
			URI:        a.URI,
			Default:    a.Default,
			Autoselect: true,
			Channels:   &m3u8.Channels{Amount: a.Channels},
		})
	}
	for _, s := range subtitles {
		alts = append(alts, &m3u8.Alternative{
			Type:       "SUBTITLES",
			GroupId:    subtitleGroupID,
			Name:       s.Name,
			Language:   s.Language,
			URI:        s.URI,
			Default:    s.Default,
			Autoselect: !s.Forced,
			Forced:     s.Forced,
		})
	}
	for i, e := range variants {
		p := m3u8.VariantParams{
			Bandwidth: uint32(e.Bandwidth),
		}
		if e.Resolution != "" {
			p.Resolution = e.Resolution
		}
		if e.Codecs != "" {
			p.Codecs = e.Codecs
		}
		if len(audios) > 0 {
			p.Audio = audioGroupID
		}
		if len(subtitles) > 0 {
			p.Subtitles = subtitleGroupID
		}
		if i == 0 && len(alts) > 0 {
			p.Alternatives = alts
		}
		root.Append(e.URI, nil, p)
	}
	return root.Encode().String()
}

// GenerateSubtitleMediaPlaylist builds a per-track HLS subtitle media
// playlist with a single VTT segment covering the whole video.
// vttURI is the URL of the WebVTT file. The output is small enough
// that string concatenation is preferred over the m3u8 library's
// segment API.
func GenerateSubtitleMediaPlaylist(duration time.Duration, vttURI string) string {
	seconds := duration.Seconds()
	target := int64(math.Ceil(seconds))
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	// v3 is the minimum that supports floating-point EXTINF, which is
	// all this playlist needs.
	b.WriteString("#EXT-X-VERSION:3\n")
	b.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	b.WriteString("#EXT-X-TARGETDURATION:")
	b.WriteString(strconv.FormatInt(target, 10))
	b.WriteByte('\n')
	// Optional per spec but some clients require it.
	b.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	b.WriteString("#EXTINF:")
	b.WriteString(strconv.FormatFloat(seconds, 'f', -1, 64))
	b.WriteString(",\n")
	b.WriteString(vttURI)
	b.WriteByte('\n')
	b.WriteString("#EXT-X-ENDLIST\n")
	return b.String()
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
