package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"ily.dev/act3/analysis/errcmp"
	"ily.dev/act3/analysis/noenv"
)

func main() {
	unitchecker.Main(
		noenv.Analyzer,
		errcmp.Analyzer,
	)
}
