package model

import (
	"fmt"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/video"
)

type AudioTrack struct {
	at schema.AudioTrack
}

func (a *AudioTrack) ID() string            { return a.at.ID }
func (a *AudioTrack) StreamIndex() int      { return int(a.at.StreamIndex) }
func (a *AudioTrack) Language() string      { return a.at.Language }
func (a *AudioTrack) Title() string         { return a.at.Title }
func (a *AudioTrack) Channels() int         { return int(a.at.Channels) }
func (a *AudioTrack) ChannelLayout() string { return a.at.ChannelLayout }
func (a *AudioTrack) SampleRate() int       { return int(a.at.SampleRate) }
func (a *AudioTrack) Codec() string         { return a.at.Codec }
func (a *AudioTrack) Profile() string       { return a.at.Profile }

// Name returns the human-readable name of the track without any
// channel-layout suffix: the source title, falling back to a
// language name, falling back to "Track N".
func (a *AudioTrack) Name() string {
	name := a.at.Title
	if name == "" {
		name = langName(a.at.Language)
	}
	if name == "" {
		name = fmt.Sprintf("Track %d", a.at.StreamIndex+1)
	}
	return name
}

// Label returns a human-readable label like
// "English (5.1)" or "Track 1 (Stereo)" describing the source
// track's channel layout. For an output rendition's display label,
// compose Name() with OutputChannelLabel(rendition.Channels) so the
// suffix reflects the rendition rather than the source.
func (a *AudioTrack) Label() string {
	layout := layoutLabel(a.at.Channels, a.at.ChannelLayout)
	return a.Name() + " (" + layout + ")"
}

// OutputChannelLabel returns a human-readable label for an output
// rendition's channel count: "Mono", "Stereo", "5.1".
func OutputChannelLabel(channels int) string {
	switch channels {
	case 1:
		return "Mono"
	case 2:
		return "Stereo"
	case 6:
		return "5.1"
	}
	return fmt.Sprintf("%dch", channels)
}

// AudioMenuLabel returns the label that must match between the HLS
// EXT-X-MEDIA NAME attribute and the player audio menu's
// data-player-audio-label-param. Run through video.SanitizeAttrString
// so a source title containing `"` produces the same byte sequence on
// both sides — the player matches by string equality.
func AudioMenuLabel(title string, channels int) string {
	return video.SanitizeAttrString(title + " (" + OutputChannelLabel(channels) + ")")
}

func layoutLabel(channels int64, layout string) string {
	switch {
	case strings.Contains(layout, "5.1"):
		return "5.1"
	case strings.Contains(layout, "7.1"):
		return "7.1"
	case channels <= 1:
		return "Mono"
	case channels == 2:
		return "Stereo"
	default:
		return fmt.Sprintf("%dch", channels)
	}
}

func langName(code string) string {
	switch code {
	case "eng":
		return "English"
	case "jpn":
		return "Japanese"
	case "spa":
		return "Spanish"
	case "fre", "fra":
		return "French"
	case "ger", "deu":
		return "German"
	case "ita":
		return "Italian"
	case "por":
		return "Portuguese"
	case "rus":
		return "Russian"
	case "kor":
		return "Korean"
	case "chi", "zho":
		return "Chinese"
	case "ara":
		return "Arabic"
	case "hin":
		return "Hindi"
	case "und":
		return ""
	default:
		return code
	}
}
