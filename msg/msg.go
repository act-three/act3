package msg

import (
	"net/url"

	"ily.dev/act3/model"
)

//sumtype:decl
type Msg interface{ msg() }

// Dialog messages.
type (
	// DialogClose closes the open dialog, if any.
	DialogClose struct{}

	// SeriesAddOpen opens the add-series dialog.
	SeriesAddOpen struct{}

	// SeriesSearch searches TVmaze for shows matching Query.
	SeriesSearch struct{ Query string }

	// SeriesSearched delivers the results of a SeriesSearch.
	SeriesSearched struct {
		Query   string
		Results []model.SeriesSearchResult
	}

	// SeriesSearchError reports a failed SeriesSearch.
	SeriesSearchError struct {
		Query string
		Err   error
	}

	// SeriesAdd creates a local series from a TVmaze show.
	SeriesAdd struct{ TVmazeID int }

	// SeriesAdded reports that SeriesAdd created Local.
	SeriesAdded struct {
		TVmazeID int
		Local    *model.SeriesHead
	}

	// MovieAddOpen opens the add-movie dialog.
	MovieAddOpen struct{}

	// MovieSearch searches TMDB for movies matching Query.
	MovieSearch struct{ Query string }

	// MovieSearched delivers the results of a MovieSearch.
	MovieSearched struct {
		Query   string
		Results []model.MovieSearchResult
	}

	// MovieSearchError reports a failed MovieSearch.
	MovieSearchError struct {
		Query string
		Err   error
	}

	// MovieAdd creates a local movie from a TMDB entry.
	MovieAdd struct{ TMDBID int }

	// MovieAdded reports that MovieAdd created Local.
	MovieAdded struct {
		TMDBID int
		Local  *model.MovieHead
	}

	// CollectionMovieAddOpen opens the add-movie picker for the
	// collection.
	CollectionMovieAddOpen struct{ CollectionID string }

	// CollectionSeriesAddOpen opens the add-series picker for the
	// collection.
	CollectionSeriesAddOpen struct{ CollectionID string }

	// CollectionPickerSearch filters the open collection picker by
	// title.
	CollectionPickerSearch struct{ Query string }

	// ImageDialogOpen opens the image-edit dialog for the item ID
	// identifies: the poster of a movie or series edition, the
	// thumbnail of an episode, or the banner of a collection.
	ImageDialogOpen struct{ ID string }

	// DownloadFileAttachOpen opens the episode picker for attaching
	// the downloaded file to episodes.
	DownloadFileAttachOpen struct{ InfoHash, Path string }

	// DownloadFileAttachOpened delivers the set of episodes the file
	// was attached to when the picker opened.
	DownloadFileAttachOpened struct {
		InfoHash, Path string
		Attached       []string
	}

	// DownloadFileAttachPick attaches the downloaded file to the
	// episode and closes the picker.
	DownloadFileAttachPick struct{ InfoHash, Path, EpisodeID string }
)

// Player messages.
type (
	// Play opens the player for the given video IDs (episode or movie).
	Play struct {
		IDs             model.PlayIDs
		Audio, Subtitle string
		PinAudio        bool
	}

	// PlayerClose closes the player.
	PlayerClose struct{}
)

type (
	URLChange  struct{ URL *url.URL }
	URLRequest struct {
		URL      *url.URL
		Internal bool
	}

	// ModelEvent reports a change to shared model state.
	ModelEvent model.Event

	// Error reports a failed action, surfaced to the user as a note.
	Error struct{ Err error }
)

// Action messages, one per user-initiated mutation.
type (
	TaskRun    struct{ ID string }
	TaskKill   struct{ ID string }
	TaskDelete struct{ ID string }

	Trash   struct{ ID string }
	Restore struct{ ID string }
	Purge   struct{ ID string }

	// CollectionAdd creates a new, empty collection.
	CollectionAdd struct{}

	CollectionMovieAdd     struct{ CollectionID, MovieID string }
	CollectionSeriesAdd    struct{ CollectionID, SeriesID string }
	CollectionMovieRemove  struct{ CollectionID, MovieID string }
	CollectionSeriesRemove struct{ CollectionID, SeriesID string }

	// SeasonAdd appends a new season to an edition.
	SeasonAdd struct{ EditionID string }
	// SeriesEditionAdd creates a new series edition by duplicating
	// EditionID.
	SeriesEditionAdd struct{ EditionID string }
	// MovieEditionAdd creates a new movie edition by duplicating
	// EditionID.
	MovieEditionAdd struct{ EditionID string }
	// MovieEditionSetDefault promotes the edition to be the default
	// for its movie.
	MovieEditionSetDefault struct{ ID string }

	// EpisodeCreate creates a new episode in a season.
	EpisodeCreate    struct{ SeasonID string }
	SeasonAddEpisode struct {
		SeasonID, EpisodeID string
		SortKey             int64
	}
	SeasonRemoveEpisode struct{ SeasonID, EpisodeID string }

	// EpisodeMove moves an episode to position Index of season
	// SeasonID, possibly from another season of the same edition.
	EpisodeMove struct {
		EpisodeID, FromSeasonID, SeasonID string
		Index                             int
	}

	VideoReimport struct{ ID string }
	VideoReencode struct{ ID string }

	EpisodeVideoSetActive struct{ EpisodeID, VideoID string }
	MovieVideoSetActive   struct{ MovieEditionID, VideoID string }

	CollectionSetTitle struct{ ID, Title string }
	SeriesSetTitle     struct{ ID, Title string }
	SeasonSetTitle     struct{ ID, Title string }

	EpisodeSetTitle   struct{ ID, Title string }
	EpisodeSetAirdate struct{ ID, Airdate string }
	EpisodeSetSummary struct{ ID, Summary string }
	EpisodeSetType    struct{ ID, Type string }

	SeriesEditionSetLabel   struct{ ID, Label string }
	SeriesEditionSetSummary struct{ ID, Summary string }

	MovieEditionSetTitle       struct{ ID, Title string }
	MovieEditionSetLabel       struct{ ID, Label string }
	MovieEditionSetReleaseDate struct{ ID, ReleaseDate string }
	// MovieEditionSetRuntime carries the runtime as entered, in
	// minutes; Update validates it.
	MovieEditionSetRuntime struct{ ID, Runtime string }
	MovieEditionSetSummary struct{ ID, Summary string }

	// DownloadImport imports the download's files into the library.
	DownloadImport        struct{ ID string }
	DownloadSetAutoImport struct {
		ID string
		On bool
	}

	// TMDBSetToken sets the TMDB API read access token.
	TMDBSetToken struct{ Token string }
	// TransmissionSetURL sets the Transmission RPC URL.
	TransmissionSetURL struct{ URL string }
	// EpisodeVideoSet attaches (or detaches) a downloaded file
	// to an episode.
	EpisodeVideoSet struct {
		InfoHash, Path, EpisodeID string
		Attach                    bool
	}
)

