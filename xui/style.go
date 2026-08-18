package ui

import "ily.dev/act3/xui/internal/sheet"

// FontSize selects a slot on the type scale.
// The zero FontSize is the body font.
type FontSize string

const (
	Body       FontSize = ""
	Caption    FontSize = "caption"
	Headline   FontSize = "headline"
	Title      FontSize = "title"
	LargeTitle FontSize = "large-title"
)

func (f FontSize) class() string {
	if f == "" {
		return ""
	}
	return "ui-font-" + string(f)
}

// setStyles adds f's declarations to ss.
func (f FontSize) setStyles(ss *sheet.StyleSet) {
	switch f {
	case Caption:
		ss.Set("font-size", "0.75rem")
		ss.Set("line-height", "1.3")
	case Headline:
		ss.Set("font-size", "1.125rem")
		ss.Set("font-weight", "600")
	case Title:
		ss.Set("font-size", "1.5rem")
		ss.Set("font-weight", "700")
		ss.Set("line-height", "1.2")
	case LargeTitle:
		ss.Set("font-size", "2rem")
		ss.Set("font-weight", "700")
		ss.Set("line-height", "1.15")
	}
}

// A FramingMode controls how an [Image] fills its available space.
type FramingMode int

const (
	// Native displays the image at its native size.
	Native FramingMode = iota

	// ScaledToFit displays the complete image, preserving its aspect
	// ratio while expanding as far as possible within the available
	// space. This is also known as "letterboxed" and "pillarboxed".
	ScaledToFit

	// ScaledToFill crops the image to fill its available space and
	// display as much of the image as possible without distortion.
	ScaledToFill
)

func (m FramingMode) class() string {
	switch m {
	case ScaledToFit:
		return "ui-fm-contain"
	case ScaledToFill:
		return "ui-fm-cover"
	}
	panic("unreached")
}
