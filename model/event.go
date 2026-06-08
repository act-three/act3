package model

import (
	"iter"

	"ily.dev/act3/model/progress"
)

var (
	EventLiveUpdate                = "live-update"
	EventSeasonRenumber            = "season-renumber"
	EventEpisodeSetSlug            = "episode-set-slug"
	EventSeriesSetSlug             = "series-set-slug"
	EventSeriesEditionSetSlug      = "series-edition-set-slug"
	EventMovieSetSlug              = "movie-set-slug"
	EventMovieEditionSetSlug       = "movie-edition-set-slug"
	EventMovieEditionChangePoster  = "movie-edition-change-poster"
	EventSeriesEditionChangePoster = "series-edition-change-poster"
	EventCollectionChangeBanner    = "collection-change-banner"
	EventCollectionMovieAdd        = "collection-movie-add"
	EventCollectionMovieRemove     = "collection-movie-remove"
	EventCollectionSeriesAdd       = "collection-series-add"
	EventCollectionSeriesRemove    = "collection-series-remove"
	EventCollectionSetSlug         = "collection-set-slug"
	EventEpisodeChangeThumbnail    = "episode-change-thumbnail"
	EventSeasonAdd                 = "season-add"
	EventSeasonEpisodeAdd          = "season-episode-add"
	EventSeasonEpisodeRemove       = "season-episode-remove"
	EventDownloadFileAttach        = "download-file-attach"
	EventUploadProgress            = "upload-progress"
	EventTaskStatsChange           = "task-stats-change"
	EventTrash                     = "trash"
	EventTrashCascade              = "trash-cascade"
	EventRestore                   = "restore"
	EventPurge                     = "purge"
)

type Event struct {
	Type     string
	Progress *progress.Item

	ID      string
	Addr    []string
	NewText string
	OldText string
	// ParentSlug, on edition slug events, is the slug addressing the
	// edition's series or movie. EditionSlugs, when the default
	// edition gains a slug, lists the slugs addressing the parent's
	// other editions.
	ParentSlug   string
	EditionSlugs []string
	TrashKind    TrashKind
	TrashItems   []TrashItem
}

func (m *Model) emitEvent(ev *Event) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for ch := range m.sub {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (m *Model) Events(ctx Context) iter.Seq[*Event] {
	ch := make(chan *Event, 64)
	m.subMu.Lock()
	m.sub[ch] = struct{}{}
	m.subMu.Unlock()
	return func(yield func(*Event) bool) {
		defer func() {
			m.subMu.Lock()
			delete(m.sub, ch)
			m.subMu.Unlock()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-ch:
				if !yield(ev) {
					return
				}
			}
		}
	}
}
