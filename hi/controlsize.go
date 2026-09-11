package hi

// A ControlSize is a size class for interactive controls.
type ControlSize int

const (
	Mini ControlSize = iota - 2
	Small
	Regular
	Large
)
