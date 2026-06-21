package model

import (
	"fmt"

	"ily.dev/act3/database/schema"
)

type AudioTrack struct {
	at schema.AudioTrack
}

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
