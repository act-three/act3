package ui

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

func (m FramingMode) css() string {
	switch m {
	case ScaledToFit:
		return "contain"
	case ScaledToFill:
		return "cover"
	}
	panic("unreached")
}
