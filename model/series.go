package model

import (
	"path"
	"slices"
	"strconv"
	"strings"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
	"ily.dev/act3/priority"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/xstrings"
	"kr.dev/errorfmt"
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

func (sr *SeriesHead) ID() string          { return sr.sr.ID }
func (sr *SeriesHead) Slug() string        { return sr.sr.Slug }
func (sr *SeriesHead) PremieredOn() string { return sr.sr.PremieredOn }
func (sr *SeriesHead) Status() string      { return sr.sr.Status }
func (sr *SeriesHead) Title() string       { return sr.sr.Title }
func (sr *SeriesHead) TVmazeID() *int64    { return sr.sr.TVmazeID }

func (sr *SeriesHead) addr(field string) []string {
	return []string{"series", sr.ID(), field}
}

func (sr *SeriesHead) TitleAddr() []string { return sr.addr("title") }
func (sr *SeriesHead) SlugAddr() []string  { return sr.addr("slug") }

func (sr *SeriesHead) TitleField() (string, []string) { return sr.Title(), sr.TitleAddr() }
func (sr *SeriesHead) SlugField() (string, []string)  { return sr.Slug(), sr.SlugAddr() }

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

func (sw *SeriesWork) Kind() string  { return "series" }
func (sw *SeriesWork) Title() string { return sw.SeriesHead.Title() }

func (sw *SeriesWork) TheaterPath() string {
	return path.Join(sw.SeriesHead.TheaterPath(), sw.SeriesEditionHead.Slug())
}

func (sw *SeriesWork) EditorPath() string {
	return SeriesEditionEditorPath(&sw.SeriesHead, &sw.SeriesEditionHead)
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

// A SeriesSearchResult pairs a TVmaze show with the local series
// already created from it, if any.
type SeriesSearchResult struct {
	Show  tvmaze.Show
	Local *SeriesHead
}

// SearchSeries searches TVmaze for shows matching query and pairs
// each result with the local series created from it, if any.
func (m *Model) SearchSeries(ctx Context, query string) (results []SeriesSearchResult, err error) {
	defer errorfmt.Handlef("search series: %w", &err)
	found, err := m.tvmaze.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	ids := make([]*int64, len(found))
	for i, r := range found {
		id := int64(r.Show.ID)
		ids[i] = &id
	}
	err = m.WithTxR(func(tx *TxR) error {
		heads, err := tx.SeriesHeadListByTVmazeID(ctx, ids)
		if err != nil {
			return err
		}
		local := make(map[int64]*SeriesHead, len(heads))
		for _, sr := range heads {
			local[*sr.TVmazeID()] = sr
		}
		for _, r := range found {
			results = append(results, SeriesSearchResult{
				Show:  r.Show,
				Local: local[int64(r.Show.ID)],
			})
		}
		return nil
	})
	return results, err
}

// AddSeriesByTVmazeID fetches show id from TVmaze
// and creates a local series from it.
func (m *Model) AddSeriesByTVmazeID(ctx Context, id int) (sw *SeriesWork, err error) {
	defer errorfmt.Handlef("add series: %w", &err)
	show, err := m.tvmaze.GetShow(ctx, id)
	if err != nil {
		return nil, err
	}
	err = m.WithTxRW(func(tx *TxRW) error {
		var err error
		sw, err = tx.SeriesCreateByTVmazeID(ctx, show)
		return err
	})
	return sw, err
}

var reservedSlugs = map[string]bool{
	"app":         true,
	"collections": true,
	"-":           true,
}

func isReservedSlug(s string) bool {
	return reservedSlugs[s]
}

// seriesEnsureSlug aligns the Series' slug with its current title,
// announcing the change on a live rename so a viewer can follow it,
// and keeps the Slug table row in sync. Safe to call on live series
// (title-change) or trashed ones (restore).
func (tx *TxRW) seriesEnsureSlug(ctx Context, id string) error {
	sr, err := tx.q.SeriesGet(ctx, id)
	if err != nil {
		return err
	}
	var allow []string
	if sr.DeletedAt == nil {
		allow = []string{sr.Slug}
	}
	slug, err := tx.generateSeriesSlug(ctx, sr.Title, sr.PremieredOn, id, allow...)
	if err != nil {
		return err
	}
	if slug != sr.Slug {
		// Only a live rename is announced: a trashed series' pages
		// have no viewers to follow it, and its former slug may
		// meanwhile address something else entirely.
		if sr.DeletedAt == nil {
			tx.emitDetail(Detail{SlugChangeID: id})
		}
		if err := tx.q.SeriesSlugSet(ctx, schema.SeriesSlugSetParams{Slug: slug, ID: id}); err != nil {
			return err
		}
	}
	if sr.DeletedAt != nil || slug != sr.Slug {
		return tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: slug, Kind: "series", Target: id,
		})
	}
	return nil
}

