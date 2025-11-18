package ui

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Spinner() html.Node {
	return html.Div(
		Class(`
			relative
			opacity-65
			[&>span]:absolute
			[&>span]:top-0
			[&>span]:left-[calc(50%-12.5%/2)]
			[&>span]:w-1/8
			[&>span]:h-full
			[&>span]:before:content[""]
			[&>span]:before:block
			[&>span]:before:w-full
			[&>span]:before:h-1/3
			[&>span]:before:rounded-[3px]
			[&>span]:before:bg-current
		`),
		attr.EnvAttr("class", spinnerSizeKey, spinnerMD),
	)(spinnerLeaves...)
}

var spinnerLeaves = func() (a []html.Node) {
	for i := range 8 {
		a = append(a, html.Span(
			Class("animate-spinner-leaf"),
			Class(fmt.Sprintf("%s %s",
				rotate8Table[i], delay8Table[i],
			)),
		))
	}
	return a
}()

var (
	SpinnerSM = html.WithValue(spinnerSizeKey, spinnerSM)
	SpinnerMD = html.WithValue(spinnerSizeKey, spinnerMD) // default
	SpinnerLG = html.WithValue(spinnerSizeKey, spinnerLG)
)

const (
	spinnerSM = "size-3"
	spinnerMD = "size-4"
	spinnerLG = "size-5"
)

var (
	rotate8Table = map[int]string{
		0: "rotate-0",
		1: "rotate-45",
		2: "rotate-90",
		3: "rotate-135",
		4: "rotate-180",
		5: "rotate-225",
		6: "rotate-270",
		7: "rotate-315",
	}

	delay8Table = map[int]string{
		0: "[animation-delay:-800ms]",
		1: "[animation-delay:-700ms]",
		2: "[animation-delay:-600ms]",
		3: "[animation-delay:-500ms]",
		4: "[animation-delay:-400ms]",
		5: "[animation-delay:-300ms]",
		6: "[animation-delay:-200ms]",
		7: "[animation-delay:-100ms]",
	}
)
