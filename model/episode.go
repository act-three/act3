package model

import (
	"fmt"
	"path"
	"strings"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/xstrings"
)

type EpisodeType uint

const (
	Regular EpisodeType = 1 << iota
	SignificantSpecial
	InsignificantSpecial

	AnySpecial  = SignificantSpecial | InsignificantSpecial
	AnyEpisode  = Regular | AnySpecial
	Significant = Regular | SignificantSpecial
)

var episodeTypeByName = map[string]EpisodeType{
	"regular":               Regular,
	"special":               SignificantSpecial,
	"insignificant_special": InsignificantSpecial,
}

type seasonEpisode struct {
	ep   schema.Episode
	snep schema.SeasonEpisode
}

type Episode struct {
	ep     schema.Episode
	snep   schema.SeasonEpisode
	type_  EpisodeType
	sn     *SeasonHead
	so     *SeriesEditionHead
	sr     *SeriesHead
	prog   []*progress.Item
	videos []*Video
}

func newEpisode(
	sr *SeriesHead,
	so *SeriesEditionHead,
	sn *SeasonHead,
	snepData schema.SeasonEpisode,
	epData schema.Episode,
	prog []*progress.Item,
	videos []*Video,
) *Episode {
	ep := &Episode{
		ep:     epData,
		snep:   snepData,
		type_:  episodeTypeByName[epData.Type],
		sn:     sn,
		so:     so,
		sr:     sr,
		prog:   prog,
		videos: videos,
	}
	return ep
}

func (ep *Episode) ID() string                 { return ep.ep.ID }
func (ep *Episode) Slug() string               { return ep.snep.Slug }
func (ep *Episode) Title() string              { return ep.ep.Title }
func (ep *Episode) Airdate() string            { return ep.ep.Airdate }
func (ep *Episode) Summary() string            { return ep.ep.Summary }
func (ep *Episode) ImageURL() string           { return ep.ep.TVmazeImageURL }
func (ep *Episode) Type() string               { return ep.ep.Type }
func (ep *Episode) Progress() []*progress.Item { return ep.prog }
func (ep *Episode) Videos() []*Video           { return ep.videos }
func (ep *Episode) SnnEnn() string {
	if eNN := ep.snep.Number; eNN != 0 {
		return fmt.Sprintf("S%02dE%02d", ep.sn.sn.Number, eNN)
	}
	return fmt.Sprintf("S%02d Special", ep.sn.sn.Number)
}

func (ep *Episode) AirdateAddr() []string { return ep.addr("airdate") }
func (ep *Episode) SummaryAddr() []string { return ep.addr("summary") }
func (ep *Episode) TitleAddr() []string   { return ep.addr("title") }
func (ep *Episode) TypeAddr() []string    { return ep.addr("type") }

func (ep *Episode) TitleField() (string, []string) { return ep.Title(), ep.TitleAddr() }
func (ep *Episode) TypeField() (string, []string)  { return ep.Type(), ep.TypeAddr() }

func (ep *Episode) addr(field string) []string {
	return []string{"episode", ep.ep.ID, field}
}

func (ep *Episode) PlayerPath(v *Video) string {
	return fmt.Sprintf("/-/player/%s/%s/%s", v.ID(), ep.ID(), ep.so.ID())
}

func (ep *Episode) HasType(types EpisodeType) bool {
	return types&ep.type_ != 0
}

func (ep *Episode) Label() string {
	l := ep.snep.Label
	if l == "Special" {
		return "Special: " + ep.ep.Title
	}
	return ep.snep.Label + ". " + ep.ep.Title
}

func (ep *Episode) EditDialogPath() string {
	return "/-/dialog/episode-edit/" + ep.ep.ID
}

func (ep *Episode) SeriesHead() *SeriesHead {
	return ep.sr
}

func (ep *Episode) SeriesEditionHead() *SeriesEditionHead {
	return ep.so
}

func (ep *Episode) SeasonHead() *SeasonHead {
	return ep.sn
}

func (ep *Episode) TheaterPath() string {
	if ep.so.Slug() == "" {
		return path.Join("/", ep.sr.Slug(), ep.Slug())
	}
	return path.Join("/", ep.sr.Slug(), ep.so.Slug(), ep.Slug())
}

// EpisodeBySlug looks up an episode by its slug components.
// edSlug selects the edition; empty string selects the default.
func (tx *TxR) EpisodeBySlug(ctx Context, seriesSlug, edSlug, epSlug string) (*Episode, error) {
	sed, err := tx.q.SeriesEditionGetBySlug(ctx, schema.SeriesEditionGetBySlugParams{
		SeriesSlug:  seriesSlug,
		EditionSlug: edSlug,
	})
	if err != nil {
		return nil, err
	}
	snep, err := tx.q.SeasonEpisodeGetBySlug(ctx, schema.SeasonEpisodeGetBySlugParams{
		EditionID: sed.ID,
		Slug:      epSlug,
	})
	if err != nil {
		return nil, err
	}
	return tx.episodeInContext(ctx, snep.EpisodeID, sed.ID, "", "")
}

// Episode is like EpisodeInEdition, but it assumes the Air Date edition.
func (tx *TxR) Episode(ctx Context, id string) (*Episode, error) {
	return tx.episodeInContext(ctx, id, "", AirDate, "")
}

func (tx *TxR) EpisodeInEdition(ctx Context, id, edID string) (*Episode, error) {
	return tx.episodeInContext(ctx, id, edID, "", "")
}

