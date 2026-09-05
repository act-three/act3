package ui

// A Badge displays a text label in a red capsule.
func Badge(label string) View {
	// TODO: keep the label on one line (white-space:nowrap)
	// once we have some sort of nowrap modifier.
	return Text(label).
		Bold().
		Font(Caption).
		Padding(EdgesLetterbox(2), EdgesPillarbox(8)).
		Foreground(textOn(Red.color())).
		Background(Red).
		BorderShape(Capsule)
}
