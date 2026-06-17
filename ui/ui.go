// Package ui provides reusable UI components built on the html package.
//
// # Inline-Updating Controls
//
// Components that POST to a server endpoint (Toggle, and future inline
// controls) expect handlers to conform to a simple contract:
//
//   - Accept a POST with form-encoded body.
//   - On success, respond with 204 No Content (no body).
//   - On failure, respond with a 4xx or 5xx status.
//
// The component's Stimulus controller interprets the status code and
// handles UI state accordingly (commit optimistic update on success,
// revert on failure).
// No HTML is returned — the controller owns the DOM.
package ui

import "ily.dev/domi"

func Group(nodes ...domi.Node) domi.Node {
	return domi.Fragment(nodes...)
}
