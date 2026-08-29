package ui_test

import _ "embed"

// staticCSS is the static stylesheet.
// The tests embed it themselves since package ui does not export it.
//
//go:embed ui.css
var staticCSS string
