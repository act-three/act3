package norm

import (
	"cmp"
	"crypto/rand"
	"log/slog"
	"math"
	"slices"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/service/tvmaze"
)

type Episode struct {
	Season        int
	Episode       schema.EpisodeCreateParams
	SeasonEpisode schema.SeasonEpisodeCreateParams
	ImageURL      string
	number        int
	tvmazeID      int
}

var knownEpisodeTypes = map[string]bool{
	"regular":               true,
	"significant_special":   true,
	"insignificant_special": true,
}

// TVmazeEpisodes normalizes TVmaze episode data into DB-ready params.
//
// Fields EditionID, SeasonID, and EpisodeID are unset.
// The caller must set them after creating seasons and episodes.
//
// Derived fields Number and Label are also unset.
// Slug is initialized to a placeholder value.
// The caller should call renumberSeason to set all three
// after saving all Episode and SeasonEpisode records.
//
// Episodes with unrecognized types are logged and omitted.
func TVmazeEpisodes(eps []tvmaze.Episode) []Episode {
	var out []Episode

	for _, te := range eps {
		if !knownEpisodeTypes[te.Type] {
			slog.Error("unknown TVmaze episode type; omitting",
				"type", te.Type, "episode", te.Name, "tvmazeID", te.ID)
			continue
		}

		out = append(out, Episode{
			Season:   te.Season,
			number:   te.Number,
			tvmazeID: te.ID,
			ImageURL: te.Image.Medium(),
			Episode: schema.EpisodeCreateParams{
				Title:   te.Name,
				Summary: te.Summary,
				Type:    te.Type,
				Airdate: te.Airdate,
				Runtime: int64(te.Runtime),
			},
			SeasonEpisode: schema.SeasonEpisodeCreateParams{
				Slug: rand.Text(),
			},
		})
	}

	slices.SortFunc(out, func(a, b Episode) int {
		return cmp.Or(
			cmp.Compare(
				cmp.Or(a.Episode.Airdate, "AAAA-AA-AA"),
				cmp.Or(b.Episode.Airdate, "AAAA-AA-AA"),
			),
			cmp.Compare(
				cmp.Or(a.number, math.MaxInt),
				cmp.Or(b.number, math.MaxInt),
			),
			cmp.Compare(a.tvmazeID, b.tvmazeID),
		)
	})
	for i := range out {
		out[i].SeasonEpisode.SortKey = int64(i)
	}
	return out
}
