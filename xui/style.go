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

// decls returns f's declarations.
func (f FontSize) decls() []decl {
	size := func(size, weight, height string) []decl {
		return []decl{
			{"font-size", size},
			{"font-weight", weight},
			{"line-height", height},
		}
	}
	switch f {
	case Body:
		return size("1rem", "400", "1.4")
	case Caption:
		return size("0.75rem", "400", "1.3")
	case Headline:
		return size("1.125rem", "600", "1.4")
	case Title:
		return size("1.5rem", "700", "1.2")
	case LargeTitle:
		return size("2rem", "700", "1.15")
	}
	return nil
}

// setStyles adds f's declarations to ss.
func (f FontSize) setStyles(ss *sheet.StyleSet) {
	for _, d := range f.decls() {
		ss.Set(d.property, d.value)
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

func (m FramingMode) objectFit() string {
	switch m {
	case ScaledToFit:
		return "contain"
	case ScaledToFill:
		return "cover"
	}
	panic("unreached")
}
