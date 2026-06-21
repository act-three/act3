package model

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/xstrings"
)

// CollectionHead is the lightweight representation used in lists.
type CollectionHead struct {
	col schema.Collection
}

func (c *CollectionHead) ID() string    { return c.col.ID }
func (c *CollectionHead) Slug() string  { return c.col.Slug }
func (c *CollectionHead) Title() string { return c.col.Title }
func (c *CollectionHead) Banner() Image {
	return Image{ID: c.col.BannerID, Kind: ImageBanner}
}

func (c *CollectionHead) addr(field string) []string {
	return []string{"collection", c.ID(), field}
}

func (c *CollectionHead) TitleAddr() []string  { return c.addr("title") }
func (c *CollectionHead) SlugAddr() []string   { return c.addr("slug") }
func (c *CollectionHead) BannerAddr() []string { return c.addr("banner") }

func (c *CollectionHead) BannerField() (Image, []string) {
	return c.Banner(), c.BannerAddr()
}

func (c *CollectionHead) TitleField() (string, []string) { return c.Title(), c.TitleAddr() }
func (c *CollectionHead) SlugField() (string, []string)  { return c.Slug(), c.SlugAddr() }

func (c *CollectionHead) TheaterPath() string {
	return path.Join("/", c.col.Slug)
}

func (c *CollectionHead) EditorPath() string {
	return "/app/collections/" + c.col.Slug
}

func (c *CollectionHead) PlaylistPath() string {
	return c.TheaterPath() + "/playlist"
}

// Collection is the full representation with associated movies and series.
type Collection struct {
	CollectionHead
	movies []*MovieWork
	series []*SeriesWork
}

func (c *Collection) Movies() []*MovieWork  { return c.movies }
func (c *Collection) Series() []*SeriesWork { return c.series }

// Works returns all movies and series in c in release order.
func (c *Collection) Works() []Work {
	works := make([]Work, 0, len(c.movies)+len(c.series))
	for _, mw := range c.movies {
		works = append(works, mw)
	}
	for _, sw := range c.series {
		works = append(works, sw)
	}
	slices.SortFunc(works, func(a, b Work) int {
		return cmp.Compare(releaseDate(a), releaseDate(b))
	})
	return works
}

func releaseDate(w Work) string {
	switch v := w.(type) {
	case *MovieWork:
		return v.Year()
	case *SeriesWork:
		return v.PremieredOn()
	}
	return ""
}

func (c *Collection) MovieCountAddr() []string { return c.addr("movie-count") }
func (c *Collection) MovieCountField() (string, []string) {
	return fmt.Sprintf("%d Movies", len(c.movies)), c.MovieCountAddr()
}

func (c *Collection) SeriesCountAddr() []string { return c.addr("series-count") }
func (c *Collection) SeriesCountField() (string, []string) {
	return fmt.Sprintf("%d Series", len(c.series)), c.SeriesCountAddr()
}

func (tx *TxR) CollectionHead(id string) *CollectionHead {
	colData := txmust1(tx.q.CollectionGet(id))
	return &CollectionHead{col: colData}
}

func (tx *TxR) CollectionHeadList() []*CollectionHead {
	a := txmust1(tx.q.CollectionList())
	cols := make([]*CollectionHead, len(a))
	for i := range a {
		cols[i] = &CollectionHead{col: a[i]}
	}
	return cols
}

// Collection returns the collection for id.
// If id is not found, Collection aborts the tx.
func (tx *TxR) Collection(id string) *Collection {
	colData := txmust1(tx.q.CollectionGet(id))
	return tx.collectionFromData(colData)
}

// CollectionMovieSearchResult pairs a library movie matching a
// picker search with whether it is already in the collection.
type CollectionMovieSearchResult struct {
	Movie        *MovieWork
	InCollection bool
}

// CollectionMovieSearch lists library movies whose titles contain
// query, case-insensitively, marking those already in collection
// colID. A blank query matches nothing.
func (tx *TxR) CollectionMovieSearch(colID, query string) []CollectionMovieSearchResult {
	col := tx.Collection(colID)
	if strings.TrimSpace(query) == "" {
		return nil
	}
	existing := make(map[string]bool, len(col.Movies()))
	for _, mo := range col.Movies() {
		existing[mo.MovieHead.ID()] = true
	}
	all := tx.MovieWorkList()
	query = strings.ToLower(query)
	var matches []CollectionMovieSearchResult
	for _, mw := range all {
		if strings.Contains(strings.ToLower(mw.Title()), query) {
			matches = append(matches, CollectionMovieSearchResult{
				Movie:        mw,
				InCollection: existing[mw.MovieHead.ID()],
			})
		}
	}
	return matches
}

