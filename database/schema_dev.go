//go:build !prod

package database

// checkProdUpdates is a no-op outside production builds, where an
// in-development unfrozen update is expected.
func checkProdUpdates(int) {}
