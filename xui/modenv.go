package ui

import (
	"cmp"

	"ily.dev/domi"
)

// A modEnv modifies the environment given to a node.
type modEnv func(environment) environment

func (m modEnv) modify(n node) node {
	return nodeEnv{m, n}
}

func modAttr(attr domi.Attr) modEnv {
	return func(env environment) environment {
		env.add(attr)
		return env
	}
}

// modFixedSize gives the subtree unbounded available space on the given axes,
// so the subtree resolves to its ideal size there (see build).
// The subtree's outermost box is a fill boundary:
// it must sit in its real container at that resolved size,
// even when the container has slack to offer.
func modFixedSize(axes AxisSet) modEnv {
	return func(env environment) environment {
		env.unbounded |= axes
		env.fillMask |= axes
		// Note that a box can override max-content with its own declaration.
		if axes.hasAll(Horizontal) {
			env.style.Set("width", "max-content")
		}
		if axes.hasAll(Vertical) {
			env.style.Set("height", "max-content")
		}
		return env
	}
}

// modStyle emits one CSS declaration onto a subview's outermost box.
// Containers can use it to apply a style to their direct subviews.
func modStyle(property, value string) modEnv {
	return func(env environment) environment {
		env.style.Set(property, value)
		return env
	}
}

func modTagDefault(name string) modEnv {
	return func(env environment) environment {
		env.tag = cmp.Or(env.tag, name)
		return env
	}
}
