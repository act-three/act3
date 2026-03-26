package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/turbo"
)

// LiveText renders an inline element containing text.
// It can be updated in place using turbo streams.
// See LiveTextUpdate.
//
// The addr values must identify the text
// with sufficient precision to update it reliably.
// For instance, if text is the contents of a database record,
// addr might contain three parts:
// the table name, the primary key of the record, and the field name.
//
//	LiveText("me@example.org", "users", "345", "email")
//
// However, addr can be anything, and text doesn't have to be a database field.
// The text can be derived or synthesized,
// as long as addr is used consistently and is unambiguous.
func LiveText(text string, addr ...string) html.Node {
	var attrs attr.Node = attr.Attr("data-live")
	for i, a := range addr {
		attrs = attr.Group(attrs, attr.Attr(fmt.Sprintf("data-addr%d", i))(a))
	}
	return html.Span(attrs)(html.Text(text))
}

// LiveTextUpdate renders a turbo streams item that updates
// text previously rendered by LiveText.
// Values in addr must match the values given to LiveText,
// and the contents will be replaced with the new text value.
func LiveTextUpdate(text string, addr ...string) html.Node {
	sel := &strings.Builder{}
	sel.WriteString("[data-live]")
	for i, a := range addr {
		fmt.Fprintf(sel, `[data-addr%d="%s"]`, i, cssEscape(a))
	}
	return turbo.ReplaceTargets(sel.String(), turbo.Morph)(
		LiveText(text, addr...),
	)
}

// cssEscape escapes a string for use inside a CSS double-quoted string.
// It escapes backslash, double quote, and control characters
// using CSS's backslash-hex escape format.
func cssEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '\\' || r == '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\%x `, r)
		default:
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}
