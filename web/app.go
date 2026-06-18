package web

import (
	"context"
	"iter"
	"net/url"

	"ily.dev/domi"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	"ily.dev/act3/ui"
)

type app struct {
	// constant config
	model *model.Model

	// mutable state
	path    string            // request path
	odesc   map[string]string // object descriptor, if any, see model.SlugResolve
	dialog  dialog
	player  *player
	notes   []ui.Note
	noteSeq int
}

// player is the resolved content the open video player renders. Update
// looks it up when the play form is submitted; holding the lookups
// (rather than re-reading each render) keeps the open player a snapshot
// and lets viewPlayer stay free of I/O and error handling.
type player struct {
	video        *model.Video
	episode      *model.Episode          // set for an episode
	movie        *model.MovieEditionHead // set for a movie edition
	qualityOpts  []model.QualityOption
	captionsOpts []model.SubtitleOption
	audioOpts    []model.AudioOption
	audio        string
	subtitle     string
	pinAudio     bool
}

func newApp(ctx context.Context, c *Config, u *url.URL) (*app, cmd) {
	a := &app{
		model: c.Model,
	}
	a.setPath(ctx, u)
	return a, nil
}

func (a *app) Subscriptions(ctx context.Context) domi.Sub[msg.Msg] {
	type key struct{}
	return domi.Subscription(key{}, func(ctx context.Context) iter.Seq[msg.Msg] {
		return func(yield func(msg.Msg) bool) {
			for ev := range a.model.Events(ctx) {
				if !yield((*msg.ModelEvent)(ev)) {
					return
				}
			}
		}
	})
}