// CollectionSeriesSearchResult pairs a library series matching a
// picker search with whether it is already in the collection.
type CollectionSeriesSearchResult struct {
	Series       *SeriesWork
	InCollection bool
}

// CollectionSeriesSearch is the series analog of
// CollectionMovieSearch.
func (tx *TxR) CollectionSeriesSearch(colID, query string) []CollectionSeriesSearchResult {
	col := tx.Collection(colID)
	if strings.TrimSpace(query) == "" {
		return nil
	}
	existing := make(map[string]bool, len(col.Series()))
	for _, sr := range col.Series() {
		existing[sr.SeriesHead.ID()] = true
	}
	all := tx.SeriesWorkList()
	query = strings.ToLower(query)
	var matches []CollectionSeriesSearchResult
	for _, sw := range all {
		if strings.Contains(strings.ToLower(sw.Title()), query) {
			matches = append(matches, CollectionSeriesSearchResult{
				Series:       sw,
				InCollection: existing[sw.SeriesHead.ID()],
			})
		}
	}
	return matches
}

func (tx *TxR) collectionBySlug(slug string) (*Collection, bool) {
	colData, ok := txfind1(tx.q.CollectionGetBySlug(slug))
	if !ok {
		return nil, false
	}
	return tx.collectionFromData(colData), true
}

func (tx *TxR) collectionFromData(colData schema.Collection) *Collection {
	movieIDs := txmust1(tx.q.CollectionMovieList(colData.ID))
	allWorks := tx.MovieWorkList()
	memberIDs := make(map[string]bool, len(movieIDs))
	for _, mo := range movieIDs {
		memberIDs[mo.ID] = true
	}
	var movies []*MovieWork
	for _, mw := range allWorks {
		if memberIDs[mw.MovieHead.ID()] {
			movies = append(movies, mw)
		}
	}
	seriesIDs := txmust1(tx.q.CollectionSeriesList(colData.ID))
	allSeries := tx.SeriesWorkList()
	seriesMemberIDs := make(map[string]bool, len(seriesIDs))
	for _, sr := range seriesIDs {
		seriesMemberIDs[sr.ID] = true
	}
	var series []*SeriesWork
	for _, sw := range allSeries {
		if seriesMemberIDs[sw.SeriesHead.ID()] {
			series = append(series, sw)
		}
	}
	slices.SortFunc(movies, func(a, b *MovieWork) int {
		return cmp.Compare(a.Year(), b.Year())
	})
	slices.SortFunc(series, func(a, b *SeriesWork) int {
		return cmp.Compare(a.PremieredOn(), b.PremieredOn())
	})
	return &Collection{
		col:    colData,
		movies: movies,
		series: series,
	}
}

// CollectionStats returns the total number of playable items
// and their combined runtime in minutes.
func (tx *TxR) CollectionStats(id string) (itemCount, runtimeMinutes int64) {
	row := txmust1(tx.q.CollectionGetStats(id))
	return row.Itemcount, row.Runtimeminutes
}

// CollectionPlayables returns the default movie editions and
// all episodes from default series editions in the collection,
// sorted by release date.
func (tx *TxR) CollectionPlayables(id string) []Playable {
	movieIDs := txmust1(tx.q.CollectionMovieList(id))
	var playables []Playable
	for _, mo := range movieIDs {
		med := txmust1(tx.movieEditionBySlug(mo.Slug, ""))
		playables = append(playables, med)
	}
	seriesIDs := txmust1(tx.q.CollectionSeriesList(id))
	for _, sr := range seriesIDs {
		sed := txmust1(tx.seriesEditionBySlug(sr.Slug, ""))
		for ep := range sed.Episodes(Significant) {
			playables = append(playables, ep)
		}
	}
	slices.SortFunc(playables, func(a, b Playable) int {
		return cmp.Compare(a.ReleaseDate(), b.ReleaseDate())
	})
	return playables
}

