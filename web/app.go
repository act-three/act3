package web

import (
	"context"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) app(ctx context.Context, tx *model.TxR, title string, body html.Node) (html.Node, error) {
	stats, err := tx.TaskStats(ctx)
	if err != nil {
		return nil, err
	}
	return view.App(title, body, view.AppConfig{
		TaskCount:      stats.Queued + stats.Running,
		TaskCountError: stats.CountError,
	}), nil
}
