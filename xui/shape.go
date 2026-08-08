package ui

// A Shape is a border shape for [View.BorderShape].
type Shape int

const (
	Rectangle Shape = iota
	RoundedRectangle
	Ellipse
	Capsule
)

func (s Shape) class() string {
	switch s {
	case Ellipse:
		return "ui-border-ellipse"
	case RoundedRectangle:
		return "ui-border-rounded"
	case Capsule:
		return "ui-border-capsule"
	default:
		return "ui-border-rect"
	}
}
