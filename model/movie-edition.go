package model

import (
	"strconv"

	"ily.dev/act3/database/schema"
)

const (
	// DefaultEdition is the title used when automatically creating
	// the initial edition, present in every movie.
	// Other editions (e.g. "Director's Cut") are optional.
	DefaultEdition = "Default Edition"
)

type MovieEditionHead struct {
	med schema.MovieEdition
}

func (med *MovieEditionHead) ID() string       { return med.med.ID }
func (med *MovieEditionHead) Slug() string     { return med.med.Slug }
func (med *MovieEditionHead) Title() string    { return med.med.Title }
func (med *MovieEditionHead) Summary() string  { return med.med.Summary }
func (med *MovieEditionHead) Year() string     { return med.med.Year }
func (med *MovieEditionHead) Runtime() int64   { return med.med.Runtime }
func (med *MovieEditionHead) ImageURL() string { return med.med.ImageURL }

// RuntimeDisplay returns the runtime as a string, or empty if unknown (0).
func (med *MovieEditionHead) RuntimeDisplay() string {
	if med.med.Runtime != 0 {
		return strconv.FormatInt(med.med.Runtime, 10)
	}
	return ""
}

type MovieEdition struct {
	MovieEditionHead
	videos []*Video
	mo     *MovieHead
}

func newMovieEdition(
	mo *MovieHead,
	medData schema.MovieEdition,
	videosByEditionID map[string][]*Video,
) *MovieEdition {
	med := &MovieEdition{
		MovieEditionHead: MovieEditionHead{medData},
		mo:               mo,
		videos:           videosByEditionID[medData.ID],
	}
	return med
}

func (med *MovieEdition) Videos() []*Video      { return med.videos }
func (med *MovieEdition) MovieHead() *MovieHead { return med.mo }

func (med *MovieEdition) PlayerURL(v *Video) string {
	return "/-/player/" + v.ID() + "/" + med.med.ID
}

// Playable returns the first video with a playlist, or nil.
func (med *MovieEdition) Playable() *Video {
	for _, v := range med.videos {
		if v.MVPlaylist() != "" {
			return v
		}
	}
	return nil
}

// Transaction methods

func (tx *TxR) MovieEditionHead(ctx Context, id string) (*MovieEditionHead, error) {
	medData, err := tx.q.MovieEditionGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &MovieEditionHead{medData}, nil
}

func (tx *TxR) MovieEdition(ctx Context, id string) (*MovieEdition, error) {
	medData, err := tx.q.MovieEditionGet(ctx, id)
	if err != nil {
		return nil, err
	}
	moData, err := tx.q.MovieGetByEditionID(ctx, id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByMovieEditionID(ctx, id)
	if err != nil {
		return nil, err
	}

	mo := &MovieHead{moData}
	var videos []*Video
	for i := range vids {
		v := &Video{v: vids[i]}
		ats, err := tx.q.AudioTrackListByVideoID(ctx, vids[i].ID)
		if err != nil {
			return nil, err
		}
		for j := range ats {
			v.audioTracks = append(v.audioTracks, &AudioTrack{at: ats[j]})
		}
		videos = append(videos, v)
	}
	return newMovieEdition(mo, medData, map[string][]*Video{id: videos}), nil
}

// movieEditionParams holds optional metadata for a new movie edition.
type movieEditionParams struct {
	Summary  string
	Year     string
	Runtime  int64
	ImageURL string
}

func (tx *TxR) MovieEditionList(ctx Context, mo *MovieHead) ([]*MovieWork, error) {
	meds, err := tx.q.MovieEditionListByMovieID(ctx, mo.ID())
	if err != nil {
		return nil, err
	}
	works := make([]*MovieWork, len(meds))
	for i := range meds {
		works[i] = &MovieWork{
			MovieHead:        *mo,
			MovieEditionHead: MovieEditionHead{meds[i]},
		}
	}
	return works, nil
}

func (tx *TxRW) movieEditionCreate(ctx Context, title, movieID string, p movieEditionParams) (*MovieEditionHead, error) {
	slug, err := tx.generateMovieEditionSlug(ctx, title, movieID)
	if err != nil {
		return nil, err
	}
	medData, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
		Title:    title,
		Slug:     slug,
		MovieID:  movieID,
		Summary:  p.Summary,
		Year:     p.Year,
		Runtime:  p.Runtime,
		ImageURL: p.ImageURL,
	})
	if err != nil {
		return nil, err
	}
	return &MovieEditionHead{medData}, nil
}

// MovieEditionClone creates a new edition by copying metadata
// from the edition with the given srcID.
// The new edition is titled "Copy of {original title}".
func (tx *TxRW) MovieEditionClone(ctx Context, srcID string) (*MovieWork, error) {
	src, err := tx.MovieEditionHead(ctx, srcID)
	if err != nil {
		return nil, err
	}
	med, err := tx.movieEditionCreate(ctx, "Copy of "+src.Title(), src.med.MovieID, movieEditionParams{
		Summary:  src.med.Summary,
		Year:     src.med.Year,
		Runtime:  src.med.Runtime,
		ImageURL: src.med.ImageURL,
	})
	if err != nil {
		return nil, err
	}
	moData, err := tx.q.MovieGet(ctx, src.med.MovieID)
	if err != nil {
		return nil, err
	}
	return &MovieWork{
		MovieHead:        MovieHead{moData},
		MovieEditionHead: *med,
	}, nil
}

func (tx *TxRW) MovieEditionSlugSet(ctx Context, id, slug string) error {
	return tx.q.MovieEditionSlugSet(ctx, schema.MovieEditionSlugSetParams{
		Slug: slug,
		ID:   id,
	})
}

func (tx *TxRW) MovieEditionTitleSet(ctx Context, id, title string) error {
	return tx.q.MovieEditionTitleSet(ctx, schema.MovieEditionTitleSetParams{
		Title: title,
		ID:    id,
	})
}

func (tx *TxRW) MovieEditionYearSet(ctx Context, id, year string) error {
	return tx.q.MovieEditionYearSet(ctx, schema.MovieEditionYearSetParams{
		Year: year,
		ID:   id,
	})
}

func (tx *TxRW) MovieEditionRuntimeSet(ctx Context, id string, runtime int64) error {
	return tx.q.MovieEditionRuntimeSet(ctx, schema.MovieEditionRuntimeSetParams{
		Runtime: runtime,
		ID:      id,
	})
}

func (tx *TxRW) generateMovieEditionSlug(ctx Context, title, movieID string) (string, error) {
	for slug := range editionSlugCandidates(title) {
		n, err := tx.q.MovieEditionSlugExists(ctx, schema.MovieEditionSlugExistsParams{
			MovieID: movieID,
			Slug:    slug,
		})
		if err != nil {
			return "", err
		}
		if n == 0 {
			return slug, nil
		}
	}
	panic("unreachable")
}

// vidMapByMovieEditionID groups videos by their movie edition ID.
func vidMapByMovieEditionID(mvs []schema.MovieVideo, vidByID map[string]*Video) map[string][]*Video {
	m := map[string][]*Video{}
	for _, mv := range mvs {
		if v := vidByID[mv.VideoID]; v != nil {
			m[mv.MovieEditionID] = append(m[mv.MovieEditionID], v)
		}
	}
	return m
}
