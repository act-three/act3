package kind

import "fmt"

// A SlugOwner is the kind of object a top-level slug names.
//
//sumtype:decl
type SlugOwner interface {
	fmt.Stringer
	slugOwner()
}

func (Collection) slugOwner() {}
func (Movie) slugOwner()      {}
func (Series) slugOwner()     {}

var slugOwners = []SlugOwner{Collection{}, Movie{}, Series{}}

// ParseSlugOwner returns the SlugOwner named by s.
func ParseSlugOwner(s string) (SlugOwner, error) {
	for _, k := range slugOwners {
		if k.String() == s {
			return k, nil
		}
	}
	return nil, fmt.Errorf("kind: bad SlugOwner %q", s)
}
