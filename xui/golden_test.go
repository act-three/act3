package ui_test

import (
	"flag"
	"os"
	"testing"

	"ily.dev/act3/xui/internal/fixture"
)

var update = flag.Bool("update", false, "rewrite golden files with current output")

// TestGolden pins the fixture page's rendered HTML, stylesheet included, so
// any change to the lowering or the CSS shows up as a reviewable golden diff.
// Refresh with: go test ./xui -run TestGolden -update
func TestGolden(t *testing.T) {
	got, err := fixture.Document(staticCSS)
	if err != nil {
		t.Fatalf("fixture.Document: %v", err)
	}
	const path = "testdata/preview.html"
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if got != string(want) {
		t.Errorf("rendered page differs from %s; run go test ./xui -run TestGolden -update and review the diff", path)
	}
}
