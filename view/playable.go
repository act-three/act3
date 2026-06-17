package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

// playForm wraps the play button and any audio/subtitle Selects (named
// "a" and "s") in a form whose submit opens the player for p's video.
func playForm(p model.PlayIDs, controls ...domi.Node) domi.Node {
	children := append([]domi.Node{playFormPinAudio()}, controls...)
	// u-contents: lay the controls out as if the form weren't there.
	return html.Form(
		Class("u-contents"),
		onPlay(p),
		stimulus.Controller("pin-audio"),
	)(children...)
}

func playFormPinAudio() domi.Node {
	return html.Input(
		attr.Type("hidden"),
		attr.Name("pin_audio"),
		stimulus.Target("pin-audio", "pinAudio"),
	)
}

// playButtonForList renders the play button for a collection-playlist
// row, or a disabled button when the row has no playable video.
func playButtonForList(p model.Playable) domi.Node {
	pb := p.PlayIDs()
	if !pb.Playable() {
		return Button(
			ButtonSurface,
			ButtonCircle,
			Inert(true),
		)(Icon("line/x"))
	}
	return playForm(pb,
		Button(
			attr.Type("submit"),
			ButtonSurface,
			ButtonCircle,
		)(Icon("solid/play")),
	)
}

func playableAudioSelect(opts []model.AudioOption) domi.Node {
	if len(opts) == 0 {
		return nil
	}
	init := opts[0]
	return Select(
		attr.Name("a"),
		SelectSize3,
		SelectValue(init.ID),
	)(
		SelectTrigger()(
			Icon("line/recording-01"),
			SelectLabel(audioOptionLabel(init)),
		),
		SelectContent()(
			rangeNodes(opts, func(o model.AudioOption) domi.Node {
				return SelectItem(o.ID)(TruncTail(audioOptionLabel(o)))
			}),
		),
	)
}

func playableSubtitleSelect(opts []model.SubtitleOption) domi.Node {
	if len(opts) == 0 {
		return nil
	}
	return Select(
		attr.Name("s"),
		SelectSize3,
		SelectValue(""),
	)(
		SelectTrigger()(
			Icon("line/message-text-square-02"),
			SelectLabel("Off"),
		),
		SelectContent()(
			SelectItem("")(domi.Text("Off")),
			rangeNodes(opts, func(o model.SubtitleOption) domi.Node {
				return SelectItem(o.ID)(TruncTail(o.Label))
			}),
		),
	)
}

func audioOptionLabel(o model.AudioOption) string {
	return o.Title + " (" + model.OutputChannelLabel(o.Channels) + ")"
}
