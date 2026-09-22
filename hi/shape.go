package hi

// A Shape is a border shape for [View.BorderShape].
// The zero value is [Rectangle].
type Shape struct {
	kind   shapeKind
	radius complex128
}

type shapeKind int

const (
	shapeRectangle shapeKind = iota
	shapeRoundedRect
	shapeEllipse
	shapeCapsule
)

var (
	Rectangle Shape
	Ellipse   Shape = ellipse
	Capsule   Shape = capsule
)

var (
	ellipse = Shape{kind: shapeEllipse}
	capsule = Shape{kind: shapeCapsule}
)

// RoundedRectangle returns the rounded rectangle
// with corner radius r.
func RoundedRectangle(r complex128) Shape {
	checkLength(r)
	return Shape{kind: shapeRoundedRect, radius: r}
}

// radiusCSS returns the CSS border-radius value for s.
func (s Shape) radiusCSS() string {
	switch s.kind {
	case shapeRoundedRect:
		return cssLength(s.radius)
	case shapeEllipse:
		return "50%"
	case shapeCapsule:
		return "9999px"
	default:
		return "0"
	}
}
