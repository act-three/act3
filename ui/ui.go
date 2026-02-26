package ui

import "ily.dev/act3/html"

type key int

const (
	buttonVariantKey key = iota
	buttonSizeKey
	buttonShapeKey
	spinnerSizeKey
	progressSizeKey
	lineClampKey
	fontWeightKey
	textSelectKey
	textSizeKey
	tableVariantKey
	tableSizeKey
	inputSizeKey
	linkUnderlineKey
)

func Group(nodes ...html.Node) html.Node {
	return html.Group(nodes...)
}
