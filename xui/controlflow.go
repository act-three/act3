package ui

import (
	"fmt"

	"ily.dev/domi"
)

// Empty is an empty view.
// It occupies no space.
// Modifiers applied to it have no effect.
func Empty() View { return base(nil) }

// A Group is a sequence of views.
// It contributes the views to its enclosing view
// as if they had been written there directly.
//
// A modifier applied to a Group is applied to each member individually.
// These are equivalent:
//
//	Group(a, b).Background(Danger).Padding(8)
//	Group(
//	    a.Background(Danger).Padding(8),
//	    b.Background(Danger).Padding(8),
//	)
func Group(v ...View) View {
	var b base
	for _, c := range v {
		b = append(b, c.nodes()...)
	}
	return b
}

// If returns v if cond is true and [Empty] otherwise.
func If(cond bool, v View) View { return IfElse(cond, v, Empty()) }

// IfElse returns a if cond is true and b otherwise.
func IfElse(cond bool, a, b View) View {
	if cond {
		return a
	}
	return b
}

// When calls f and returns the result when cond is true.
// Otherwise, it returns [Empty].
func When(cond bool, f func() View) View { return WhenElse(cond, f, Empty) }

// WhenElse calls a when cond is true
// or else b.
// It returns the result.
func WhenElse(cond bool, a, b func() View) View {
	if cond {
		return a()
	}
	return b()
}

// For calls f once for each item in items,
// and returns the resulting views as a [Group].
//
// If key is not nil,
// For also calls key for each item,
// and uses the returned string
// as the item's key.
// Keyed views are diffed by identity rather than position:
// inserting, removing, or reordering items in the middle of a list
// moves the surviving subviews intact to their new positions
// instead of replacing their contents.
//
// The value of a key must be nonempty,
// stable (any given item should be assigned the same key every time),
// and unique within the enclosing view.
//
// When key is not nil,
// the value returned by f must be a single view;
// if f returns [Empty] or a [Group] with more than one view,
// For panics.
func For[T any, S ~[]T](items S, key func(T) string, f func(T) View) View {
	var b base
	for _, it := range items {
		ns := f(it).nodes()
		if key == nil {
			b = append(b, ns...)
			continue
		}
		if len(ns) != 1 {
			panic(fmt.Sprintf("ui: For item with key %q has %d views, want exactly 1", key(it), len(ns)))
		}
		b = append(b, keyNode{key: key(it), node: ns[0]})
	}
	return b
}

type keyNode struct {
	key  string
	node node
}

func (k keyNode) render(env environment) box {
	b := k.node.render(env)
	b.node = domi.WithKey(k.key, b.node)
	return b
}
