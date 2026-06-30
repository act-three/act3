package web

import (
	"ily.dev/act3/model"
)

// A dialog holds the app-side state of the open modal dialog.
// The view renders the open dialog every cycle; closing it is
// removing it from the app state (see msg.DialogClose). Any
// navigation also closes it.
type dialog interface{ isDialog() }

// seriesAddDialog is the state of the add-series dialog: a TVmaze
// search box plus its results.
type seriesAddDialog struct {
	query     string
	searching bool
	results   []model.SeriesSearchResult
}

func (*seriesAddDialog) isDialog() {}

// movieAddDialog is the state of the add-movie dialog: a TMDB
// search box plus its results.
type movieAddDialog struct {
	query     string
	searching bool
	results   []model.MovieSearchResult
}

func (*movieAddDialog) isDialog() {}

// The collection pickers search the library, so their dialogs hold
// no results: the view runs the search on every render, which also
// keeps the already-in-collection marks fresh after an add.

// collectionMovieAddDialog is the state of the add-movie-to-
// collection picker.
type collectionMovieAddDialog struct {
	colID string
	query string
}

func (*collectionMovieAddDialog) isDialog() {}

// collectionSeriesAddDialog is the state of the add-series-to-
// collection picker.
type collectionSeriesAddDialog struct {
	colID string
	query string
}

func (*collectionSeriesAddDialog) isDialog() {}

// imageDialog is the state of an image-edit dialog: the poster,
// banner, or thumbnail of the identified item, with an upload
// control. The view resolves the item on each render, so an upload
// refreshes the image in place.
type imageDialog struct {
	kind string // upload form-field name
	id   string
}

func (*imageDialog) isDialog() {}

// downloadFileAttachPopover is the state of the episode picker for
// attaching a downloaded file. Like the dialogs, it occupies the
// one-overlay-at-a-time slot; toggling an episode keeps it open for
// further picks.
type downloadFileAttachPopover struct {
	infoHash string
	path     string
	// attached pins the episodes the file was attached to when the
	// picker opened, so its attached group keeps a stable shape as
	// episodes are toggled — no layout shifts, and an accidental
	// detach stays in view to revert. Until the snapshot arrives
	// (ready), the picker renders nothing.
	attached []string
	ready    bool
}

func (*downloadFileAttachPopover) isDialog() {}
