package norm

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"strconv"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/service/tvmaze"
)

type Episode struct {
	Season        int
	Episode       schema.EpisodeCreateParams
	SeasonEpisode schema.SeasonEpisodeCreateParams
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

		sortDate := te.Airdate
		if sortDate == "" {
			sortDate = "AAAA-AA-AA"
		}

		var sortKey string
		if te.Number != 0 {
			sortKey = sortDate + "-" + fmt.Sprintf("%05d", te.Number)
		} else {
			sortKey = sortDate + "-" + "AAAAA"
		}

		out = append(out, Episode{
			Season: te.Season,
			Episode: schema.EpisodeCreateParams{
				Title:          te.Name,
				Summary:        te.Summary,
				Type:           te.Type,
				Airdate:        te.Airdate,
				Runtime:        int64(te.Runtime),
				TVmazeImageURL: te.Image.Medium(),
			},
			SeasonEpisode: schema.SeasonEpisodeCreateParams{
				SortKey: sortKey + "-" + strconv.Itoa(te.ID),
				Slug:    rand.Text(),
			},
		})
	}

	return out
}
