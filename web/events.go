package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/model/progress"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
)

func (c *Config) events(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	rc := http.NewResponseController(w)
	for ev := range c.Model.Events(ctx) {
		if node := c.eventView(ev); node != nil {
			turbo.EncodeSSE(w, node)
			rc.Flush()
		}
	}
}

func (c *Config) eventView(ev *model.Event) html.Node {
	switch ev.Type {
	case progress.EventOpen:
		return view.ProgressItemAppend(ev.Progress)
	case progress.EventUpdate:
		return view.ProgressItemUpdate(ev.Progress)
	case progress.EventClose:
		return view.ProgressItemRemove(ev.Progress)
	case model.EventSeriesSetTitle:
		return view.SeriesSetTitle(ev.ID, ev.Text)
	case model.EventMovieEditionSetTitle:
		return view.MovieEditionSetTitle(ev.ID, ev.Text)
	}
	return nil
}
