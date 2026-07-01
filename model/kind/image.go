package kind

import "fmt"

// An ImageOwner is the kind of object an uploaded image is attached to.
//
//sumtype:decl
type ImageOwner interface {
	fmt.Stringer
	imageOwner()
}

func (Collection) imageOwner()    {}
func (Episode) imageOwner()       {}
func (MovieEdition) imageOwner()  {}
func (SeriesEdition) imageOwner() {}

var imageOwners = []ImageOwner{
	Collection{},
	Episode{},
	MovieEdition{},
	SeriesEdition{},
}

// ParseImageOwner returns the ImageOwner named by s.
func ParseImageOwner(s string) (ImageOwner, error) {
	for _, k := range imageOwners {
		if k.String() == s {
			return k, nil
		}
	}
	return nil, fmt.Errorf("kind: bad ImageOwner %q", s)
}
