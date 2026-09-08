package ui

import (
	"fmt"
	"math"
	"testing"
)

func TestCSSPx(t *testing.T) {
	for _, tt := range []struct {
		value float64
		want  string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{8, "8px"},
		{1000, "1000px"},
		{3.5, "3.5px"},
		{1000.25, "1000.25px"},
		{0.30000000000000004, "0.3px"},
		{12.345678, "12.346px"},
		{-12.345678, "-12.346px"},
		{0.0004, "0"},
		{-0.0004, "0"},
		{0.0006, "0.001px"},
		{-0.0006, "-0.001px"},
	} {
		if got := cssPx(tt.value); got != tt.want {
			t.Errorf("cssPx(%g) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestNonFiniteLengthPanics(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func(float64)
	}{
		{"Width", func(x float64) { Width(x) }},
		{"Height", func(x float64) { Height(x) }},
		{"MinWidth", func(x float64) { MinWidth(x) }},
		{"MinHeight", func(x float64) { MinHeight(x) }},
		{"IdealWidth", func(x float64) { IdealWidth(x) }},
		{"IdealHeight", func(x float64) { IdealHeight(x) }},
		{"Edges", func(x float64) { Edges(x) }},
		{"EdgesPillarbox", func(x float64) { EdgesPillarbox(x) }},
		{"EdgesLetterbox", func(x float64) { EdgesLetterbox(x) }},
		{"EdgeTop", func(x float64) { EdgeTop(x) }},
		{"EdgeBottom", func(x float64) { EdgeBottom(x) }},
		{"EdgeLeading", func(x float64) { EdgeLeading(x) }},
		{"EdgeTrailing", func(x float64) { EdgeTrailing(x) }},
		{"BorderStroke", func(x float64) { BorderStroke(x, Red) }},
		{"View.BorderStroke", func(x float64) { Text("x").BorderStroke(x, Red) }},
		{"HStack.Gap", func(x float64) { HStack().Gap(x) }},
		{"VStack.Gap", func(x float64) { VStack().Gap(x) }},
		{"Grid.Gap", func(x float64) { Grid(Columns(1)).Gap(x) }},
		{"ColumnMinWidth", func(x float64) { ColumnMinWidth(x) }},
		{"cssPx", func(x float64) { cssPx(x) }},
	} {
		for _, x := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
			t.Run(fmt.Sprintf("%s/%g", tt.name, x), func(t *testing.T) {
				defer wantNonFiniteLengthPanic(t, x)
				// No rendering: the caller must see the panic immediately.
				tt.call(x)
			})
		}
	}
}

func TestNonFiniteEdgeSpacePanics(t *testing.T) {
	for _, x := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
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
					defer wantNonFiniteLengthPanic(t, x)
					method.call(Edges(1), tt.space, Edges(2))
				})
			}
		}
	}
}

func wantNonFiniteLengthPanic(t *testing.T, x float64) {
	t.Helper()
	want := fmt.Sprintf("ui: non-finite pixel length %g", x)
	if got := recover(); got != want {
		t.Errorf("panic = %v, want %s", got, want)
	}
}
