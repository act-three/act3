package model

import (
	"database/sql"
	"iter"
	"slices"
	"strconv"
	"strings"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
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

func (sr *SeriesHead) PlayURL() string {
	return "/" + sr.sr.Slug
}

func (sr *SeriesHead) EditURL() string {
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

type Series struct {
	SeriesHead
	editions []*SeriesEdition
	epByID   map[string]*Episode
}

func newSeries(
	srData schema.Series,
	seds []schema.SeriesEdition,
	sns []schema.Season,
	sneps []schema.SeasonEpisode,
	eps []schema.Episode,
	progByEpisodeID func(string) []*progress.Item,
	videosByEpisodeID map[string][]*Video,
) *Series {
	sr := &Series{
		SeriesHead: SeriesHead{srData},
		epByID:     map[string]*Episode{},
	}

	epByID := epMapByID(eps)
	snepBySeasonID := snepMapBySeasonID(sneps)

	snByEditionID := map[string][]schema.Season{}
	for i := range sns {
		soID := sns[i].EditionID
		snByEditionID[soID] = append(snByEditionID[soID], sns[i])
	}

	for _, soData := range seds {
		sns := snByEditionID[soData.ID]
		sed := newSeriesEdition(&sr.SeriesHead, soData, sns, snepBySeasonID, epByID, progByEpisodeID, videosByEpisodeID)
		sr.editions = append(sr.editions, sed)
	}
	return sr
}

func (sr *Series) EditionByTitle(title string) *SeriesEdition {
	if sr == nil {
		return nil
	}
	for _, sed := range sr.editions {
		if sed.Title() == title {
			return sed
		}
	}
	return nil
}

func (sr *Series) DefaultEdition() *SeriesEdition {
	if sr == nil {
		return nil
	}
	for _, sed := range sr.editions {
		if sed.sed.Slug == "" {
			return sed
		}
	}
	return nil
}

func (sr *Series) EditionBySlug(slug string) *SeriesEdition {
	if sr == nil {
		return nil
	}
	for _, sed := range sr.editions {
		if sed.sed.Slug == slug {
			return sed
		}
	}
	return nil
}

func (sr *Series) SeriesEditionSeq() iter.Seq[*SeriesEdition] {
	return slices.Values(sr.editions)
}

func (tx *TxR) SeriesHeadList(ctx Context) ([]*SeriesHead, error) {
	a, err := tx.q.SeriesList(ctx)
	if err != nil {
		return nil, err
	}
	return newSeriesHeadList(a), nil
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

func (tx *TxRW) generateSeriesSlug(ctx Context, title string, premiered *string, id string) (string, error) {
	slug := xstrings.ToSlug(title)
	if slug == "" {
		slug = id
	}
	if !isReservedSlug(slug) {
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
	sedData, err := tx.SeriesEditionCreate(ctx,
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

// SeriesEditionBySlug looks up a series by its slug
// and returns the edition matching edSlug
// (empty string for the default edition).
func (tx *TxR) SeriesEditionBySlug(ctx Context, slug, edSlug string) (*SeriesEdition, error) {
	// TODO(april): avoid loading other editions here.
	srData, err := tx.q.SeriesGetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	id := srData.ID
	seds, err := tx.q.SeriesEditionListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	sns, err := tx.q.SeasonListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	eps, err := tx.q.EpisodeListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	evs, err := tx.q.EpisodeVideoListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListBySeriesID(ctx, id)
	if err != nil {
		return nil, err
	}

	vidByID := vidMapByID(vids)
	videosByEpisodeID := vidMapByEpisodeID(evs, vidByID)

	sr := newSeries(srData, seds, sns, sneps, eps, tx.m.prog.List, videosByEpisodeID)
	sed := sr.EditionBySlug(edSlug)
	if sed == nil {
		return nil, sql.ErrNoRows
	}
	return sed, nil
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
