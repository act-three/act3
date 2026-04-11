package model

import (
	"fmt"
	"path"
	"slices"

	"ily.dev/act3/database/schema"
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

func (med *MovieEdition) PlayerPath() string {
	if v := med.PlayableVideo(); v != nil {
		return med.VideoPlayerPath(v)
	}
	return ""
}

func (med *MovieEdition) VideoPlayerPath(v *Video) string {
	return "/-/player/" + v.ID() + "/" + med.med.ID
}

// PlayableVideo returns the first video with a playlist, or nil.
func (med *MovieEdition) PlayableVideo() *Video {
	for _, v := range med.videos {
		if v.MVPlaylist() != "" {
			return v
		}
	}
	return nil
}

func (tx *TxR) MovieEditionHead(ctx Context, id string) (*MovieEditionHead, error) {
	medData, err := tx.q.MovieEditionGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &MovieEditionHead{med: medData}, nil
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

// movieEditionParams holds metadata for a new movie edition.
type movieEditionParams struct {
	Title       string
	Summary     string
	ReleaseDate string
	Runtime     int64
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
			MovieEditionHead: MovieEditionHead{med: meds[i]},
		}
	}
	return works, nil
}

func (tx *TxRW) movieEditionCreate(ctx Context, label, movieID string, p movieEditionParams) (*MovieEditionHead, error) {
	slug, err := tx.movieEditionFindSlug(ctx, label, movieID)
	if err != nil {
		return nil, err
	}
	medData, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{
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
func (tx *TxRW) MovieEditionClone(ctx Context, srcID string) (*MovieWork, error) {
	src, err := tx.MovieEditionHead(ctx, srcID)
	if err != nil {
		return nil, err
	}
	med, err := tx.movieEditionCreate(ctx, "Copy of "+src.Label(), src.med.MovieID, movieEditionParams{
		Title:       src.med.Title,
		Summary:     src.med.Summary,
		ReleaseDate: src.med.ReleaseDate,
		Runtime:     src.med.Runtime,
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

func (tx *TxRW) MovieEditionLabelSet(ctx Context, id, label string) error {
	med, err := tx.q.MovieEditionGet(ctx, id)
	if err != nil {
		return err
	}
	err = tx.q.MovieEditionLabelSet(ctx, schema.MovieEditionLabelSetParams{
		Label: label,
		ID:    id,
	})
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&MovieEditionHead{med: med}).LabelAddr(),
			NewText: label,
			OldText: med.Label,
		})
	})
	slug, err := tx.movieEditionFindSlug(ctx, label, med.MovieID, med.Slug)
	if err != nil {
		return err
	}
	if slug == med.Slug {
		return nil
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventMovieEditionSetSlug,
			ID:      id,
			NewText: slug,
			OldText: med.Slug,
		})
	})
	return tx.q.MovieEditionSlugSet(ctx, schema.MovieEditionSlugSetParams{
		Slug: slug,
		ID:   id,
	})
}

func (tx *TxRW) MovieEditionTitleSet(ctx Context, id, title string) error {
	med, err := tx.q.MovieEditionGet(ctx, id)
	if err != nil {
		return err
	}
	err = tx.q.MovieEditionTitleSet(ctx, schema.MovieEditionTitleSetParams{
		Title: title,
		ID:    id,
	})
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&MovieEditionHead{med: med}).TitleAddr(),
			NewText: title,
			OldText: med.Title,
		})
	})
	if med.Slug != "" {
		return nil // not the default edition
	}
	mo, err := tx.q.MovieGetByEditionID(ctx, id)
	if err != nil {
		return err
	}
	slug, err := tx.movieFindSlug(ctx, title, yearFromReleaseDate(med.ReleaseDate), mo.ID, mo.Slug)
	if err != nil {
		return err
	}
	if slug == mo.Slug {
		return nil
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventMovieSetSlug,
			ID:      mo.ID,
			NewText: slug,
			OldText: mo.Slug,
		})
	})
	err = tx.q.SlugUpdate(ctx, schema.SlugUpdateParams{
		Slug:   slug,
		Target: mo.ID,
	})
	if err != nil {
		return err
	}
	return tx.q.MovieSlugSet(ctx, schema.MovieSlugSetParams{
		Slug: slug,
		ID:   mo.ID,
	})
}

func (tx *TxRW) MovieEditionReleaseDateSet(ctx Context, id, date string) error {
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&MovieEditionHead{med: schema.MovieEdition{ID: id}}).ReleaseDateAddr(),
			NewText: date,
		})
	})
	return tx.q.MovieEditionReleaseDateSet(ctx, schema.MovieEditionReleaseDateSetParams{
		ReleaseDate: date,
		ID:          id,
	})
}

func (tx *TxRW) MovieEditionRuntimeSet(ctx Context, id string, runtime int64) error {
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&MovieEditionHead{med: schema.MovieEdition{ID: id}}).RuntimeAddr(),
			NewText: fmt.Sprintf("%d", runtime),
		})
	})
	return tx.q.MovieEditionRuntimeSet(ctx, schema.MovieEditionRuntimeSetParams{
		Runtime: runtime,
		ID:      id,
	})
}

func (tx *TxRW) MovieEditionSummarySet(ctx Context, id, summary string) error {
	err := tx.q.MovieEditionSummarySet(ctx, schema.MovieEditionSummarySetParams{
		Summary: summary,
		ID:      id,
	})
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&MovieEditionHead{med: schema.MovieEdition{ID: id}}).SummaryAddr(),
			NewText: summary,
		})
	})
	return nil
}

func (tx *TxRW) MovieEditionPosterIDSet(ctx Context, id, posterID string) error {
	med, err := tx.q.MovieEditionGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type: EventMovieEditionChangePoster,
			ID:   id,
		})
	})
	err = tx.q.MovieEditionPosterIDSet(ctx, schema.MovieEditionPosterIDSetParams{
		PosterID: posterID,
		ID:       id,
	})
	if err != nil {
		return err
	}
	if isPlaceholderImageID(med.PosterID) {
		return nil
	}
	return tx.imageDelete(ctx, med.PosterID)
}

// MovieEditionSetDefault promotes the given edition to be
// the default (Slug="") for its movie.
// The previous default gets a slug generated from its label.
func (tx *TxRW) MovieEditionSetDefault(ctx Context, id string) error {
	med, err := tx.q.MovieEditionGet(ctx, id)
	if err != nil {
		return err
	}
	if med.Slug == "" {
		return nil // already default
	}
	old, err := tx.q.MovieEditionGetDefault(ctx, med.MovieID)
	if err != nil {
		return err
	}
	// Generate a slug for the old default based on its label.
	oldSlug, err := tx.movieEditionFindSlug(ctx, old.Label, med.MovieID)
	if err != nil {
		return err
	}
	// Assign the old default a real slug, freeing up "".
	err = tx.q.MovieEditionSlugSet(ctx, schema.MovieEditionSlugSetParams{
		Slug: oldSlug,
		ID:   old.ID,
	})
	if err != nil {
		return err
	}
	// Promote the new edition to default.
	return tx.q.MovieEditionSlugSet(ctx, schema.MovieEditionSlugSetParams{
		Slug: "",
		ID:   med.ID,
	})
}

func (tx *TxRW) movieEditionFindSlug(ctx Context, label, movieID string, allow ...string) (string, error) {
	for slug := range editionSlugCandidates(label) {
		if slices.Contains(allow, slug) {
			return slug, nil
		}
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
