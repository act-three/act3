package model

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/xstrings"
)

const (
	// DefaultEdition is the label used when automatically creating
	// the initial edition, present in every movie.
	// Other editions (e.g. "Director's Cut") are optional.
	DefaultEdition = "Default Edition"
)

type MovieEditionHead struct {
	med schema.MovieEdition
}

func (med *MovieEditionHead) ID() string          { return med.med.ID }
func (med *MovieEditionHead) Slug() string        { return med.med.Slug }
func (med *MovieEditionHead) Title() string       { return med.med.Title }
func (med *MovieEditionHead) Label() string       { return med.med.Label }
func (med *MovieEditionHead) Summary() string     { return med.med.Summary }
func (med *MovieEditionHead) ReleaseDate() string { return med.med.ReleaseDate }
func (med *MovieEditionHead) Runtime() int64      { return med.med.Runtime }

// Year returns the 4-digit year portion of the release date,
// or the empty string if the release date is unset or too short.
func (med *MovieEditionHead) Year() string {
	return yearFromReleaseDate(med.med.ReleaseDate)
}

func yearFromReleaseDate(d string) string {
	if len(d) < 4 {
		return ""
	}
	return d[:4]
}

func (med *MovieEditionHead) Poster() Image {
	return Image{ID: med.med.PosterID, Kind: ImagePoster}
}

func (med *MovieEditionHead) addr(field string) []string {
	return []string{"movie-edition", med.ID(), field}
}

func (med *MovieEditionHead) TitleAddr() []string       { return med.addr("title") }
func (med *MovieEditionHead) LabelAddr() []string       { return med.addr("label") }
func (med *MovieEditionHead) ReleaseDateAddr() []string { return med.addr("release-date") }
func (med *MovieEditionHead) RuntimeAddr() []string     { return med.addr("runtime") }
func (med *MovieEditionHead) SummaryAddr() []string     { return med.addr("summary") }
func (med *MovieEditionHead) SlugAddr() []string        { return med.addr("slug") }
func (med *MovieEditionHead) PosterAddr() []string      { return med.addr("poster") }

func (med *MovieEditionHead) PosterField() (Image, []string) {
	return med.Poster(), med.PosterAddr()
}

func (med *MovieEditionHead) TitleField() (string, []string) { return med.Title(), med.TitleAddr() }
func (med *MovieEditionHead) LabelField() (string, []string) { return med.Label(), med.LabelAddr() }
func (med *MovieEditionHead) ReleaseDateField() (string, []string) {
	return med.ReleaseDate(), med.ReleaseDateAddr()
}
func (med *MovieEditionHead) RuntimeField() (string, []string) {
	return med.RuntimeString(), med.RuntimeAddr()
}
func (med *MovieEditionHead) SummaryField() (string, []string) {
	return med.Summary(), med.SummaryAddr()
}
func (med *MovieEditionHead) SlugField() (string, []string) { return med.Slug(), med.SlugAddr() }

func (med *MovieEditionHead) RuntimeString() string {
	return fmt.Sprintf("%d", med.med.Runtime)
}

func (med *MovieEditionHead) basename() string {
	var p []string
	p = append(p, med.Title())
	if y := med.Year(); y != "" {
		p = append(p, "("+y+")")
	}
	if med.Slug() != "" {
		p = append(p, med.Label())
	}
	return xstrings.SanitizeFilename(strings.Join(p, " "))
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
		MovieEditionHead: MovieEditionHead{med: medData},
		mo:               mo,
		videos:           videosByEditionID[medData.ID],
	}
	return med
}

func (med *MovieEdition) Videos() []*Video      { return med.videos }
func (med *MovieEdition) MovieHead() *MovieHead { return med.mo }

func (med *MovieEdition) TheaterPath() string {
	return path.Join(med.mo.TheaterPath(), med.Slug())
}

func (med *MovieEdition) EditorPath() string {
	return path.Join(med.mo.EditorPath(), med.Slug())
}

func (med *MovieEdition) Info() []string {
	if med.Label() != DefaultEdition {
		return []string{med.Label()}
	}
	return nil
}

func (med *MovieEdition) ImageField() (Image, []string) { return med.PosterField() }
func (med *MovieEdition) ImageAspect() (n, d int)       { return ImagePoster.Aspect() }

func (med *MovieEdition) Runtime() string {
	if r := med.MovieEditionHead.Runtime(); r > 0 {
		return fmt.Sprintf("%d", r)
	}
	return ""
}

