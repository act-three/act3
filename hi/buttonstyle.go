package hi

// ButtonStyle specifies the appearance of buttons in a view.
type ButtonStyle int

const (
	// Bordered specifies a bordered button style suitable
	// for most buttons.
	Bordered ButtonStyle = iota

	// Prominent specifies a prominent bordered button style.
	// It is suitable for a primary or important action.
	// It should be used sparingly.
	Prominent

	// Subtle specifies a bordered button style with
	// reduced prominence.
	Subtle

	// Borderless specifies a borderless button style.
	Borderless

	// Destructive specifies a prominent bordered button style
	// colored red to indicate danger.
	Destructive

	// DestructiveSubtle specifies a bordered button style with
	// reduced prominence and colored red to indicate danger.
	DestructiveSubtle
)
