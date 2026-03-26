package model

import (
	"path"
	"slices"
	"strconv"
	"strings"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/xstrings"
)

type SeriesHead struct {
	sr schema.Series
}

func newSeriesHeadList(list []schema.Series) []*SeriesHead {
	sss := make([]*SeriesHead, len(list))
	for i := range sss {
		sss[i] = &SeriesHead{list[i]}
	}
	return sss
}

func (sr *SeriesHead) ID() string           { return sr.sr.ID }
func (sr *SeriesHead) Slug() string         { return sr.sr.Slug }
func (sr *SeriesHead) PremieredOn() *string { return sr.sr.PremieredOn }
func (sr *SeriesHead) Status() string       { return sr.sr.Status }
func (sr *SeriesHead) Title() string        { return sr.sr.Title }
func (sr *SeriesHead) TVmazeID() *int64     { return sr.sr.TVmazeID }

func (sr *SeriesHead) TheaterPath() string {
	return "/" + sr.sr.Slug
}

func (sr *SeriesHead) EditorPath() string {
	return "/app/series/" + sr.sr.Slug
}

// SeriesWork contains information needed to display
// a concise representation of a series edition in UI.
// It contains metadata about the series itself plus an edition.
type SeriesWork struct {
	SeriesHead
	SeriesEditionHead
}

func (sw *SeriesWork) Title() string    { return sw.SeriesHead.Title() }
func (sw *SeriesWork) ImageURL() string { return sw.TVmazeImageURL() }

func (sw *SeriesWork) TheaterPath() string {
	return path.Join(sw.SeriesHead.TheaterPath(), sw.SeriesEditionHead.Slug())
}

func (sw *SeriesWork) EditorPath() string {
	return path.Join(sw.SeriesHead.EditorPath(), sw.SeriesEditionHead.Slug())
}

func (tx *TxR) SeriesHead(ctx Context, id string) (*SeriesHead, error) {
	srData, err := tx.q.SeriesGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &SeriesHead{srData}, nil
}

func (tx *TxR) SeriesHeadListByTVmazeID(ctx Context, id []*int64) ([]*SeriesHead, error) {
	a, err := tx.q.SeriesListByTVmazeID(ctx, id)
	if err != nil {
		return nil, err
	}
	return newSeriesHeadList(a), nil
}

var reservedSlugs = map[string]bool{
	"app": true,
	"-":   true,
}

func isReservedSlug(s string) bool {
	return reservedSlugs[s]
}

func (tx *TxRW) generateSeriesSlug(ctx Context, title string, premiered *string, id string, allow ...string) (string, error) {
	slug := xstrings.ToSlug(title)
	if slug == "" {
		slug = id
	}
	if !isReservedSlug(slug) {
		if slices.Contains(allow, slug) {
			return slug, nil
		}
		n, err := tx.q.SeriesSlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return slug, nil
		}
	}

	// Try slug-year.
	if premiered != nil {
		year, _, _ := strings.Cut(*premiered, "-")
		if year != "" {
			candidate := slug + "-" + year
			if slices.Contains(allow, candidate) {
				return candidate, nil
			}
			n, err := tx.q.SeriesSlugExists(ctx, candidate)
			if err != nil {
				return "", err
			}
			if n == 0 {
				return candidate, nil
			}
		}
	}

	// Last resort: slug-id.
	return slug + "-" + id, nil
}

func (tx *TxRW) SeriesTitleSet(ctx Context, id, title string) error {
	err := tx.q.SeriesTitleSet(ctx, schema.SeriesTitleSetParams{
		Title: title,
		ID:    id,
	})
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventSeriesSetTitle,
			ID:      id,
			NewText: title,
		})
	})
	sr, err := tx.q.SeriesGet(ctx, id)
	if err != nil {
		return err
	}
	slug, err := tx.generateSeriesSlug(ctx, title, sr.PremieredOn, id, sr.Slug)
	if err != nil {
		return err
	}
	if slug == sr.Slug {
		return nil
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventSeriesSetSlug,
			ID:      id,
			NewText: slug,
			OldText: sr.Slug,
		})
	})
	return tx.q.SeriesSlugSet(ctx, schema.SeriesSlugSetParams{
		Slug: slug,
		ID:   id,
	})
}

func (tx *TxRW) SeriesCreateByTVmazeID(ctx Context, show *tvmaze.Show) (*SeriesWork, error) {
	id64 := int64(show.ID)

	srID := "sr" + flurry.NewID()
	slug, err := tx.generateSeriesSlug(ctx,
		show.Name, show.Premiered, srID)
	if err != nil {
		return nil, err
	}

	srData, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
		ID:          srID,
		Slug:        slug,
		Title:       show.Name,
		Status:      show.Status,
		Language:    show.Language,
		PremieredOn: show.Premiered,
		EndedOn:     show.Ended,

		TVmazeID:        &id64,
		TVmazeURL:       &show.URL,
		TVmazeUpdatedAt: int64(show.Updated),
		IMDBID:          show.Externals.IMDB,
		TVDBID:          show.Externals.TheTVDB,
		TVRageID:        show.Externals.TVRage,
	})
	if err != nil {
		return nil, err
	}
	sedData, err := tx.seriesEditionCreate(ctx,
		AirDate, srID, show.Summary, show.Image.Medium())
	if err != nil {
		return nil, err
	}
	err = tx.addTask(ctx, taskFetchEpisodes,
		strconv.FormatInt(id64, 10), sedData.ID)
	if err != nil {
		return nil, err
	}
	for _, g := range show.Genres {
		err := tx.q.SeriesGenreAdd(ctx, schema.SeriesGenreAddParams{
			SeriesID:  srID,
			GenreName: g,
		})
		if err != nil {
			return nil, err
		}
	}
	return &SeriesWork{
		SeriesHead:        SeriesHead{srData},
		SeriesEditionHead: SeriesEditionHead{sedData},
	}, nil
}

// SeriesWorkList returns the default edition of each series.
func (tx *TxR) SeriesWorkList(ctx Context) ([]*SeriesWork, error) {
	editions, err := tx.q.SeriesEditionListDefault(ctx)
	if err != nil {
		return nil, err
	}
	series, err := tx.q.SeriesList(ctx)
	if err != nil {
		return nil, err
	}
	edBySeriesID := make(map[string]schema.SeriesEdition, len(editions))
	for _, ed := range editions {
		edBySeriesID[ed.SeriesID] = ed
	}
	var works []*SeriesWork
	for _, sr := range series {
		ed, ok := edBySeriesID[sr.ID]
		if !ok {
			continue // no default edition; skip
		}
		works = append(works, &SeriesWork{
			SeriesHead:        SeriesHead{sr},
			SeriesEditionHead: SeriesEditionHead{ed},
		})
	}
	return works, nil
}
