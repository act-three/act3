package ui

// A Card displays its subview surrounded by a border.
func Card(subview View) View {
	return HStack(subview).
		modify(modTagDefault("ui-card")).
		Background(surfaceColor).
		BorderStroke(1, borderColor).
		BorderShape(RoundedRectangle)
}
