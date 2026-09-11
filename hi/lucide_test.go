package hi

import (
	"io/fs"
	"strings"
	"testing"
)

func TestBundledIcons(t *testing.T) {
	entries, err := fs.ReadDir(lucideIcons, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".svg")
		if defaultIconSource(name) == nil {
			t.Errorf("bundled icon %q failed to parse", name)
		}
	}
	for _, name := range []string{"", "no-such-icon", "../placeholder", "/film", "film.svg"} {
		if defaultIconSource(name) != nil {
			t.Errorf("invalid icon name %q returned a node", name)
		}
	}
}
