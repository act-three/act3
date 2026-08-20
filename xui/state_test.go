package ui

import "testing"

// TestModifyStateRequiresStateful pins the Modify contract: only the
// modifiers with exported constructors can be state-scoped, and an
// internal modifier reaching Modify with states is a bug.
func TestModifyStateRequiresStateful(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("state-scoping an internal modifier did not panic")
		}
	}()
	Text("x").Modify(modTag{name: "b"}, Hovered)
}
