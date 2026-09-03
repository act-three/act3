package ui

// A modTransformState modifies the environment given to a node,
// associated with a set of browser states.
type modTransformState func(environment, State) environment

func (m modTransformState) withState(s State) modifier {
	return modTransform(func(env environment) environment {
		return m(env, s)
	})
}

// A modTransform modifies the environment given to a node, like modEnv.
// Additionally, if the environment has any paint modifiers
// (as indicated by nextenv.hasPaint),
// it first emits a layout-preserving wrapper box
// that applies all env modifiers,
// thus clearing any pending paint modifiers
// before applying the transform and rendering the node.
type modTransform func(environment) environment

func (m modTransform) modify(n node) node {
	return func(env environment) box {
		if !env.hasPaint {
			return n(m(env))
		}
		return wrapMod(env, modEnv(m).modify(n))
	}
}

// BorderShape sets the shape of a view's border.
func BorderShape(s Shape) Modifier {
	return modTransformState(func(env environment, state State) environment {
		env.shape = append(env.shape, term[Shape]{state, s})
		return env
	})
}

// Opacity sets a view's opacity to x, from 0 (transparent) to 1 (opaque).
func Opacity(x float64) Modifier {
	return modTransformState(func(env environment, s State) environment {
		env.opacity = append(env.opacity, term[float64]{s, x})
		return env
	})
}
