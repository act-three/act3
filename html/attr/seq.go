package attr

import (
	"bytes"
	"fmt"
	"io"
	"iter"
)

var combining = map[string]string{
	"class": " ",
	"style": ";",
}

// RegisterCombining registers name as a "combining" attribute.
// When a combining attribute appears more than once in an html node,
// the values are combined, separated by sep,
// into a single attribute in the rendered output.
//
// RegisterCombining must be called before Render.
// This is typically done in an [init] function in packages
// that define custom attributes.
//
// The initial set of combining attributes is:
//
//	class " "
//	style ";"
func RegisterCombining(name, sep string) {
	combining[name] = sep
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

func (s seq) renderTo(w io.Writer) error {
	comb := map[string]*bytes.Buffer{}
	seen := map[string]bool{}
	buf := &bytes.Buffer{}
	for a := range s {
		buf.Reset()
		err := a.renderTo(buf)
		if err != nil {
			return err
		}
		nameBytes, rest, found := bytes.Cut(buf.Bytes(), equals)
		name := string(nameBytes)
		if sep, ok := combining[name]; ok {
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
				cbuf.WriteString(sep)
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
