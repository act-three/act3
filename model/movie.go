package model

import (
	"database/sql"
	"errors"
	"path"
	"slices"
	"strings"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
	"ily.dev/act3/priority"
	"ily.dev/act3/service/tmdb"
	"ily.dev/act3/xstrings"
	"kr.dev/errorfmt"
)

// MovieHead is the lightweight representation used in lists.
type MovieHead struct {
	mo schema.Movie
}

func newMovieHeadList(list []schema.Movie) []*MovieHead {
	mos := make([]*MovieHead, len(list))
	for i := range mos {
		mos[i] = &MovieHead{list[i]}
	}
	return mos
}

func (mo *MovieHead) ID() string     { return mo.mo.ID }
func (mo *MovieHead) Slug() string   { return mo.mo.Slug }
func (mo *MovieHead) TMDBID() *int64 { return mo.mo.TMDBID }

func (mo *MovieHead) addr(field string) []string {
	return []string{"movie", mo.ID(), field}
}

func (mo *MovieHead) SlugAddr() []string { return mo.addr("slug") }

func (mo *MovieHead) SlugField() (string, []string) { return mo.Slug(), mo.SlugAddr() }

func (mo *MovieHead) TheaterPath() string {
	return "/" + mo.mo.Slug
}

func (mo *MovieHead) EditorPath() string {
	return "/app/movies/" + mo.mo.Slug
}

// MovieWork contains information needed to display
// a concise representation of a movie edition in UI.
// It contains metadata about the movie itself plus an edition.
type MovieWork struct {
	MovieHead
	MovieEditionHead
}

func (mw *MovieWork) Kind() string { return "movie" }

func (mw *MovieWork) TheaterPath() string {
	return path.Join(mw.MovieHead.TheaterPath(), mw.MovieEditionHead.Slug())
}

func (mw *MovieWork) EditorPath() string {
	return path.Join(mw.MovieHead.EditorPath(), mw.MovieEditionHead.Slug())
}

// Movie is the full representation with editions and their videos.
type Movie struct {
	MovieHead
	editions []*MovieEdition
}

func newMovie(
	moData schema.Movie,
	meds []schema.MovieEdition,
	videosByEditionID map[string][]*Video,
) *Movie {
	mo := &Movie{
		MovieHead: MovieHead{moData},
	}
	for _, medData := range meds {
		med := newMovieEdition(&mo.MovieHead, medData, videosByEditionID)
		mo.editions = append(mo.editions, med)
	}
	return mo
}

func (mo *Movie) EditionBySlug(slug string) *MovieEdition {
	if mo == nil {
		return nil
	}
	for _, med := range mo.editions {
		if med.med.Slug == slug {
			return med
		}
	}
	return nil
}

func (tx *TxR) MovieHead(ctx Context, id string) (*MovieHead, error) {
	moData, err := tx.q.MovieGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &MovieHead{moData}, nil
}

func (tx *TxR) MovieHeadByEditionID(ctx Context, editionID string) (*MovieHead, error) {
	moData, err := tx.q.MovieGetByEditionID(ctx, editionID)
	if err != nil {
		return nil, err
	}
	return &MovieHead{moData}, nil
}

// MovieEditionBySlug looks up a movie by its slug
// and returns the edition matching edSlug
// (empty string for the default edition).
func (tx *TxR) MovieEditionBySlug(ctx Context, slug, edSlug string) (*MovieEdition, error) {
	// TODO(april): avoid loading other editions here.
	moData, err := tx.q.MovieGetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	meds, err := tx.q.MovieEditionListByMovieID(ctx, moData.ID)
	if err != nil {
		return nil, err
	}
	mvs, err := tx.q.MovieVideoListByMovieID(ctx, moData.ID)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByMovieID(ctx, moData.ID)
	if err != nil {
		return nil, err
	}

	vidByID := vidMapByID(vids)
	videosByEditionID := vidMapByMovieEditionID(mvs, vidByID)

	mo := newMovie(moData, meds, videosByEditionID)
	med := mo.EditionBySlug(edSlug)
	if med == nil {
		return nil, sql.ErrNoRows
	}
	return med, nil
}

