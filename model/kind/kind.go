// Package kind defines the object-type enumerations
// the app uses in its forms.
// Each enum is an interface whose members are shared object types.
// Each member's String is the object's database table name.
package kind

type (
	MovieEdition  struct{}
	SeriesEdition struct{}
)

func (MovieEdition) String() string  { return "MovieEdition" }
func (SeriesEdition) String() string { return "SeriesEdition" }
