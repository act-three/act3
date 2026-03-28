package norm

import (
	"fmt"
	"strconv"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/service/tvmaze"
	"ily.dev/act3/xstrings"
)

type Episode struct {
	Season        int
	Episode       schema.EpisodeCreateParams
	SeasonEpisode schema.SeasonEpisodeCreateParams
}

// TVmazeEpisodes normalizes TVmaze episode data into DB-ready params.
// The returned SeasonEpisodeCreateParams have EditionID, SeasonID,
// and EpisodeID left empty — the caller fills those after creating
// seasons and episodes.
//
// The returned slices have the same length and order as eps.
func TVmazeEpisodes(eps []tvmaze.Episode) []Episode {
	out := make([]Episode, len(eps))

	seenSlug := map[string]bool{}
	for i, te := range eps {
		sortDate := te.Airdate
		if sortDate == "" {
			sortDate = "AAAA-AA-AA"
		}

		var epSlug, sortKey, label string
		if te.Number != 0 {
			epSlug = fmt.Sprintf("s%02de%02d", te.Season, te.Number)
			sortKey = sortDate + "-" + fmt.Sprintf("%05d", te.Number)
			label = strconv.Itoa(te.Number)
		} else {
			epSlug = fmt.Sprintf("s%02d-special", te.Season)
			sortKey = sortDate + "-" + "AAAAA"
			switch te.Type {
			case "special", "insignificant_special":
				label = "Special"
			default:
				label = "Unknown"
			}
		}

		slug := epSlug
		if titleSlug := xstrings.ToSlug(te.Name); titleSlug != "" {
			slug += "-" + titleSlug
		}
		if seenSlug[slug] {
			for j := 2; ; j++ {
				try := slug + "-" + strconv.Itoa(j)
				if !seenSlug[try] {
					slug = try
					break
				}
			}
		}
		seenSlug[slug] = true

		out[i] = Episode{
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
				Number:  int64(te.Number),
				Label:   label,
				Slug:    slug,
			},
		}
	}

	return out
}