func (tx *TxRW) CollectionCreate(title string) (*CollectionHead, error) {
	slug, err := tx.collectionFindSlug(title)
	if err != nil {
		return nil, err
	}
	colData, err := tx.q.CollectionCreate(schema.CollectionCreateParams{
		Slug:  slug,
		Title: title,
	})

	if err != nil {
		return nil, err
	}
	err = tx.q.SlugUpsert(schema.SlugUpsertParams{
		Slug:   slug,
		Kind:   "collection",
		Target: colData.ID,
	})

	if err != nil {
		return nil, err
	}
	return &CollectionHead{col: colData}, nil
}

func (tx *TxRW) CollectionMovieAdd(collectionID, movieID string) error {
	return tx.q.CollectionMovieAdd(schema.CollectionMovieAddParams{
		CollectionID: collectionID,
		MovieID:      movieID,
	})

}

func (tx *TxRW) CollectionMovieRemove(collectionID, movieID string) error {
	return tx.q.CollectionMovieDelete(schema.CollectionMovieDeleteParams{
		CollectionID: collectionID,
		MovieID:      movieID,
	})

}

func (tx *TxRW) CollectionSeriesAdd(collectionID, seriesID string) error {
	return tx.q.CollectionSeriesAdd(schema.CollectionSeriesAddParams{
		CollectionID: collectionID,
		SeriesID:     seriesID,
	})

}

func (tx *TxRW) CollectionSeriesRemove(collectionID, seriesID string) error {
	return tx.q.CollectionSeriesDelete(schema.CollectionSeriesDeleteParams{
		CollectionID: collectionID,
		SeriesID:     seriesID,
	})

}

func (tx *TxRW) CollectionBannerIDSet(id, bannerID string) error {
	col, err := tx.q.CollectionGet(id)
	if err != nil {
		return err
	}
	err = tx.q.CollectionBannerIDSet(schema.CollectionBannerIDSetParams{
		BannerID: bannerID,
		ID:       id,
	})

	if err != nil {
		return err
	}
	if isPlaceholderImageID(col.BannerID) {
		return nil
	}
	return tx.imageDelete(col.BannerID)
}

func (tx *TxRW) CollectionTitleSet(id, title string) error {
	if _, err := tx.q.CollectionGet(id); err != nil {
		return err
	}
	err := tx.q.CollectionSetTitle(schema.CollectionSetTitleParams{
		Title: title,
		ID:    id,
	})

	if err != nil {
		return err
	}
	return tx.collectionEnsureSlug(id)
}

// collectionEnsureSlug aligns the Collection's slug with its current
// title, announcing the change on a live rename so a viewer can follow
// it, and keeps the Slug table row in sync. Safe to call on live
// collections (title-change) or trashed ones (restore).
func (tx *TxRW) collectionEnsureSlug(id string) error {
	col, err := tx.q.CollectionGet(id)
	if err != nil {
		return err
	}
	var allow []string
	if col.DeletedAt == nil {
		allow = []string{col.Slug}
	}
	slug, err := tx.collectionFindSlug(col.Title, allow...)
	if err != nil {
		return err
	}
	if slug != col.Slug {
		// Only a live rename is announced; see seriesEnsureSlug.
		if col.DeletedAt == nil {
			tx.emitDetail(Detail{SlugChangeID: id})
		}
		if err := tx.q.CollectionSetSlug(schema.CollectionSetSlugParams{Slug: slug, ID: id}); err != nil {
			return err
		}
	}
	if col.DeletedAt != nil || slug != col.Slug {
		return tx.q.SlugUpsert(schema.SlugUpsertParams{
			Slug: slug, Kind: "collection", Target: id,
		})

	}
	return nil
}

func (tx *TxRW) collectionFindSlug(title string, allow ...string) (string, error) {
	base := xstrings.ToSlug(title)
	if base == "" {
		base = "collection"
	}
	slug := base
	for i := 2; ; i++ {
		if !isReservedSlug(slug) {
			if slices.Contains(allow, slug) {
				return slug, nil
			}
			n, err := tx.q.SlugExists(slug)
			if err != nil {
				return "", err
			}
			if n == 0 {
				return slug, nil
			}
		}
		slug = base + "-" + strconv.Itoa(i)
	}
}
