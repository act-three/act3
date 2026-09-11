package ui

// A Badge displays a text label in a red capsule.
func Badge(label string) View {
	// TODO: keep the label on one line (white-space:nowrap)
	// once we have some sort of nowrap modifier.
	return Text(label).
		Font(Bold, SizeEm(12i, 1.3)).
		Padding(EdgesLetterbox(2), EdgesPillarbox(8)).
		Foreground(White).
		Background(Red).
		BorderShape(Capsule)
}