func (med *MovieEdition) PlayIDs() PlayIDs {
	v := med.ActiveVideo()
	if v == nil {
		return PlayIDs{}
	}
	return PlayIDs{
		VideoID:        v.ID(),
		MovieEditionID: med.med.ID,
	}
}

// ActiveVideo returns the video marked Active for this edition, or
// nil if none is set. Theater contexts use this to pick the canonical
// playable video.
func (med *MovieEdition) ActiveVideo() *Video {
	for _, v := range med.videos {
		if v.active {
			return v
		}
	}
	return nil
}

func (tx *TxR) MovieEditionHead(id string) (*MovieEditionHead, error) {
	medData, err := tx.q.MovieEditionGet(id)
	if err != nil {
		return nil, err
	}
	return &MovieEditionHead{med: medData}, nil
}

func (tx *TxR) MovieEdition(id string) (*MovieEdition, error) {
	medData, err := tx.q.MovieEditionGet(id)
	if err != nil {
		return nil, err
	}
	moData, err := tx.q.MovieGetByEditionID(id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByMovieEditionID(id)
	if err != nil {
		return nil, err
	}
	mvs, err := tx.q.MovieVideoListByMovieEditionID(id)
	if err != nil {
		return nil, err
	}
	activeByVID := map[string]bool{}
	for _, mv := range mvs {
		if mv.Active != 0 {
			activeByVID[mv.VideoID] = true
		}
	}

	mo := &MovieHead{moData}
	var videos []*Video
	for i := range vids {
		videos = append(videos, &Video{v: vids[i], active: activeByVID[vids[i].ID]})
	}
	return newMovieEdition(mo, medData, map[string][]*Video{id: videos}), nil
}

// movieEditionParams holds metadata for a new movie edition.
type movieEditionParams struct {
	Title       string
	Summary     string
	ReleaseDate string
	Runtime     int64
}

func (tx *TxR) MovieEditionList(mo *MovieHead) ([]*MovieWork, error) {
	meds, err := tx.q.MovieEditionListByMovieID(mo.ID())
	if err != nil {
		return nil, err
	}
	works := make([]*MovieWork, len(meds))
	for i := range meds {
		works[i] = &MovieWork{
			MovieHead:        *mo,
			MovieEditionHead: MovieEditionHead{med: meds[i]},
		}
	}
	return works, nil
}

func (tx *TxRW) movieEditionCreate(label, movieID string, p movieEditionParams) (*MovieEditionHead, error) {
	slug, err := tx.movieEditionFindSlug(label, movieID)
	if err != nil {
		return nil, err
	}
	medData, err := tx.q.MovieEditionCreate(schema.MovieEditionCreateParams{
		Title:       p.Title,
		Label:       label,
		Slug:        slug,
		MovieID:     movieID,
		Summary:     p.Summary,
		ReleaseDate: p.ReleaseDate,
		Runtime:     p.Runtime,
	})

	if err != nil {
		return nil, err
	}
	return &MovieEditionHead{med: medData}, nil
}

// MovieEditionClone creates a new edition by copying metadata
// from the edition with the given srcID.
// The new edition is labeled "Copy of {original label}".
func (tx *TxRW) MovieEditionClone(srcID string) (*MovieWork, error) {
	src, err := tx.MovieEditionHead(srcID)
	if err != nil {
		return nil, err
	}
	med, err := tx.movieEditionCreate("Copy of "+src.Label(), src.med.MovieID, movieEditionParams{
		Title:       src.med.Title,
		Summary:     src.med.Summary,
		ReleaseDate: src.med.ReleaseDate,
		Runtime:     src.med.Runtime,
	})
	if err != nil {
		return nil, err
	}
	moData, err := tx.q.MovieGet(src.med.MovieID)
	if err != nil {
		return nil, err
	}
	return &MovieWork{
		MovieHead:        MovieHead{moData},
		MovieEditionHead: *med,
	}, nil
}

func (tx *TxRW) MovieEditionLabelSet(id, label string) error {
	if _, err := tx.q.MovieEditionGet(id); err != nil {
		return err
	}
	err := tx.q.MovieEditionLabelSet(schema.MovieEditionLabelSetParams{
		Label: label,
		ID:    id,
	})

	if err != nil {
		return err
	}
	return tx.movieEditionEnsureSlug(id)
}

// movieEditionEnsureSlug aligns the edition's slug with its current
// label, announcing the change on a live rename so a viewer can follow
// it. Safe to call when the edition is live (e.g. after a label change)
// or trashed (during restore); when live, the current slug is held in
// reserve so findSlug doesn't reject it as self-collision.
func (tx *TxRW) movieEditionEnsureSlug(id string) error {
	med, err := tx.q.MovieEditionGet(id)
	if err != nil {
		return err
	}
	var allow []string
	if med.DeletedAt == nil {
		allow = []string{med.Slug}
	}
	slug, err := tx.movieEditionFindSlug(med.Label, med.MovieID, allow...)
	if err != nil {
		return err
	}
	if slug == med.Slug {
		return nil
	}
	// Only a live rename is announced: a trashed edition's pages
	// have no viewers to follow it.
	if med.DeletedAt == nil {
		tx.emitDetail(Detail{SlugChangeID: id})
	}
	return tx.q.MovieEditionSlugSet(schema.MovieEditionSlugSetParams{
		Slug: slug,
		ID:   id,
	})

}

func (tx *TxRW) MovieEditionTitleSet(id, title string) error {
	med, err := tx.q.MovieEditionGet(id)
	if err != nil {
		return err
	}
	err = tx.q.MovieEditionTitleSet(schema.MovieEditionTitleSetParams{
		Title: title,
		ID:    id,
	})

	if err != nil {
		return err
	}
	if med.Slug != "" {
		return nil // not the default edition
	}
	mo, err := tx.q.MovieGetByEditionID(id)
	if err != nil {
		return err
	}
	return tx.movieEnsureSlug(mo.ID)
}

func (tx *TxRW) MovieEditionReleaseDateSet(id, date string) error {
	return tx.q.MovieEditionReleaseDateSet(schema.MovieEditionReleaseDateSetParams{
		ReleaseDate: date,
		ID:          id,
	})

}

func (tx *TxRW) MovieEditionRuntimeSet(id string, runtime int64) error {
	return tx.q.MovieEditionRuntimeSet(schema.MovieEditionRuntimeSetParams{
		Runtime: runtime,
		ID:      id,
	})

}

func (tx *TxRW) MovieEditionSummarySet(id, summary string) error {
	return tx.q.MovieEditionSummarySet(schema.MovieEditionSummarySetParams{
		Summary: summary,
		ID:      id,
	})

}

func (tx *TxRW) MovieEditionPosterIDSet(id, posterID string) error {
	med, err := tx.q.MovieEditionGet(id)
	if err != nil {
		return err
	}
	err = tx.q.MovieEditionPosterIDSet(schema.MovieEditionPosterIDSetParams{
		PosterID: posterID,
		ID:       id,
	})

	if err != nil {
		return err
	}
	if isPlaceholderImageID(med.PosterID) {
		return nil
	}
	return tx.imageDelete(med.PosterID)
}

// MovieEditionSetDefault promotes the given edition to be
// the default (Slug="") for its movie.
// The previous default gets a slug generated from its label.
func (tx *TxRW) MovieEditionSetDefault(id string) error {
	med, err := tx.q.MovieEditionGet(id)
	if err != nil {
		return err
	}
	if med.Slug == "" {
		return nil // already default
	}
	old, err := tx.q.MovieEditionGetDefault(med.MovieID)
	if err != nil {
		return err
	}
	oldSlug, err := tx.movieEditionFindSlug(old.Label, med.MovieID)
	if err != nil {
		return err
	}
	tx.emitDetail(Detail{SlugChangeID: old.ID})
	tx.emitDetail(Detail{SlugChangeID: med.ID})
	err = tx.q.MovieEditionSlugSet(schema.MovieEditionSlugSetParams{
		Slug: oldSlug,
		ID:   old.ID,
	})

	if err != nil {
		return err
	}
	return tx.q.MovieEditionSlugSet(schema.MovieEditionSlugSetParams{
		Slug: "",
		ID:   med.ID,
	})

}

func (tx *TxRW) movieEditionFindSlug(label, movieID string, allow ...string) (string, error) {
	for slug := range editionSlugCandidates(label) {
		if slices.Contains(allow, slug) {
			return slug, nil
		}
		n, err := tx.q.MovieEditionSlugExists(schema.MovieEditionSlugExistsParams{
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
		v := vidByID[mv.VideoID]
		if v == nil {
			continue
		}
		clone := *v
		clone.active = mv.Active != 0
		m[mv.MovieEditionID] = append(m[mv.MovieEditionID], &clone)
	}
	return m
}
