package errcmp_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"ily.dev/act3/analysis/errcmp"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errcmp.Analyzer, "a", "gen")
}
