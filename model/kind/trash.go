package kind

import "fmt"

// A Trash is the kind of object a trash entry refers to.
//
//sumtype:decl
type Trash interface {
	fmt.Stringer
	trash()
}

func (Collection) trash()    {}
func (Download) trash()      {}
func (Episode) trash()       {}
func (Movie) trash()         {}
func (MovieEdition) trash()  {}
func (Season) trash()        {}
func (Series) trash()        {}
func (SeriesEdition) trash() {}
func (Video) trash()         {}

var trashes = []Trash{
	Collection{},
	Download{},
	Episode{},
	Movie{},
	MovieEdition{},
	Season{},
	Series{},
	SeriesEdition{},
	Video{},
}

// ParseTrash returns the Trash named by s.
func ParseTrash(s string) (Trash, error) {
	for _, k := range trashes {
		if k.String() == s {
			return k, nil
		}
	}
	return nil, fmt.Errorf("kind: bad Trash %q", s)
}
