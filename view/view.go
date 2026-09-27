// Package view composes pages and presentation components.
// Page functions may read data through a model.TxR supplied by web.
// Reusable presentation components take model objects or values and perform no I/O.
// Rendering must not write data or perform external service calls.
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