func (tx *TxR) episodeInContext(ctx Context, id, edID, edName, edSlug string) (*Episode, error) {
	epRec, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return nil, err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
	if err != nil {
		return nil, err
	}
	vids, err := tx.q.VideoListByEpisodeID(ctx, id)
	if err != nil {
		return nil, err
	}
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
	ep := &Episode{
		ep:     epRec,
		videos: videos,
	}
	for i, snep := range sneps {
		sn, err := tx.q.SeasonGet(ctx, snep.SeasonID)
		if err != nil {
			return nil, err
		}
		seq, err := tx.q.SeriesEditionGet(ctx, sn.EditionID)
		if err != nil {
			return nil, err
		}
		if seq.ID != edID && seq.Label != edName && seq.Slug != edSlug && i < len(sneps)-1 {
			continue
		}
		sr, err := tx.q.SeriesGet(ctx, seq.SeriesID)
		if err != nil {
			return nil, err
		}
		ep.snep = snep
		ep.sn = &SeasonHead{sn}
		ep.so = &SeriesEditionHead{seq}
		ep.sr = &SeriesHead{sr}
		return ep, nil
	}
	return nil, fmt.Errorf("cannot load ep")
}

func (tx *TxRW) EpisodeAirdateSet(ctx Context, id, airdate string) error {
	ep, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&Episode{ep: ep}).AirdateAddr(),
			NewText: airdate,
			OldText: ep.Airdate,
		})
	})
	return tx.q.EpisodeAirdateSet(ctx, schema.EpisodeAirdateSetParams{
		Airdate: airdate,
		ID:      id,
	})
}

func (tx *TxRW) EpisodeTypeSet(ctx Context, id, typ string) error {
	ep, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&Episode{ep: ep}).TypeAddr(),
			NewText: typ,
			OldText: ep.Type,
		})
	})
	return tx.q.EpisodeTypeSet(ctx, schema.EpisodeTypeSetParams{
		Type: typ,
		ID:   id,
	})
}

func (tx *TxRW) EpisodeTitleSet(ctx Context, id, title string) error {
	ep, err := tx.q.EpisodeGet(ctx, id)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		tx.m.addEvent(&Event{
			Type:    EventLiveUpdate,
			Addr:    (&Episode{ep: ep}).TitleAddr(),
			NewText: title,
			OldText: ep.Title,
		})
	})
	err = tx.q.EpisodeTitleSet(ctx, schema.EpisodeTitleSetParams{
		Title: title,
		ID:    id,
	})
	if err != nil {
		return err
	}
	sneps, err := tx.q.SeasonEpisodeListByEpisodeID(ctx, id)
	if err != nil {
		return err
	}
	for _, snep := range sneps {
		slug, err := tx.generateEpisodeSlug(ctx, snep.EditionID, snep.Slug, title, id)
		if err != nil {
			return err
		}
		if slug != snep.Slug {
			err = tx.q.SeasonEpisodeSlugSet(ctx, schema.SeasonEpisodeSlugSetParams{
				Slug:      slug,
				SeasonID:  snep.SeasonID,
				EpisodeID: id,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// generateEpisodeSlug rebuilds an episode slug after a title change.
// It preserves the base (e.g. "s01e05") and replaces
// the title portion with the slugified new title.
func (tx *TxRW) generateEpisodeSlug(ctx Context, editionID, oldSlug, title, id string) (string, error) {
	// The slug is "sNNeNN-title" or "sNN-special-title".
	// Strip the old title portion from the slug.
	// The base is "sNNeNN" or "sNN-special"; the title follows after a "-".
	slug := episodeSlugBase(oldSlug)
	if titleSlug := xstrings.ToSlug(title); titleSlug != "" {
		slug += "-" + titleSlug
	}
	if slug == oldSlug {
		return slug, nil
	}
	n, err := tx.q.SeasonEpisodeSlugExists(ctx, schema.SeasonEpisodeSlugExistsParams{
		EditionID: editionID,
		Slug:      slug,
	})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return slug, nil
	}
	// Append the ID as a last resort.
	return slug + "-" + id, nil
}

// episodeSlugBase extracts the "sNNeNN" or "sNN-special" prefix
// from an episode slug segment like "s01e05-the-title".
func episodeSlugBase(epPart string) string {
	// Match "sNNeNN": s, digits, e, digits.
	i := 0
	if i >= len(epPart) || epPart[i] != 's' {
		return epPart
	}
	i++
	for i < len(epPart) && epPart[i] >= '0' && epPart[i] <= '9' {
		i++
	}
	if i < len(epPart) && epPart[i] == 'e' {
		i++
		for i < len(epPart) && epPart[i] >= '0' && epPart[i] <= '9' {
			i++
		}
		return epPart[:i]
	}
	// Try "sNN-special".
	if i < len(epPart) && epPart[i] == '-' {
		const special = "-special"
		if strings.HasPrefix(epPart[i:], special) {
			return epPart[:i+len(special)]
		}
	}
	return epPart
}

func epMapByID(eps []schema.Episode) map[string]*schema.Episode {
	epByID := map[string]*schema.Episode{}
	for i := range eps {
		epByID[eps[i].ID] = &eps[i]
	}
	return epByID
}

func snepMapBySeasonID(sneps []schema.SeasonEpisode) map[string][]schema.SeasonEpisode {
	snepBySeasonID := map[string][]schema.SeasonEpisode{}
	for i := range sneps {
		snID := sneps[i].SeasonID
		snepBySeasonID[snID] = append(snepBySeasonID[snID], sneps[i])
	}
	return snepBySeasonID
}
