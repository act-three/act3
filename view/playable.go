package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func playButtonForList(p model.Playable) html.Node {
	return Button(
		Href(p.PlayerPath()),
		Attr("data-turbo-frame")("player"),
		ButtonSurface,
		ButtonCircle,
		Inert(p.PlayerPath() == ""),
	)(Icon(playButtonIcon(p)))
}

func playButtonIcon(p model.Playable) string {
	if p.PlayerPath() == "" {
		return "line/x"
	}
	return "solid/play"
}
