package model

import (
	"iter"
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

func (mo *MovieHead) ID() string       { return mo.mo.ID }
func (mo *MovieHead) Slug() string     { return mo.mo.Slug }
func (mo *MovieHead) Title() string    { return mo.mo.Title }
func (mo *MovieHead) Summary() string  { return mo.mo.Summary }
func (mo *MovieHead) Year() int64      { return mo.mo.Year }
func (mo *MovieHead) Runtime() int64   { return mo.mo.Runtime }
func (mo *MovieHead) ImageURL() string { return mo.mo.ImageURL }
func (mo *MovieHead) TMDBID() *int64   { return mo.mo.TMDBID }
func (mo *MovieHead) IMDBID() *string  { return mo.mo.IMDBID }

func (mo *MovieHead) PlayURL() string {
	return "/" + mo.mo.Slug
}

func (mo *MovieHead) EditURL() string {
	return "/app/movies/" + mo.mo.Slug
}

// YearDisplay returns the year as a string, or empty if
// unknown (0).
func (mo *MovieHead) YearDisplay() string {
	if mo.mo.Year != 0 {
		return strconv.FormatInt(mo.mo.Year, 10)
	}
	return ""
}

// Movie is the full representation with editions and their videos.
type Movie struct {
	MovieHead
	editions  []*MovieEdition
	edByID    map[string]*MovieEdition
	edByTitle map[string]*MovieEdition
}

func newMovie(
	moData schema.Movie,
	meds []schema.MovieEdition,
	videosByEditionID map[string][]*Video,
) *Movie {
	mo := &Movie{
		MovieHead: MovieHead{moData},
		edByID:    map[string]*MovieEdition{},
		edByTitle: map[string]*MovieEdition{},
	}
	for _, medData := range meds {
		med := newMovieEdition(&mo.MovieHead, medData, videosByEditionID)
		mo.editions = append(mo.editions, med)
		mo.edByID[med.ID()] = med
		mo.edByTitle[med.Title()] = med
	}
	return mo
}

func (mo *Movie) EditionByTitle(title string) *MovieEdition {
	if mo == nil {
		return nil
	}
	return mo.edByTitle[title]
}

func (mo *Movie) MovieEditionSeq() iter.Seq[*MovieEdition] {
	return movieEditionSeq(mo.editions)
}

// PlayerURL returns the player URL for a video within this movie.
func (mo *Movie) PlayerURL(v *Video) string {
	return "/-/player/" + v.ID() + "/" + mo.ID()
}

// Transaction methods

func (tx *TxR) MovieHeadList(ctx Context) ([]*MovieHead, error) {
	a, err := tx.q.MovieList(ctx)
	if err != nil {
		return nil, err
	}
	return newMovieHeadList(a), nil
}

func (tx *TxR) MovieBySlug(ctx Context, slug string) (*Movie, error) {
	moData, err := tx.q.MovieGetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return tx.movieFromData(ctx, moData)
}

func (tx *TxR) Movie(ctx Context, id string) (*Movie, error) {
	moData, err := tx.q.MovieGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return tx.movieFromData(ctx, moData)
}

func (tx *TxR) movieFromData(ctx Context, moData schema.Movie) (*Movie, error) {
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
	// Load audio tracks for each video.
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

	return newMovie(moData, meds, videosByEditionID), nil
}

func (tx *TxR) RenditionForStreamingListByMovieID(
	ctx Context, moID string,
) ([]schema.RenditionForStreaming, error) {
	return tx.q.RenditionForStreamingListByMovieID(ctx, moID)
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

func (tx *TxRW) MovieCreate(ctx Context, title string, year int64) (*MovieHead, error) {
	moID := "mo" + flurry.NewID()
	slug, err := tx.generateMovieSlug(ctx, title, year, moID)
	if err != nil {
		return nil, err
	}
	moData, err := tx.q.MovieCreate(ctx, schema.MovieCreateParams{
		ID:      moID,
		Slug:    slug,
		Title:   title,
		Summary: "",
		Year:    year,
	})
	if err != nil {
		return nil, err
	}
	_, err = tx.MovieEditionCreate(ctx, DefaultEdition, moID)
	if err != nil {
		return nil, err
	}
	return &MovieHead{moData}, nil
}

func (tx *TxRW) MovieCreateByTMDBID(
	ctx Context, movie *tmdb.Movie,
) (*MovieHead, error) {
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
			ID:       moID,
			Slug:     slug,
			Title:    movie.Title,
			Summary:  movie.Overview,
			Year:     year,
			Runtime:  int64(movie.Runtime),
			ImageURL: tmdb.ImageURL(movie.PosterPath),
			TMDBID:   &id64,
			IMDBID:   movie.IMDBID,
		})
	if err != nil {
		return nil, err
	}
	_, err = tx.MovieEditionCreate(ctx, DefaultEdition, moID)
	if err != nil {
		return nil, err
	}
	return &MovieHead{moData}, nil
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
