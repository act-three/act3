package ui

// A Badge displays a text label in a red capsule.
func Badge(label string) View {
	return Text(label).
		LineLimit(1).
		Font(SemiBold, SizeCapAbs(8i, 8i)).
		Padding(EdgesPillarbox(4i), EdgesLetterbox(6i)).
		FrameBounds(MinWidth(20i)). // equal to line height+padding
		Foreground(White).
		Background(Red).
		BorderShape(Capsule)
}
