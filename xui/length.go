package ui

import (
	"fmt"
	"math"
	"strings"
)

type rect struct{ width, height float64 }

func checkLength(v float64) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		panic(fmt.Sprintf("ui: non-finite pixel length %g", v))
	}
}

func cssPx(v float64) string {
	checkLength(v)
	s := fmt.Sprintf("%.3f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	if s == "0" || s == "-0" {
		return "0"
	}
	return s + "px"
}
