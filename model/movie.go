package model

import (
	"database/sql"
	"path"
	"strconv"
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
func (mo *MovieHead) Title() string  { return mo.mo.Title }
func (mo *MovieHead) TMDBID() *int64 { return mo.mo.TMDBID }

func (mo *MovieHead) TheaterURL() string {
	return "/" + mo.mo.Slug
}

func (mo *MovieHead) EditorURL() string {
	return "/app/movies/" + mo.mo.Slug
}

// MovieWork contains information needed to display
// a concise representation of a movie edition in UI.
// It contains metadata about the movie itself plus an edition.
type MovieWork struct {
	MovieHead
	MovieEditionHead
}

func (mw *MovieWork) Title() string { return mw.MovieHead.Title() }

func (mw *MovieWork) TheaterURL() string {
	if mw.MovieEditionHead.Slug() == "" {
		return mw.MovieHead.TheaterURL()
	}
	return path.Join(mw.MovieHead.TheaterURL(), mw.MovieEditionHead.Slug())
}

func (mw *MovieWork) EditorURL() string {
	if mw.MovieEditionHead.Slug() == "" {
		return mw.MovieHead.EditorURL()
	}
	return mw.MovieHead.EditorURL() + "/" + mw.MovieEditionHead.Slug()
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

// Transaction methods

func (tx *TxR) MovieHeadList(ctx Context) ([]*MovieHead, error) {
	a, err := tx.q.MovieList(ctx)
	if err != nil {
		return nil, err
	}
	return newMovieHeadList(a), nil
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
			url:      "/-/dl/" + vid.OriginalHash + "/" + filename,
			filename: filename,
			label: "Original (" +
				strings.ReplaceAll(vid.ReleasePath, "/", " / ") + ")",
		})
	}
	return rends, nil
}

func (tx *TxRW) MovieCreate(ctx Context, title string, year int64) (*MovieWork, error) {
	moID := "mo" + flurry.NewID()
	slug, err := tx.generateMovieSlug(ctx, title, year, moID)
	if err != nil {
		return nil, err
	}
	moData, err := tx.q.MovieCreate(ctx, schema.MovieCreateParams{
		ID:    moID,
		Slug:  slug,
		Title: title,
	})
	if err != nil {
		return nil, err
	}
	medHead, err := tx.movieEditionCreate(ctx, DefaultEdition, moID, movieEditionParams{
		Year: year,
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

	var year int64
	if len(movie.ReleaseDate) >= 4 {
		y, err := strconv.ParseInt(movie.ReleaseDate[:4], 10, 64)
		if err == nil {
			year = y
		}
	}

	moID := "mo" + flurry.NewID()
	slug, err := tx.generateMovieSlug(ctx, movie.Title, year, moID)
	if err != nil {
		return nil, err
	}

	moData, err := tx.q.MovieCreate(ctx,
		schema.MovieCreateParams{
			ID:     moID,
			Slug:   slug,
			Title:  movie.Title,
			TMDBID: &id64,
			IMDBID: movie.IMDBID,
		})
	if err != nil {
		return nil, err
	}
	medHead, err := tx.movieEditionCreate(ctx, DefaultEdition, moID, movieEditionParams{
		Summary:  movie.Overview,
		Year:     year,
		Runtime:  int64(movie.Runtime),
		ImageURL: tmdb.ImageURL(movie.PosterPath),
	})
	if err != nil {
		return nil, err
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

func (tx *TxRW) generateMovieSlug(ctx Context, title string, year int64, id string) (string, error) {
	slug := xstrings.ToSlug(title)
	if slug == "" {
		slug = id
	}
	if !isReservedSlug(slug) {
		n, err := tx.q.MovieSlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		sn, err := tx.q.SeriesSlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if n == 0 && sn == 0 {
			return slug, nil
		}
	}

	// Try slug-year.
	if year != 0 {
		candidate := slug + "-" + strconv.FormatInt(year, 10)
		n, err := tx.q.MovieSlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		sn, err := tx.q.SeriesSlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if n == 0 && sn == 0 {
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
