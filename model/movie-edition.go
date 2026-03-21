package model

import (
	"fmt"
	"iter"
	"path"
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

func (med *MovieEdition) EditURL() string {
	if med.med.IsDefault != nil {
		return med.mo.EditURL()
	}
	return path.Join(
		med.mo.EditURL(),
		*med.med.Slug,
	)
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
	slug, isDefault, err := tx.generateMovieEditionSlug(ctx, title, movieID)
	if err != nil {
		return nil, err
	}
	p := schema.MovieEditionCreateParams{
		Title:   title,
		MovieID: movieID,
	}
	if slug != "" {
		p.Slug = &slug
	}
	if isDefault {
		p.IsDefault = new(int64(1))
	}
	medData, err := tx.q.MovieEditionCreate(ctx, p)
	if err != nil {
		return nil, err
	}
	return &MovieEditionHead{medData}, nil
}

func (tx *TxRW) generateMovieEditionSlug(ctx Context, title, movieID string) (slug string, isDefault bool, err error) {
	n, err := tx.q.MovieEditionDefaultExists(ctx, movieID)
	if err != nil {
		return "", false, err
	}
	if n == 0 {
		return "", true, nil
	}
	slug = xstrings.ToSlug(title)
	if slug == "" {
		slug = "edition"
	}
	n, err = tx.q.MovieEditionSlugExists(ctx, schema.MovieEditionSlugExistsParams{
		MovieID: movieID,
		Slug:    &slug,
	})
	if err != nil {
		return "", false, err
	}
	base := slug
	for i := 2; n > 0; i++ {
		slug = fmt.Sprintf("%s-%d", base, i)
		n, err = tx.q.MovieEditionSlugExists(ctx, schema.MovieEditionSlugExistsParams{
			MovieID: movieID,
			Slug:    &slug,
		})
		if err != nil {
			return "", false, err
		}
	}
	return slug, false, nil
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
