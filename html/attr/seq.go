package attr

import (
	"bytes"
	"fmt"
	"io"
	"iter"

	"ily.dev/act3/html/internal/env"
)

var isCombining = map[string]bool{
	"class": true,
}

// RegisterCombining registeres the named attribute as a
// "combining" attribute.
//
// It must be called before rendering any nodes.
// This is typically done in an [init] function in each package
// that defines custom attributes.
func RegisterCombining(name string) {
	isCombining[name] = true
}

type seq iter.Seq[Node]

// Group returns a Node that renders all items in attr
// separated by spaces.
//
// The returned Node does not repeat attributes:
//
//  1. For each *combining* attribute present,
//     it combines the given values into a single value,
//     separated by spaces.
//     Initially, "class" is the only combining attribute.
//     More can be registered with [RegisterCombining].
//  2. For all other attributes,
//     it outputs only the first occurrence of each.
func Group(attr ...Node) Node {
	return seq(func(yield func(Node) bool) {
		for _, v := range attr {
			if g, ok := v.(seq); ok {
				for vv := range g {
					if vv != nil && !yield(vv) {
						return
					}
				}
			} else if v != nil && !yield(v) {
				return
			}
		}
	})
}

func (s seq) Has(name string) bool {
	for n := range s {
		if n.Has(name) {
			return true
		}
	}
	return false
}

func (s seq) renderTo(w io.Writer, env env.Env) error {
	comb := map[string]*bytes.Buffer{}
	seen := map[string]bool{}
	buf := &bytes.Buffer{}
	for a := range s {
		buf.Reset()
		err := a.renderTo(buf, env)
		if err != nil {
			return err
		}
		nameBytes, rest, found := bytes.Cut(buf.Bytes(), equals)
		name := string(nameBytes)
		if isCombining[name] {
			if !found {
				continue
			}
			rest, found1 := bytes.CutPrefix(rest, dquote)
			rest, found2 := bytes.CutSuffix(rest, dquote)
			if !found1 || !found2 {
				return fmt.Errorf("attr %s requires double quotes: %#q", name, buf.Bytes())
			}
			cbuf, ok := comb[name]
			if ok {
				cbuf.Write(space)
			} else {
				cbuf = &bytes.Buffer{}
				comb[name] = cbuf
			}
			cbuf.Write(rest)
		} else if !seen[name] && name != "" {
			seen[name] = true
			w.Write(space)
			w.Write(buf.Bytes())
		}
	}
	for name, b := range comb {
		w.Write(space)
		io.WriteString(w, name)
		w.Write(equalsQuote)
		w.Write(b.Bytes()) // already escaped
		w.Write(dquote)
	}
	return nil
}

func (seq) attr() {}