func (tx *TxRW) generateSeriesSlug(ctx Context, title, premiered, id string, allow ...string) (string, error) {
	slug := xstrings.ToSlug(title)
	if slug == "" {
		slug = id
	}
	if !isReservedSlug(slug) {
		if slices.Contains(allow, slug) {
			return slug, nil
		}
		n, err := tx.q.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return slug, nil
		}
	}

	// Try slug-year.
	if premiered != "" {
		year, _, _ := strings.Cut(premiered, "-")
		if year != "" {
			candidate := slug + "-" + year
			if slices.Contains(allow, candidate) {
				return candidate, nil
			}
			n, err := tx.q.SlugExists(ctx, candidate)
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
	if _, err := tx.q.SeriesGet(ctx, id); err != nil {
		return err
	}
	err := tx.q.SeriesTitleSet(ctx, schema.SeriesTitleSetParams{
		Title: title,
		ID:    id,
	})
	if err != nil {
		return err
	}
	return tx.seriesEnsureSlug(ctx, id)
}

func (tx *TxRW) SeriesCreateByTVmazeID(ctx Context, show *tvmaze.Show) (*SeriesWork, error) {
	id64 := int64(show.ID)

	srID := "sr" + flurry.NewID()
	var premiered, ended string
	if show.Premiered != nil {
		premiered = *show.Premiered
	}
	if show.Ended != nil {
		ended = *show.Ended
	}
	slug, err := tx.generateSeriesSlug(ctx,
		show.Name, premiered, srID)
	if err != nil {
		return nil, err
	}

	srData, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
		ID:          srID,
		Slug:        slug,
		Title:       show.Name,
		Status:      show.Status,
		PremieredOn: premiered,
		EndedOn:     ended,

		TVmazeID: &id64,
		IMDBID:   show.Externals.IMDB,
		TVDBID:   show.Externals.TheTVDB,
		TVRageID: show.Externals.TVRage,
	})
	if err != nil {
		return nil, err
	}
	err = tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
		Slug:   slug,
		Kind:   "series",
		Target: srID,
	})
	if err != nil {
		return nil, err
	}
	sedData, err := tx.seriesEditionCreate(ctx, "Air Date", srID, show.Summary)
	if err != nil {
		return nil, err
	}
	if show.Image != nil {
		err = tx.addTaskWithPriority(ctx, priority.FetchPoster, taskFetchSeriesPoster, sedData.ID, show.Image.OriginalURL)
		if err != nil {
			return nil, err
		}
	}
	err = tx.addTask(ctx, taskFetchEpisodes,
		strconv.FormatInt(id64, 10), sedData.ID)
	if err != nil {
		return nil, err
	}
	return &SeriesWork{
		SeriesHead:        SeriesHead{srData},
		SeriesEditionHead: SeriesEditionHead{sed: sedData},
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
			SeriesEditionHead: SeriesEditionHead{sed: ed},
		})
	}
	return works, nil
}
