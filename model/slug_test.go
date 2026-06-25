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
	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		moID := "mo" + flurry.NewID()
		if _, err := tx.q.MovieCreate(schema.MovieCreateParams{ID: moID, Slug: "star-wars"}); err != nil {
			return err
		}
		if err := tx.q.SlugUpsert(schema.SlugUpsertParams{Slug: "star-wars", Kind: "movie", Target: moID}); err != nil {
			return err
		}
		fx.movieID = moID
		def, err := tx.q.MovieEditionCreate(schema.MovieEditionCreateParams{MovieID: moID, Slug: "", Label: DefaultEdition, Title: "Star Wars"})
		if err != nil {
			return err
		}
		fx.movieDefMed = def.ID
		named, err := tx.q.MovieEditionCreate(schema.MovieEditionCreateParams{MovieID: moID, Slug: "special-edition", Label: "Special Edition", Title: "Star Wars"})
		if err != nil {
			return err
		}
		fx.movieNamedMed = named.ID

		srID := "sr" + flurry.NewID()
		if _, err := tx.q.SeriesCreate(schema.SeriesCreateParams{
			ID: srID, Slug: "star-trek-voyager", Title: "Star Trek: Voyager", Status: "Ended", PremieredOn: "1995-01-16",
		}); err != nil {
			return err
		}
		if err := tx.q.SlugUpsert(schema.SlugUpsertParams{Slug: "star-trek-voyager", Kind: "series", Target: srID}); err != nil {
			return err
		}
		fx.seriesID = srID
		defSed, err := tx.q.SeriesEditionCreate(schema.SeriesEditionCreateParams{Label: DefaultEdition, Slug: "", SeriesID: srID})
		if err != nil {
			return err
		}
		fx.seriesDefSed = defSed.ID
		namedSed, err := tx.q.SeriesEditionCreate(schema.SeriesEditionCreateParams{Label: "Remastered", Slug: "remastered", SeriesID: srID})
		if err != nil {
			return err
		}
		fx.seriesNamedSed = namedSed.ID
		sn, err := tx.q.SeasonCreate(schema.SeasonCreateParams{EditionID: defSed.ID, SortKey: "01", Title: "Season 1", Number: 1})
		if err != nil {
			return err
		}
		fx.seasonID = sn.ID
		ep, err := tx.q.EpisodeCreate(schema.EpisodeCreateParams{Title: "Caretaker", Type: "regular", Runtime: 90})
		if err != nil {
			return err
		}
		fx.episodeID = ep.ID
		if err := tx.q.SeasonEpisodeCreate(schema.SeasonEpisodeCreateParams{
			EditionID: defSed.ID, SeasonID: sn.ID, EpisodeID: ep.ID,
			SortKey: 1, Label: "1", Number: 1, Slug: "s1e1-caretaker",
		}); err != nil {
			return err
		}

		col, err := tx.q.CollectionCreate(schema.CollectionCreateParams{
			Slug: "phase-three", Title: "Phase Three",
		})

		if err != nil {
			return err
		}
		if err := tx.q.SlugUpsert(schema.SlugUpsertParams{Slug: "phase-three", Kind: "collection", Target: col.ID}); err != nil {
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

// TestMovieEditionReleaseDateSetRecomputesSlug covers the manual
// workflow where a duplicate-title movie first falls back to a
// title-id slug and then, once a release date supplies a year, should
// settle on the title-year form.
func TestMovieEditionReleaseDateSetRecomputesSlug(t *testing.T) {
	m := newTestModel(t)
	ctx := context.Background()
	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		// An existing movie claims the bare "dune" slug.
		if _, err := tx.MovieCreate("Dune", "1984-12-14"); err != nil {
			return err
		}
		// A new blank movie titled "Dune" can't reuse that slug and,
		// lacking a year, falls back to the title-id form.
		dup, err := tx.MovieCreate("", "")
		if err != nil {
			return err
		}
		moID := dup.MovieHead.ID()
		if err := tx.MovieEditionTitleSet(dup.MovieEditionHead.ID(), "Dune"); err != nil {
			return err
		}
		if got, want := tx.MovieHead(moID).Slug(), "dune-"+moID; got != want {
			t.Fatalf("after title set, slug = %q, want %q", got, want)
		}
		// Setting the release date should recompute the slug to the
		// title-year form now that a year is available.
		if err := tx.MovieEditionReleaseDateSet(dup.MovieEditionHead.ID(), "2021-10-22"); err != nil {
			return err
		}
		if got, want := tx.MovieHead(moID).Slug(), "dune-2021"; got != want {
			t.Fatalf("after release date set, slug = %q, want %q", got, want)
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
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
			err := m.WithTxR(ctx, func(tx *TxR) error {
				got = tx.SlugResolve(tt.components)
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

func TestSlugPath(t *testing.T) {
	m := newTestModel(t)
	fx := newSlugResolveFixture(t, m)
	tests := []struct {
		name  string
		odesc map[string]string
		want  string
	}{
		{"movie default edition", map[string]string{"kind": KindMovieEdition, "mo": fx.movieID, "med": fx.movieDefMed}, "/star-wars"},
		{"movie named edition", map[string]string{"kind": KindMovieEdition, "mo": fx.movieID, "med": fx.movieNamedMed}, "/star-wars/special-edition"},
		{"series default edition", map[string]string{"kind": KindSeriesEdition, "sr": fx.seriesID, "sed": fx.seriesDefSed}, "/star-trek-voyager"},
		{"series named edition", map[string]string{"kind": KindSeriesEdition, "sr": fx.seriesID, "sed": fx.seriesNamedSed}, "/star-trek-voyager/remastered"},
		{"episode of default edition", map[string]string{"kind": KindEpisode, "sr": fx.seriesID, "sed": fx.seriesDefSed, "sn": fx.seasonID, "ep": fx.episodeID}, "/star-trek-voyager/s1e1-caretaker"},
		{"collection overview", map[string]string{"kind": KindCollectionOverview, "col": fx.collectionID}, "/phase-three"},
		{"collection playlist", map[string]string{"kind": KindCollectionPlaylist, "col": fx.collectionID}, "/phase-three/playlist"},
		{"nil", nil, ""},
		{"unknown kind", map[string]string{"kind": "bogus"}, ""},
		{"wrong container", map[string]string{"kind": KindMovieEdition, "mo": "mo-wrong", "med": fx.movieDefMed}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var got string
			err := m.WithTxR(ctx, func(tx *TxR) error {
				got = tx.SlugPath(tt.odesc)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("SlugPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlugPathStaleDescriptor(t *testing.T) {
	m := newTestModel(t)
	fx := newSlugResolveFixture(t, m)
	ctx := context.Background()
	if err := m.WithTxRW(ctx, func(tx *TxRW) error {
		return tx.q.SlugDelete(fx.movieID)
	}); err != nil {
		t.Fatal(err)
	}

	err := m.WithTxR(ctx, func(tx *TxR) error {
		got := tx.SlugPath(map[string]string{
			"kind": KindMovieEdition,
			"mo":   fx.movieID,
			"med":  fx.movieDefMed,
		})
		if got != "" {
			t.Fatalf("SlugPath() = %q, want empty string", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
