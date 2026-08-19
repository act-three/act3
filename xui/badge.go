package ui

// A Badge is a solid color capsule shape showing text.
func Badge(label string) View {
	// TODO: keep the label on one line (white-space:nowrap)
	// once we have some sort of nowrap modifier.
	return Text(label).
		Bold().
		Font(Caption).
		Foreground(CSSColor("#fff")).
		Padding(EdgesLetterbox(2), EdgesPillarbox(8)).
		Background(Accent).
		BorderShape(Capsule)
}
