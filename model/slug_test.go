package model

import (
	"context"
	"maps"
	"testing"

	"ily.dev/act3/database/flurry"
	"ily.dev/act3/database/schema"
)

// slugResolveFixture builds one of each addressable work, with both a
// default and a named edition where editions exist, so SlugResolve can
// be exercised across every resolution shape.
type slugResolveFixture struct {
	movieID       string
	movieDefMed   string
	movieNamedMed string

	seriesID       string
	seriesDefSed   string
	seriesNamedSed string
	seasonID       string
	episodeID      string

	collectionID string
}

func newSlugResolveFixture(t *testing.T, m *Model) slugResolveFixture {
	t.Helper()
	ctx := context.Background()
	var fx slugResolveFixture
	err := m.WithTxRW(func(tx *TxRW) error {
		moID := "mo" + flurry.NewID()
		if _, err := tx.q.MovieCreate(ctx, schema.MovieCreateParams{ID: moID, Slug: "star-wars"}); err != nil {
			return err
		}
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{Slug: "star-wars", Kind: "movie", Target: moID}); err != nil {
			return err
		}
		fx.movieID = moID
		def, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{MovieID: moID, Slug: "", Label: DefaultEdition, Title: "Star Wars"})
		if err != nil {
			return err
		}
		fx.movieDefMed = def.ID
		named, err := tx.q.MovieEditionCreate(ctx, schema.MovieEditionCreateParams{MovieID: moID, Slug: "special-edition", Label: "Special Edition", Title: "Star Wars"})
		if err != nil {
			return err
		}
		fx.movieNamedMed = named.ID

		srID := "sr" + flurry.NewID()
		if _, err := tx.q.SeriesCreate(ctx, schema.SeriesCreateParams{
			ID: srID, Slug: "star-trek-voyager", Title: "Star Trek: Voyager", Status: "Ended", PremieredOn: "1995-01-16",
		}); err != nil {
			return err
		}
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{Slug: "star-trek-voyager", Kind: "series", Target: srID}); err != nil {
			return err
		}
		fx.seriesID = srID
		defSed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{Label: DefaultEdition, Slug: "", SeriesID: srID})
		if err != nil {
			return err
		}
		fx.seriesDefSed = defSed.ID
		namedSed, err := tx.q.SeriesEditionCreate(ctx, schema.SeriesEditionCreateParams{Label: "Remastered", Slug: "remastered", SeriesID: srID})
		if err != nil {
			return err
		}
		fx.seriesNamedSed = namedSed.ID
		sn, err := tx.q.SeasonCreate(ctx, schema.SeasonCreateParams{EditionID: defSed.ID, SortKey: "01", Title: "Season 1", Number: 1})
		if err != nil {
			return err
		}
		fx.seasonID = sn.ID
		ep, err := tx.q.EpisodeCreate(ctx, schema.EpisodeCreateParams{Title: "Caretaker", Type: "regular", Runtime: 90})
		if err != nil {
			return err
		}
		fx.episodeID = ep.ID
		if err := tx.q.SeasonEpisodeCreate(ctx, schema.SeasonEpisodeCreateParams{
			EditionID: defSed.ID, SeasonID: sn.ID, EpisodeID: ep.ID,
			SortKey: 1, Label: "1", Number: 1, Slug: "s1e1-caretaker",
		}); err != nil {
			return err
		}

		col, err := tx.q.CollectionCreate(ctx, schema.CollectionCreateParams{
			Slug: "phase-three", Title: "Phase Three",
		})
		if err != nil {
			return err
		}
		if err := tx.q.SlugUpsert(ctx, schema.SlugUpsertParams{Slug: "phase-three", Kind: "collection", Target: col.ID}); err != nil {
			return err
		}
		fx.collectionID = col.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fx
}

func TestSlugResolve(t *testing.T) {
	m := newTestModel(t)
	fx := newSlugResolveFixture(t, m)
	tests := []struct {
		name       string
		components []string
		want       map[string]string // nil means no object
	}{
		{"movie default edition", []string{"star-wars"}, map[string]string{"kind": KindMovieEdition, "mo": fx.movieID, "med": fx.movieDefMed}},
		{"movie named edition", []string{"star-wars", "special-edition"}, map[string]string{"kind": KindMovieEdition, "mo": fx.movieID, "med": fx.movieNamedMed}},
		{"movie unknown edition", []string{"star-wars", "nope"}, nil},
		{"series default edition", []string{"star-trek-voyager"}, map[string]string{"kind": KindSeriesEdition, "sr": fx.seriesID, "sed": fx.seriesDefSed}},
		{"series named edition", []string{"star-trek-voyager", "remastered"}, map[string]string{"kind": KindSeriesEdition, "sr": fx.seriesID, "sed": fx.seriesNamedSed}},
		{"episode of default edition", []string{"star-trek-voyager", "s1e1-caretaker"}, map[string]string{"kind": KindEpisode, "sr": fx.seriesID, "sed": fx.seriesDefSed, "sn": fx.seasonID, "ep": fx.episodeID}},
		{"collection overview", []string{"phase-three"}, map[string]string{"kind": KindCollectionOverview, "col": fx.collectionID}},
		{"collection playlist", []string{"phase-three", "playlist"}, map[string]string{"kind": KindCollectionPlaylist, "col": fx.collectionID}},
		{"collection bad segment", []string{"phase-three", "bogus"}, nil},
		{"not found", []string{"nonesuch"}, nil},
		{"empty", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var got map[string]string
			err := m.WithTxR(func(tx *TxR) error {
				got = tx.SlugResolve(ctx, tt.components)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(got, tt.want) {
				t.Fatalf("resolved to %v, want %v", got, tt.want)
			}
		})
	}
}
