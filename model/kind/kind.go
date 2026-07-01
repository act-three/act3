// Package kind defines the object-type enumerations
// the app uses in its forms.
// Each enum is an interface whose members are shared object types.
// Each member's String is the object's database table name.
package kind

type (
	Collection    struct{}
	Episode       struct{}
	MovieEdition  struct{}
	SeriesEdition struct{}
)

func (Collection) String() string    { return "Collection" }
func (Episode) String() string       { return "Episode" }
func (MovieEdition) String() string  { return "MovieEdition" }
func (SeriesEdition) String() string { return "SeriesEdition" }
