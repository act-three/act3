package hi_test

import _ "embed"

// staticCSS is the static stylesheet.
// The tests embed it themselves since package hi does not export it.
//
//go:embed hi.css
var staticCSS string
