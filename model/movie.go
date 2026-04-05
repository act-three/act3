package model

import (
	"database/sql"
	"path"
	"slices"
	"strings"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
	"ily.dev/act3/service/tmdb"
	"ily.dev/act3/xstrings"
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

func (tx *TxR) MovieHeadList(ctx Context) ([]*MovieHead, error) {
	a, err := tx.q.MovieList(ctx)
	if err != nil {
		return nil, err
	}
	return newMovieHeadList(a), nil
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
	for _, v := range vidByID {
		ats, err := tx.q.AudioTrackListByVideoID(ctx, v.v.ID)
		if err != nil {
			return nil, err
		}
		for j := range ats {
			v.audioTracks = append(v.audioTracks, &AudioTrack{at: ats[j]})
		}
	}
	videosByEditionID := vidMapByMovieEditionID(mvs, vidByID)

	mo := newMovie(moData, meds, videosByEditionID)
	med := mo.EditionBySlug(edSlug)
	if med == nil {
		return nil, sql.ErrNoRows
	}
	return med, nil
}

// RenditionForDownloadListForMovie is like
// RenditionForDownloadList but for a movie.
func (tx *TxR) RenditionForDownloadListForMovie(
	ctx Context, moID string,
) ([]*RenditionForDownload, error) {
	vids, err := tx.q.VideoListByMovieID(ctx, moID)
	if err != nil {
		return nil, err
	}
	var rends []*RenditionForDownload
	for _, vid := range vids {
		filename := "movie.mkv"
		rends = append(rends, &RenditionForDownload{
			path:     "/-/dl/" + vid.OriginalHash + "/" + filename,
			filename: filename,
			label: "Original (" +
				strings.ReplaceAll(vid.ReleasePath, "/", " / ") + ")",
		})
	}
	return rends, nil
}

func (tx *TxRW) MovieCreate(ctx Context, title, year string) (*MovieWork, error) {
	moID := "mo" + flurry.NewID()
	slug, err := tx.movieFindSlug(ctx, title, year, moID)
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
	err = tx.q.SlugCreate(ctx, schema.SlugCreateParams{
		Slug:   slug,
		Kind:   "movie",
		Target: moID,
	})
	if err != nil {
		return nil, err
	}
	medHead, err := tx.movieEditionCreate(ctx, DefaultEdition, moID, movieEditionParams{
		Title: title,
		Year:  year,
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

	var year string
	if len(movie.ReleaseDate) >= 4 {
		year = movie.ReleaseDate[:4]
	}

	moID := "mo" + flurry.NewID()
	slug, err := tx.movieFindSlug(ctx, movie.Title, year, moID)
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
	err = tx.q.SlugCreate(ctx, schema.SlugCreateParams{
		Slug:   slug,
		Kind:   "movie",
		Target: moID,
	})
	if err != nil {
		return nil, err
	}
	medHead, err := tx.movieEditionCreate(ctx, DefaultEdition, moID, movieEditionParams{
		Title:   movie.Title,
		Summary: movie.Overview,
		Year:    year,
		Runtime: int64(movie.Runtime),
	})
	if err != nil {
		return nil, err
	}
	if movie.PosterPath != nil {
		u := tmdb.PosterURL(*movie.PosterPath)
		err = tx.addTask(ctx, taskFetchMoviePoster, medHead.ID(), u)
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
			MovieEditionHead: MovieEditionHead{ed},
		})
	}
	return works, nil
}
