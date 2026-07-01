// Package kind defines the object-type enumerations
// the app uses in its forms.
// Each enum is an interface whose members are shared object types.
// Each member's String is the object's database table name.
package kind

type (
	Collection    struct{}
	Download      struct{}
	Episode       struct{}
	Movie         struct{}
	MovieEdition  struct{}
	Season        struct{}
	Series        struct{}
	SeriesEdition struct{}
	Video         struct{}
)

func (Collection) String() string    { return "Collection" }
func (Download) String() string      { return "Download" }
func (Episode) String() string       { return "Episode" }
func (Movie) String() string         { return "Movie" }
func (MovieEdition) String() string  { return "MovieEdition" }
func (Season) String() string        { return "Season" }
func (Series) String() string        { return "Series" }
func (SeriesEdition) String() string { return "SeriesEdition" }
func (Video) String() string         { return "Video" }
