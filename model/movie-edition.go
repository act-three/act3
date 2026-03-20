package model

import (
	"fmt"
	"iter"
	"slices"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/xstrings"
)

const (
	// DefaultEdition is the primary edition,
	// present in every movie.
	// Other editions (e.g. "Director's Cut") are optional.
	DefaultEdition = "Default"
)

type MovieEditionHead struct {
	med schema.MovieEdition
}

func (med *MovieEditionHead) ID() string    { return med.med.ID }
func (med *MovieEditionHead) Slug() string  { return med.med.Slug }
func (med *MovieEditionHead) Title() string { return med.med.Title }

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

func (tx *TxRW) MovieEditionCreate(ctx Context, title, movieID string) (*MovieEditionHead, error) {
	slug, err := tx.generateMovieEditionSlug(ctx, title, movieID)
	if err != nil {
		return nil, err
	}
	medData, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
		Title:   title,
		Slug:    slug,
		MovieID: movieID,
	})
	if err != nil {
		return nil, err
	}
	return &MovieEditionHead{medData}, nil
}

func (tx *TxRW) generateMovieEditionSlug(ctx Context, title, movieID string) (string, error) {
	slug := xstrings.ToSlug(title)
	if slug == "" {
		slug = "edition"
	}
	n, err := tx.q.MovieEditionSlugExists(ctx, schema.MovieEditionSlugExistsParams{
		MovieID: movieID,
		Slug:    slug,
	})
	if err != nil {
		return "", err
	}
	for i := 2; n > 0; i++ {
		slug = fmt.Sprintf("%s-%d", slug, i)
		n, err = tx.q.MovieEditionSlugExists(ctx, schema.MovieEditionSlugExistsParams{
			MovieID: movieID,
			Slug:    slug,
		})
		if err != nil {
			return "", err
		}
	}
	return slug, nil
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

// MovieEditionSeq returns an iterator over editions
// sorted by title, for use in UI selectors.
func movieEditionSeq(meds []*MovieEdition) iter.Seq[*MovieEdition] {
	sorted := slices.Clone(meds)
	slices.SortFunc(sorted, func(a, b *MovieEdition) int {
		// Pin the default edition first.
		aDefault := a.Title() == DefaultEdition
		bDefault := b.Title() == DefaultEdition
		if aDefault != bDefault {
			if aDefault {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Title(), b.Title())
	})
	return slices.Values(sorted)
}
