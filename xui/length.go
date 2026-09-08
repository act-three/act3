package ui

import (
	"fmt"
	"math"
	"strings"
)

type rect struct{ width, height complex128 }

func checkLength(v complex128) {
	if math.IsNaN(real(v)) || math.IsInf(real(v), 0) ||
		math.IsNaN(imag(v)) || math.IsInf(imag(v), 0) {
		panic(fmt.Sprintf("ui: non-finite length %g", v))
	}
}

func cssLength(v complex128) string {
	checkLength(v)
	px := cssLengthUnit(real(v), "px", 3)
	// Seven rem decimal places can represent each thousandth of a
	// scaled pixel exactly at the 16px basis (0.001 / 16 = 0.0000625).
	rem := cssLengthUnit(imag(v)/16, "rem", 7)
	switch {
	case rem == "0rem":
		return px
	case px == "0px":
		return rem
	case strings.HasPrefix(rem, "-"):
		return "calc(" + px + " - " + rem[1:] + ")"
	default:
		return "calc(" + px + " + " + rem + ")"
	}
}

func cssLengthUnit(v float64, unit string, precision int) string {
	s := fmt.Sprintf("%.*f", precision, v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	if s == "0" || s == "-0" {
		s = "0"
	}
	return s + unit
}
