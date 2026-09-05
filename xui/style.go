package ui

// FontSize selects a slot on the type scale.
type FontSize string

const (
	Body         FontSize = "body"
	Caption      FontSize = "caption"
	HeadlineFont FontSize = "headline"
	Title        FontSize = "title"
	LargeTitle   FontSize = "large-title"
)

// values returns the font size, weight, and line height of the slot,
// or empty strings if f is not defined.
func (f FontSize) values() (size, weight, height string) {
	switch f {
	case Body:
		return "1rem", "400", "1.4"
	case Caption:
		return "0.75rem", "400", "1.3"
	case HeadlineFont:
		return "1.125rem", "600", "1.4"
	case Title:
		return "1.5rem", "700", "1.2"
	case LargeTitle:
		return "2rem", "700", "1.15"
	}
	return "", "", ""
}
