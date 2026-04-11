package model

import (
	"iter"

	"ily.dev/act3/model/progress"
)

var (
	EventLiveUpdate                = "live-update"
	EventSeasonRenumber            = "season-renumber"
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
	EventDownloadFileAttach        = "download-file-attach"
	EventTaskStatsChange           = "task-stats-change"
)

type Event struct {
	Type     string
	Progress *progress.Item

	ID      string
	Addr    []string
	NewText string
	OldText string
}

func (m *Model) addEvent(ev *Event) {
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
