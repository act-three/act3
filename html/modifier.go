package html

import (
	"io"

	"ily.dev/act3/html/internal/env"
)

// A Modifier can be passed to [Node.With]
// to modify the render environment for a node.
type Modifier struct {
	f func(env.Env) env.Env
}

// WithValue returns a Modifier that sets the given key-value pair.
func WithValue(key, value any) Modifier {
	return Modifier{func(e env.Env) env.Env {
		return env.WithValue(e, key, value)
	}}
}

type modNode struct {
	n Node
	f func(env.Env) env.Env
}

func modify(n Node, m Modifier) Node {
	return modNode{n, m.f}
}

func (n modNode) renderTo(w io.Writer, env env.Env) error {
	env = n.f(env)
	return n.n.renderTo(w, env)
}

func (n modNode) With(m Modifier) Node { return modify(n, m) }