func (tx *TxR) MovieDownloadList(ctx Context, med *MovieEdition) ([]*RenditionForDownload, error) {
	active := med.ActiveVideo()
	if active == nil {
		return nil, nil
	}

	var rends []*RenditionForDownload
	rfd, err := tx.q.RenditionGetDownloadByVideoID(ctx, active.ID())
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil && rfd.Key != "" {
		rends = append(rends, &RenditionForDownload{
			path:  path.Join("/-/dl", rfd.ID, med.ID()),
			label: "Best Quality MP4 (Recommended)",
		})
	}

	ext := videoExtensionForContentType(active.v.OriginalType)
	rends = append(rends, &RenditionForDownload{
		path:  path.Join("/-/dl", active.ID(), med.ID()),
		label: "Original " + strings.ToUpper(ext),
	})
	return rends, nil
}

func (tx *TxRW) MovieCreate(ctx Context, title, releaseDate string) (*MovieWork, error) {
	moID := "mo" + flurry.NewID()
	slug, err := tx.movieFindSlug(ctx, title, yearFromReleaseDate(releaseDate), moID)
	if err != nil {
		return nil, err
	}
	moData, err := tx.q.MovieCreate(ctx, schema.MovieCreateParams{
		ID:   moID,
		Slug: slug,
	})
	if err != nil {
		return nil, err
	}
	err = tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
		Slug:   slug,
		Kind:   "movie",
		Target: moID,
	})
	if err != nil {
		return nil, err
	}
	medHead, err := tx.movieEditionCreate(ctx, DefaultEdition, moID, movieEditionParams{
		Title:       title,
		ReleaseDate: releaseDate,
	})
	if err != nil {
		return nil, err
	}
	return &MovieWork{
		MovieHead:        MovieHead{moData},
		MovieEditionHead: *medHead,
	}, nil
}

func (tx *TxRW) MovieCreateByTMDBID(
	ctx Context, movie *tmdb.Movie,
) (*MovieWork, error) {
	id64 := int64(movie.ID)

	moID := "mo" + flurry.NewID()
	slug, err := tx.movieFindSlug(ctx, movie.Title, yearFromReleaseDate(movie.ReleaseDate), moID)
	if err != nil {
		return nil, err
	}

	moData, err := tx.q.MovieCreate(ctx,
		schema.MovieCreateParams{
			ID:     moID,
			Slug:   slug,
			TMDBID: &id64,
			IMDBID: movie.IMDBID,
		})
	if err != nil {
		return nil, err
	}
	err = tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
		Slug:   slug,
		Kind:   "movie",
		Target: moID,
	})
	if err != nil {
		return nil, err
	}
	medHead, err := tx.movieEditionCreate(ctx, DefaultEdition, moID, movieEditionParams{
		Title:       movie.Title,
		Summary:     movie.Overview,
		ReleaseDate: movie.ReleaseDate,
		Runtime:     int64(movie.Runtime),
	})
	if err != nil {
		return nil, err
	}
	if movie.PosterPath != nil {
		u := tmdb.PosterURL(*movie.PosterPath)
		err = tx.addTaskWithPriority(ctx, priority.FetchPoster, taskFetchMoviePoster, medHead.ID(), u)
		if err != nil {
			return nil, err
		}
	}
	return &MovieWork{
		MovieHead:        MovieHead{moData},
		MovieEditionHead: *medHead,
	}, nil
}

func (tx *TxR) MovieHeadListByTMDBID(
	ctx Context, ids []*int64,
) ([]*MovieHead, error) {
	a, err := tx.q.MovieListByTMDBID(ctx, ids)
	if err != nil {
		return nil, err
	}
	return newMovieHeadList(a), nil
}

// MovieSearchResult pairs a TMDB search result with the local movie
// it matches, if any.
type MovieSearchResult struct {
	Movie tmdb.SearchResult
	Local *MovieHead
}

