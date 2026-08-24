package ui

import "ily.dev/act3/xui/internal/sheet"

// FontSize selects a slot on the type scale.
type FontSize string

const (
	Body       FontSize = "body"
	Caption    FontSize = "caption"
	Headline   FontSize = "headline"
	Title      FontSize = "title"
	LargeTitle FontSize = "large-title"
)

// values returns the font size, weight, and line height of the slot,
// or empty strings if f is not defined.
func (f FontSize) values() (size, weight, height string) {
	switch f {
	case Body:
		return "1rem", "400", "1.4"
	case Caption:
		return "0.75rem", "400", "1.3"
	case Headline:
		return "1.125rem", "600", "1.4"
	case Title:
		return "1.5rem", "700", "1.2"
	case LargeTitle:
		return "2rem", "700", "1.15"
	}
	return "", "", ""
}

// setStyles adds f's declarations to ss.
func (f FontSize) setStyles(ss *sheet.StyleSet) {
	size, weight, height := f.values()
	if size == "" {
		return
	}
	ss.Set("font-size", size)
	ss.Set("font-weight", weight)
	ss.Set("line-height", height)
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

func (m FramingMode) objectFit() string {
	switch m {
	case ScaledToFit:
		return "contain"
	case ScaledToFill:
		return "cover"
	}
	panic("unreached")
}
