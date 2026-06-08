// Package view (along with its subdirectories) holds pure functions
// for generating HTML trees from model objects.
// Code in these packages must not perform I/O (e.g. database or network).
package view

import (
	"ily.dev/domi"
)

var group = domi.Group

func isUserAdmin() bool {
	// TODO(april): make this work properly once we have user accounts,
	// maybe via context.
	return false
}
