package ui

// A Card displays its subview surrounded by a border.
func Card(subview View) View {
	// TODO: don't use custom CSS once we have a border-width modifier.
	return HStack(subview).Class("ui-card")
}
