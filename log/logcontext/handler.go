package logcontext

import (
	"context"
	"log/slog"
)

type key int

var argsKey key

type mod struct {
	group string
	args  []any
}

type modStack []mod

func With(ctx context.Context, args ...any) context.Context {
	stack, _ := ctx.Value(argsKey).(modStack)
	stack = append(stack, mod{args: args})
	return context.WithValue(ctx, argsKey, stack)
}

func WithGroup(ctx context.Context, name string) context.Context {
	stack, _ := ctx.Value(argsKey).(modStack)
	stack = append(stack, mod{group: name})
	return context.WithValue(ctx, argsKey, stack)
}

func Handler(h slog.Handler) slog.Handler {
	return handler{h}
}

type handler struct {
	underlying slog.Handler
}

func (h handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.underlying.Enabled(ctx, level)
}

func (h handler) Handle(ctx context.Context, r slog.Record) error {
	stack, _ := ctx.Value(argsKey).(modStack)
	r = r.Clone()
	r.AddAttrs(conv(stack)...)
	return h.underlying.Handle(ctx, r)
}

func (h handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return handler{h.underlying.WithAttrs(attrs)}
}

func (h handler) WithGroup(name string) slog.Handler {
	return handler{h.underlying.WithGroup(name)}
}

func conv(stack []mod) (a []slog.Attr) {
	for _, m := range stack {
		if m.group != "" {
			g := conv(stack[1:])
			a = append(a, slog.GroupAttrs(m.group, g...))
			return a
		}
		a = append(a, argsToAttrSlice(m.args)...)
	}
	return a
}

func argsToAttrSlice(args []any) []slog.Attr {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return attrs
}

const badKey = "!BADKEY"

// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}
		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}
