package model

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strconv"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/xstrings"
)

// CollectionHead is the lightweight representation used in lists.
type CollectionHead struct {
	col schema.Collection
}

func (c *CollectionHead) ID() string       { return c.col.ID }
func (c *CollectionHead) Slug() string     { return c.col.Slug }
func (c *CollectionHead) Title() string    { return c.col.Title }
func (c *CollectionHead) BannerID() string { return c.col.BannerID }

func (c *CollectionHead) addr(field string) []string {
	return []string{"collection", c.ID(), field}
}

func (c *CollectionHead) TitleAddr() []string { return c.addr("title") }
func (c *CollectionHead) SlugAddr() []string  { return c.addr("slug") }

func (c *CollectionHead) TitleField() (string, []string) { return c.Title(), c.TitleAddr() }
func (c *CollectionHead) SlugField() (string, []string)  { return c.Slug(), c.SlugAddr() }

func (c *CollectionHead) BannerPath() string {
	return BannerPath(c.col.BannerID)
}

func (c *CollectionHead) TheaterPath() string {
	return path.Join("/", c.col.Slug)
}

func (c *CollectionHead) EditorPath() string {
	return "/app/collections/" + c.col.Slug
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

func (tx *TxR) CollectionHead(ctx Context, id string) (*CollectionHead, error) {
	colData, err := tx.q.CollectionGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &CollectionHead{colData}, nil
}

func (tx *TxR) CollectionHeadList(ctx Context) ([]*CollectionHead, error) {
	a, err := tx.q.CollectionList(ctx)
	if err != nil {
		return nil, err
	}
	cols := make([]*CollectionHead, len(a))
	for i := range a {
		cols[i] = &CollectionHead{a[i]}
	}
	return cols, nil
}

func (tx *TxR) Collection(ctx Context, id string) (*Collection, error) {
	colData, err := tx.q.CollectionGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return tx.collectionFromData(ctx, colData)
}

func (tx *TxR) CollectionBySlug(ctx Context, slug string) (*Collection, error) {
	colData, err := tx.q.CollectionGetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return tx.collectionFromData(ctx, colData)
}

func (tx *TxR) collectionFromData(ctx Context, colData schema.Collection) (*Collection, error) {
	movieIDs, err := tx.q.CollectionMovieList(ctx, colData.ID)
	if err != nil {
		return nil, err
	}
	allWorks, err := tx.MovieWorkList(ctx)
	if err != nil {
		return nil, err
	}
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
	seriesIDs, err := tx.q.CollectionSeriesList(ctx, colData.ID)
	if err != nil {
		return nil, err
	}
	allSeries, err := tx.SeriesWorkList(ctx)
	if err != nil {
		return nil, err
	}
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
		CollectionHead: CollectionHead{colData},
		movies:         movies,
		series:         series,
	}, nil
}

// CollectionStats returns the total number of playable items
// and their combined runtime in minutes.
func (tx *TxR) CollectionStats(ctx Context, id string) (itemCount, runtimeMinutes int64, err error) {
	row, err := tx.q.CollectionGetStats(ctx, id)
	if err != nil {
		return 0, 0, err
	}
	return row.Itemcount, row.Runtimeminutes, nil
}

