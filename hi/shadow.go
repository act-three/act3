package hi

import (
	"slices"
	"strings"
)

// BorderShadow draws a shadow of the given color
// around the outside of a view's border.
//
// The distances dx and dy move the shadow right and down.
// The spread distance makes the shadow bigger.
// The blur radius blurs the shadow image.
//
// The shadow is drawn exclusively outside the view's border shape.
// It takes no layout space.
// If any of the view's content overlaps with the shadow,
// the shadow is drawn behind it.
func BorderShadow(dx, dy, spread, blur complex128, c Color) Modifier {
	checkLength(dx)
	checkLength(dy)
	checkLength(blur)
	checkLength(spread)
	return modEnvState(func(env environment, state State) environment {
		s := shadow{dx, dy, blur, spread, c.color()}
		env.shadow = append(env.shadow, term[shadow]{state, s})
		env.hasPaint = true
		return env
	})
}

type shadow struct {
	dx, dy complex128
	blur   complex128
	spread complex128
	color  color
}

func borderShadowList(t theme, shadows []shadow) string {
	var layers []string
	// Traversal records outer modifiers first; CSS lists frontmost first.
	for _, s := range slices.Backward(shadows) {
		blur := cssLength(s.blur)
		if real(s.blur) < 0 || imag(s.blur) < 0 {
			blur = "max(0px," + blur + ")"
		}
		layers = append(layers, cssLength(s.dx)+" "+cssLength(s.dy)+" "+blur+" "+
			cssLength(s.spread)+" "+s.color.colorCoords(t).css())
	}
	if len(layers) == 0 {
		return "none"
	}
	return strings.Join(layers, ",")
}
