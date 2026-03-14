package model

import (
	"fmt"
	"strings"

	"ily.dev/act3/database/schema"
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
func (a *AudioTrack) Codec() string         { return a.at.Codec }

// Label returns a human-readable label like
// "English (5.1)" or "Track 1 (Stereo)".
func (a *AudioTrack) Label() string {
	name := a.at.Title
	if name == "" {
		name = langName(a.at.Language)
	}
	if name == "" {
		name = fmt.Sprintf("Track %d", a.at.StreamIndex+1)
	}
	layout := layoutLabel(a.at.Channels, a.at.ChannelLayout)
	return name + " (" + layout + ")"
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
