package ui

import _ "embed"

// CSS is the static stylesheet
// backing the prototype's components and layout primitives.
// Serve it once per page.
//
//go:embed ui.css
var CSS string
