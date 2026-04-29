package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) appTasks(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	running := c.Model.RunningTasks()
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		tasks, err := tx.TaskList(ctx)
		if err != nil {
			return nil, err
		}
		var queued, failed []*model.Task
		for _, t := range tasks {
			if t.Failed() {
				failed = append(failed, t)
			} else {
				queued = append(queued, t)
			}
		}
		title, body := view.AppTasks(running, queued, failed)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) doTaskRun(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	ctx := req.Context()
	err := c.Model.RunTaskNow(ctx, req.PathValue("id"))
	if err != nil {
		return nil, err
	}
	http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
	return nil, nil
}

func (c *Config) doTaskKill(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	c.Model.KillTask(req.PathValue("id"))
	http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
	return nil, nil
}

func (c *Config) doTaskDelete(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		err := tx.TaskDelete(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, "/app/tasks", http.StatusSeeOther)
		return nil, nil
	})
}
