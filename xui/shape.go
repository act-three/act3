package ui

// A Shape is a border shape for [View.BorderShape].
type Shape int

const (
	Rectangle Shape = iota
	RoundedRectangle
	Ellipse
	Capsule
)

// radius returns the CSS border-radius value for s.
func (s Shape) radius() string {
	switch s {
	case Ellipse:
		return "50%"
	case RoundedRectangle:
		return "var(--ui-radius)"
	case Capsule:
		return "9999px"
	default:
		return "0"
	}
}
