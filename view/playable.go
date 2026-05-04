package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

// playButtonForList renders a play button for a list-view row. The
// playable controller is attached to the button itself (no Selects);
// on connect it does the audioTracks feature check and stamps
// pin_audio=1 into the href when the browser lacks the API. Every
// page that links to the player needs this check — see
// PlayerForEpisode for the URL contract.
func playButtonForList(p model.Playable) html.Node {
	href := p.PlayerPath()
	if href == "" {
		return Button(
			Attr("data-turbo-frame")("player"),
			ButtonSurface,
			ButtonCircle,
			Inert(true),
		)(Icon("line/x"))
	}
	return Button(
		Href(href),
		Attr("data-turbo-frame")("player"),
		ButtonSurface,
		ButtonCircle,
		playable(href),
		playableTarget,
	)(Icon("solid/play"))
}

// playable wires the play button's href to the audioTracks feature
// check (always) and to the audio/subtitle Selects when present. The
// JS rebuilds the href on connect and on every select:change event
// bubbled up through the controller's element. baseURL is the bare
// /-/player/... path; the JS layers ?a / ?s / ?pin_audio on top.
func playable(baseURL string) attr.Node {
	return attr.Group(
		stimulus.Controller("playable"),
		stimulus.Value("playable", "base-url")(baseURL),
		stimulus.Action("select:change->playable#updateHref"),
	)
}

func playableAudioSelect(opts []model.AudioOption) html.Node {
	if len(opts) == 0 {
		return nil
	}
	init := opts[0]
	return Select(
		SelectSize3,
		SelectValue(init.ID),
		stimulus.Target("playable", "audioSelect"),
	)(
		SelectTrigger()(
			Icon("line/recording-01"),
			SelectLabel(audioOptionLabel(init)),
		),
		SelectContent()(
			html.Range(opts, func(o model.AudioOption) html.Node {
				return SelectItem(o.ID)(html.Text(audioOptionLabel(o)))
			}),
		),
	)
}

func playableSubtitleSelect(opts []model.SubtitleOption) html.Node {
	if len(opts) == 0 {
		return nil
	}
	return Select(
		SelectSize3,
		SelectValue(""),
		stimulus.Target("playable", "subtitleSelect"),
	)(
		SelectTrigger()(
			Icon("line/message-text-square-02"),
			SelectLabel("Off"),
		),
		SelectContent()(
			SelectItem("")(html.Text("Off")),
			html.Range(opts, func(o model.SubtitleOption) html.Node {
				return SelectItem(o.ID)(html.Text(o.Label))
			}),
		),
	)
}

var playableTarget = stimulus.Target("playable", "playButton")

func audioOptionLabel(o model.AudioOption) string {
	return o.Title + " (" + model.OutputChannelLabel(o.Channels) + ")"
}
