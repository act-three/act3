package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

type inputSize int

const (
	inputSize2 inputSize = iota // default
	inputSize1
	inputSize3
)

var (
	InputSize1 = html.WithValue(inputSizeKey, inputSize1)
	InputSize2 = html.WithValue(inputSizeKey, inputSize2)
	InputSize3 = html.WithValue(inputSizeKey, inputSize3)
)

var inputSizeClasses = map[inputSize]string{
	inputSize1: "a$input+size-1",
	inputSize2: "a$input+size-2",
	inputSize3: "a$input+size-3",
}

func InputText(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Class("a$input"),
		attr.FuncAttr("class", func(get func(any) any) string {
			s, _ := get(inputSizeKey).(inputSize)
			return inputSizeClasses[s]
		}),
		attr.Group(attrs...),
	)
}

func InputSubmit(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Group(attrs...),
		attr.Type("submit"),
	)
}