// CollectionPlayables returns the default movie editions and
// all episodes from default series editions in the collection,
// sorted by release date.
func (tx *TxR) CollectionPlayables(ctx Context, id string) ([]Playable, error) {
	movieIDs, err := tx.q.CollectionMovieList(ctx, id)
	if err != nil {
		return nil, err
	}
	var playables []Playable
	for _, mo := range movieIDs {
		med, err := tx.MovieEditionBySlug(ctx, mo.Slug, "")
		if err != nil {
			return nil, err
		}
		playables = append(playables, med)
	}
	seriesIDs, err := tx.q.CollectionSeriesList(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, sr := range seriesIDs {
		sed, err := tx.SeriesEditionBySlug(ctx, sr.Slug, "")
		if err != nil {
			return nil, err
		}
		for ep := range sed.Episodes(Significant) {
			playables = append(playables, ep)
		}
	}
	slices.SortFunc(playables, func(a, b Playable) int {
		return cmp.Compare(a.ReleaseDate(), b.ReleaseDate())
	})
	return playables, nil
}

func (tx *TxRW) CollectionCreate(ctx Context, title string) (*CollectionHead, error) {
	slug, err := tx.collectionFindSlug(ctx, title)
	if err != nil {
		return nil, err
	}
	colData, err := tx.q.CollectionCreate(ctx, schema.CollectionCreateParams{
		Slug:  slug,
		Title: title,
	})
	if err != nil {
		return nil, err
	}
	err = tx.q.SlugCreate(ctx, schema.SlugCreateParams{
		Slug:   slug,
		Kind:   "collection",
		Target: colData.ID,
	})
	if err != nil {
		return nil, err
	}
	return &CollectionHead{colData}, nil
}

func (tx *TxRW) CollectionMovieAdd(ctx Context, collectionID, movieID string) error {
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventCollectionMovieAdd,
			ID:      collectionID,
			NewText: movieID,
		})
	})
	return tx.q.CollectionMovieAdd(ctx, schema.CollectionMovieAddParams{
		CollectionID: collectionID,
		MovieID:      movieID,
	})
}

func (tx *TxRW) CollectionMovieRemove(ctx Context, collectionID, movieID string) error {
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventCollectionMovieRemove,
			ID:      collectionID,
			NewText: movieID,
		})
	})
	return tx.q.CollectionMovieDelete(ctx, schema.CollectionMovieDeleteParams{
		CollectionID: collectionID,
		MovieID:      movieID,
	})
}

func (tx *TxRW) CollectionSeriesAdd(ctx Context, collectionID, seriesID string) error {
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventCollectionSeriesAdd,
			ID:      collectionID,
			NewText: seriesID,
		})
	})
	return tx.q.CollectionSeriesAdd(ctx, schema.CollectionSeriesAddParams{
		CollectionID: collectionID,
		SeriesID:     seriesID,
	})
}

func (tx *TxRW) CollectionSeriesRemove(ctx Context, collectionID, seriesID string) error {
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventCollectionSeriesRemove,
			ID:      collectionID,
			NewText: seriesID,
		})
	})
	return tx.q.CollectionSeriesDelete(ctx, schema.CollectionSeriesDeleteParams{
		CollectionID: collectionID,
		SeriesID:     seriesID,
	})
}

func (tx *TxRW) CollectionBannerIDSet(ctx Context, id, bannerID string) error {
	col, err := tx.q.CollectionGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventCollectionChangeBanner,
			ID:      id,
			OldText: col.BannerID,
			NewText: bannerID,
		})
	})
	err = tx.q.CollectionSetBannerID(ctx, schema.CollectionSetBannerIDParams{
		BannerID: bannerID,
		ID:       id,
	})
	if err != nil {
		return err
	}
	if col.BannerID != "" {
		tx.m.store.Remove(col.BannerID)
	}
	return nil
}

func (tx *TxRW) CollectionTitleSet(ctx Context, id, title string) error {
	col, err := tx.q.CollectionGet(ctx, id)
	if err != nil {
		return err
	}
	err = tx.q.CollectionSetTitle(ctx, schema.CollectionSetTitleParams{
		Title: title,
		ID:    id,
	})
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&CollectionHead{col}).TitleAddr(),
			NewText: title,
			OldText: col.Title,
		})
	})
	slug, err := tx.collectionFindSlug(ctx, title, col.Slug)
	if err != nil {
		return err
	}
	if slug == col.Slug {
		return nil
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventCollectionSetSlug,
			ID:      id,
			NewText: slug,
			OldText: col.Slug,
		})
	})
	err = tx.q.SlugUpdate(ctx, schema.SlugUpdateParams{
		Slug:   slug,
		Target: id,
	})
	if err != nil {
		return err
	}
	return tx.q.CollectionSetSlug(ctx, schema.CollectionSetSlugParams{
		Slug: slug,
		ID:   id,
	})
}

func (tx *TxRW) collectionFindSlug(ctx Context, title string, allow ...string) (string, error) {
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
			n, err := tx.q.SlugExists(ctx, slug)
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