func OnURLChange(u *url.URL) Msg { return &URLChange{u} }

func OnURLRequest(u *url.URL, internal bool) Msg { return &URLRequest{u, internal} }

func (*CollectionAdd) msg()              {}
func (*CollectionMovieAdd) msg()         {}
func (*CollectionMovieAddOpen) msg()     {}
func (*CollectionMovieRemove) msg()      {}
func (*CollectionPickerSearch) msg()     {}
func (*CollectionSeriesAdd) msg()        {}
func (*CollectionSeriesAddOpen) msg()    {}
func (*CollectionSeriesRemove) msg()     {}
func (*CollectionSetTitle) msg()         {}
func (*DialogClose) msg()                {}
func (*DownloadFileAttachOpen) msg()     {}
func (*DownloadFileAttachOpened) msg()   {}
func (*DownloadFileAttachPick) msg()     {}
func (*DownloadImport) msg()             {}
func (*DownloadSetAutoImport) msg()      {}
func (*EpisodeCreate) msg()              {}
func (*EpisodeMove) msg()                {}
func (*EpisodeSetAirdate) msg()          {}
func (*EpisodeSetSummary) msg()          {}
func (*EpisodeSetTitle) msg()            {}
func (*EpisodeSetType) msg()             {}
func (*EpisodeVideoSet) msg()            {}
func (*EpisodeVideoSetActive) msg()      {}
func (*Error) msg()                      {}
func (*ImageDialogOpen) msg()            {}
func (*ModelEvent) msg()                 {}
func (*MovieAdd) msg()                   {}
func (*MovieAddOpen) msg()               {}
func (*MovieAdded) msg()                 {}
func (*MovieEditionAdd) msg()            {}
func (*MovieEditionSetDefault) msg()     {}
func (*MovieEditionSetLabel) msg()       {}
func (*MovieEditionSetReleaseDate) msg() {}
func (*MovieEditionSetRuntime) msg()     {}
func (*MovieEditionSetSummary) msg()     {}
func (*MovieEditionSetTitle) msg()       {}
func (*MovieSearch) msg()                {}
func (*MovieSearchError) msg()           {}
func (*MovieSearched) msg()              {}
func (*MovieVideoSetActive) msg()        {}
func (*Play) msg()                       {}
func (*PlayerClose) msg()                {}
func (*Purge) msg()                      {}
func (*Restore) msg()                    {}
func (*SeasonAdd) msg()                  {}
func (*SeasonAddEpisode) msg()           {}
func (*SeasonRemoveEpisode) msg()        {}
func (*SeasonSetTitle) msg()             {}
func (*SeriesAdd) msg()                  {}
func (*SeriesAddOpen) msg()              {}
func (*SeriesAdded) msg()                {}
func (*SeriesEditionAdd) msg()           {}
func (*SeriesEditionSetLabel) msg()      {}
func (*SeriesEditionSetSummary) msg()    {}
func (*SeriesSearch) msg()               {}
func (*SeriesSearchError) msg()          {}
func (*SeriesSearched) msg()             {}
func (*SeriesSetTitle) msg()             {}
func (*TMDBSetToken) msg()               {}
func (*TaskDelete) msg()                 {}
func (*TaskKill) msg()                   {}
func (*TaskRun) msg()                    {}
func (*TransmissionSetURL) msg()         {}
func (*Trash) msg()                      {}
func (*URLChange) msg()                  {}
func (*URLRequest) msg()                 {}
func (*VideoReencode) msg()              {}
func (*VideoReimport) msg()              {}