// SearchMovies searches TMDB for movies matching query, marking
// results that are already in the library.
func (m *Model) SearchMovies(ctx Context, query string) (results []MovieSearchResult, err error) {
	defer errorfmt.Handlef("search movies: %w", &err)
	found, err := m.tmdb.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	ids := make([]*int64, len(found))
	for i, r := range found {
		id := int64(r.ID)
		ids[i] = &id
	}
	err = m.WithTxR(func(tx *TxR) error {
		heads, err := tx.MovieHeadListByTMDBID(ctx, ids)
		if err != nil {
			return err
		}
		local := make(map[int64]*MovieHead, len(heads))
		for _, mo := range heads {
			local[*mo.TMDBID()] = mo
		}
		for _, r := range found {
			results = append(results, MovieSearchResult{
				Movie: r,
				Local: local[int64(r.ID)],
			})
		}
		return nil
	})
	return results, err
}

// AddMovieByTMDBID creates a local movie from a TMDB entry.
func (m *Model) AddMovieByTMDBID(ctx Context, id int) (mw *MovieWork, err error) {
	defer errorfmt.Handlef("add movie: %w", &err)
	movie, err := m.tmdb.GetMovie(ctx, id)
	if err != nil {
		return nil, err
	}
	err = m.WithTxRW(func(tx *TxRW) error {
		var err error
		mw, err = tx.MovieCreateByTMDBID(ctx, movie)
		return err
	})
	return mw, err
}

// movieEnsureSlug aligns the Movie's slug with its current (default
// edition) title, emitting EventMovieSetSlug if it changes, and keeps
// the Slug table row in sync. Safe to call on live movies (label-/
// title-change) or trashed ones (restore).
func (tx *TxRW) movieEnsureSlug(ctx Context, id string) error {
	mo, err := tx.q.MovieGet(ctx, id)
	if err != nil {
		return err
	}
	med, err := tx.q.MovieEditionGetDefault(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var allow []string
	if mo.DeletedAt == nil {
		allow = []string{mo.Slug}
	}
	slug, err := tx.movieFindSlug(ctx, med.Title, yearFromReleaseDate(med.ReleaseDate), id, allow...)
	if err != nil {
		return err
	}
	if slug != mo.Slug {
		// Only a live rename is announced; see seriesEnsureSlug.
		if mo.DeletedAt == nil {
			tx.onCommit(func() {
				tx.m.emitEvent(&Event{
					Type:    EventMovieSetSlug,
					ID:      id,
					OldText: mo.Slug,
					NewText: slug,
				})
			})
		}
		if err := tx.q.MovieSlugSet(ctx, schema.MovieSlugSetParams{Slug: slug, ID: id}); err != nil {
			return err
		}
	}
	if mo.DeletedAt != nil || slug != mo.Slug {
		return tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{
			Slug: slug, Kind: "movie", Target: id,
		})
	}
	return nil
}

func (tx *TxRW) movieFindSlug(ctx Context, title, year, id string, allow ...string) (string, error) {
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
	if year != "" {
		candidate := slug + "-" + xstrings.ToSlug(year)
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

	// Last resort: slug-id.
	return slug + "-" + id, nil
}

// MovieWorkList returns the default edition of each movie.
func (tx *TxR) MovieWorkList(ctx Context) ([]*MovieWork, error) {
	editions, err := tx.q.MovieEditionListDefault(ctx)
	if err != nil {
		return nil, err
	}
	movies, err := tx.q.MovieList(ctx)
	if err != nil {
		return nil, err
	}
	edByMovieID := make(map[string]schema.MovieEdition, len(editions))
	for _, ed := range editions {
		edByMovieID[ed.MovieID] = ed
	}
	var works []*MovieWork
	for _, mo := range movies {
		ed, ok := edByMovieID[mo.ID]
		if !ok {
			continue // no default edition; skip
		}
		works = append(works, &MovieWork{
			MovieHead:        MovieHead{mo},
			MovieEditionHead: MovieEditionHead{med: ed},
		})
	}
	return works, nil
}
