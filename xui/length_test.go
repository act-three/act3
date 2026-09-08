package ui

import (
	"fmt"
	"math"
	"testing"
)

func TestCSSLength(t *testing.T) {
	for _, tt := range []struct {
		value complex128
		want  string
	}{
		{0, "0px"},
		{complex(math.Copysign(0, -1), 0), "0px"},
		{8, "8px"},
		{1000, "1000px"},
		{3.5, "3.5px"},
		{1000.25, "1000.25px"},
		{0.30000000000000004, "0.3px"},
		{12.345678, "12.346px"},
		{-12.345678, "-12.346px"},
		{0.0004, "0px"},
		{-0.0004, "0px"},
		{0.0006, "0.001px"},
		{-0.0006, "-0.001px"},
		{1i, "0.0625rem"},
		{0.001i, "0.0000625rem"},
		{0.30000000000000004i, "0.01875rem"},
		{1 + 1i, "calc(1px + 0.0625rem)"},
		{1 - 1i, "calc(1px - 0.0625rem)"},
		{0.0004 + 1i, "0.0625rem"},
		{1 + 0.0000001i, "1px"},
		{complex(0, math.Copysign(0, -1)), "0px"},
	} {
		if got := cssLength(tt.value); got != tt.want {
			t.Errorf("cssLength(%g) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestNonFiniteLengthPanics(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func(complex128)
	}{
		{"Width", func(x complex128) { Width(x) }},
		{"Height", func(x complex128) { Height(x) }},
		{"MinWidth", func(x complex128) { MinWidth(x) }},
		{"MinHeight", func(x complex128) { MinHeight(x) }},
		{"IdealWidth", func(x complex128) { IdealWidth(x) }},
		{"IdealHeight", func(x complex128) { IdealHeight(x) }},
		{"Edges", func(x complex128) { Edges(x) }},
		{"EdgesPillarbox", func(x complex128) { EdgesPillarbox(x) }},
		{"EdgesLetterbox", func(x complex128) { EdgesLetterbox(x) }},
		{"EdgeTop", func(x complex128) { EdgeTop(x) }},
		{"EdgeBottom", func(x complex128) { EdgeBottom(x) }},
		{"EdgeLeading", func(x complex128) { EdgeLeading(x) }},
		{"EdgeTrailing", func(x complex128) { EdgeTrailing(x) }},
		{"BorderStroke", func(x complex128) { BorderStroke(x, Red) }},
		{"View.BorderStroke", func(x complex128) { Text("x").BorderStroke(x, Red) }},
		{"HStack.Gap", func(x complex128) { HStack().Gap(x) }},
		{"VStack.Gap", func(x complex128) { VStack().Gap(x) }},
		{"Grid.Gap", func(x complex128) { Grid(Columns(1)).Gap(x) }},
		{"ColumnMinWidth", func(x complex128) { ColumnMinWidth(x) }},
		{"cssLength", func(x complex128) { cssLength(x) }},
	} {
		for _, x := range nonFiniteLengths() {
			t.Run(fmt.Sprintf("%s/%g", tt.name, x), func(t *testing.T) {
				defer wantNonFiniteLengthPanic(t, x)
				// No rendering: the caller must see the panic immediately.
				tt.call(x)
			})
		}
	}
}

func TestNonFiniteEdgeSpacePanics(t *testing.T) {
	for _, x := range nonFiniteLengths() {
		for _, tt := range []struct {
			name  string
			space EdgeSpace
		}{
			{"Top", EdgeSpace{Top: x}},
			{"Bottom", EdgeSpace{Bottom: x}},
			{"Leading", EdgeSpace{Leading: x}},
			{"Trailing", EdgeSpace{Trailing: x}},
		} {
			for _, method := range []struct {
				name string
				call func(...EdgeSpace) View
			}{
				{"Padding", Text("x").Padding},
				{"Sticky", Text("x").Sticky},
			} {
				t.Run(fmt.Sprintf("%s/%s/%g", method.name, tt.name, x), func(t *testing.T) {
					defer wantNonFiniteLengthPanic(t, x+3)
					method.call(Edges(1), tt.space, Edges(2))
				})
			}
		}
	}
}

func wantNonFiniteLengthPanic(t *testing.T, x complex128) {
	t.Helper()
	want := fmt.Sprintf("ui: non-finite length %g", x)
	if got := recover(); got != want {
		t.Errorf("panic = %v, want %s", got, want)
	}
}

func nonFiniteLengths() []complex128 {
	return []complex128{
		complex(math.NaN(), 0), complex(math.Inf(1), 0), complex(math.Inf(-1), 0),
		complex(0, math.NaN()), complex(0, math.Inf(1)), complex(0, math.Inf(-1)),
	}
}
