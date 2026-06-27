//go:build prod

package database

import "fmt"

// checkProdUpdates panics when the embedded schema still carries an
// unfrozen update. A production build must ship a fully frozen schema;
// an unfrozen tip means an update reached prod without being recorded
// in frozen.txt, which the deploy gate is meant to prevent. This is the
// last-resort backstop for when it isn't.
func checkProdUpdates(unfrozen int) {
	if unfrozen != 0 {
		panic(fmt.Errorf("ddl: %d unfrozen update(s) in a prod build; freeze before deploying", unfrozen))
	}
}
