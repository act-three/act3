package model

import (
	"fmt"
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

func (c *CollectionHead) EditorPath() string {
	return "/app/collections/" + c.col.Slug
}

// Collection is the full representation with associated movies and series.
type Collection struct {
	CollectionHead
	movies []*MovieHead
	series []*SeriesHead
}

func (c *Collection) Movies() []*MovieHead  { return c.movies }
func (c *Collection) Series() []*SeriesHead { return c.series }

func (c *Collection) MovieCountAddr() []string { return c.addr("movie-count") }
func (c *Collection) MovieCountField() (string, []string) {
	return fmt.Sprintf("%d Movies", len(c.movies)), c.MovieCountAddr()
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
	movies, err := tx.q.CollectionMovieList(ctx, colData.ID)
	if err != nil {
		return nil, err
	}
	series, err := tx.q.CollectionSeriesList(ctx, colData.ID)
	if err != nil {
		return nil, err
	}
	return &Collection{
		CollectionHead: CollectionHead{colData},
		movies:         newMovieHeadList(movies),
		series:         newSeriesHeadList(series),
	}, nil
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
