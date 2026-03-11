package view

import (
	"fmt"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

func MediaPlayerForEpisode(v *model.Video, ep *model.Episode, qualityOpts []model.QualityOption) html.Node {
	return mediaPlayer(v, mediaPlayerTitleForEpisode(ep), qualityOpts)
}

func MediaPlayerForMovie(v *model.Video, mo *model.Movie, qualityOpts []model.QualityOption) html.Node {
	return mediaPlayer(v, mediaPlayerTitleForMovie(mo), qualityOpts)
}

func mediaPlayerTitleForMovie(mo *model.Movie) string {
	if y := mo.Year(); y != 0 {
		return fmt.Sprintf("%s (%d)", mo.Title(), y)
	}
	return mo.Title()
}

func mediaPlayer(v *model.Video, title string, qualityOpts []model.QualityOption) html.Node {
	var opts []PlayerQualityOption
	for _, q := range qualityOpts {
		opts = append(opts, PlayerQualityOption{Label: q.Label, URL: q.URL})
	}
	return turbo.Frame("player")(
		Player(v.PlaylistURL(), "application/vnd.apple.mpegurl", title, opts),
	)
}

func mediaPlayerTitleForEpisode(ep *model.Episode) string {
	if ep == nil {
		return "Episode " + ep.ID()
	}
	year := ""
	if d := ep.Airdate(); d != "" {
		y, _, _ := strings.Cut(d, "-")
		year = " (" + y + ")"
	}
	return fmt.Sprintf("%s %s %s%s",
		ep.SeriesHead().Title(),
		ep.SnnEnn(),
		ep.Title(),
		year,
	)
}
