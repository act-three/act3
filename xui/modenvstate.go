package ui

// A modEnvState modifies the environment given to a node,
// associated with a set of browser states.
type modEnvState func(environment, State) environment

func (m modEnvState) withState(s State) modifier {
	return modEnv(func(env environment) environment {
		return m(env, s)
	})
}

// Background fills the background of a view with c.
func Background(c Color) Modifier {
	return modEnvState(func(env environment, s State) environment {
		env.bg = append(env.bg, term[color]{s, c.color()})
		env.hasPaint = true
		return env
	})
}

// BorderStroke draws a line
// of the given width and color
// over the inside edge of a view.
//
// The stroke paints over the view's content,
// inside its border shape.
// It takes no layout space.
// To add a border around the outside of a view,
// add padding inside the border.
func BorderStroke(px float64, c Color) Modifier {
	if !(px > 0) { // this is written weird b/c of NaNs lmao
		px = 0
	}
	return modEnvState(func(env environment, s State) environment {
		env.stroke = append(env.stroke, term[stroke]{s, stroke{px, c.color()}})
		env.hasPaint = true
		return env
	})
}

// Font sets the font size for text in a view.
func Font(f FontSize) Modifier {
	if f == "" {
		return nil
	}
	return modEnvState(func(env environment, s State) environment {
		size, weight, height := f.values()
		env.fontSize = append(env.fontSize, term[string]{s, size})
		env.fontWeight = append(env.fontWeight, term[string]{s, weight})
		env.lineHeight = append(env.lineHeight, term[string]{s, height})
		return env
	})
}

// Foreground uses c to draw foreground elements in a view,
// such as text.
func Foreground(c Color) Modifier {
	return modEnvState(func(env environment, s State) environment {
		env.fg = append(env.fg, term[color]{s, c.color()})
		return env
	})
}
