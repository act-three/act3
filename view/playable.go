package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func playButtonForList(p model.Playable) html.Node {
	return Button(
		attr.Href(p.PlayerPath()),
		attr.Attr("data-turbo-frame")("player"),
		ButtonSurface,
		ButtonCircle,
		Disabled(p.PlayerPath() == ""),
	)(Icon(playButtonIcon(p)))
}

func playButtonIcon(p model.Playable) string {
	if p.PlayerPath() == "" {
		return "line/x-close"
	}
	return "solid/play"
}
