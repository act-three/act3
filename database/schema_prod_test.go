//go:build prod

package database

import "testing"

// checkProdUpdates is the prod-only backstop: it must accept a fully
// frozen schema and reject any unfrozen update.
func TestCheckProdUpdates(t *testing.T) {
	t.Run("frozen", func(t *testing.T) {
		checkProdUpdates(0) // must not panic
	})
	t.Run("unfrozen", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for an unfrozen update")
			}
		}()
		checkProdUpdates(1)
	})
}
