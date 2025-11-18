package web

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	"ily.dev/act3/sys/fsinfo"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/app"
	"ily.dev/act3/web/input"
	"ily.dev/act3/web/table"
)

var excludeFSType = map[string]bool{
	"devtmpfs": true,
	"efivarfs": true,
	"tmpfs":    true,
}

type filesystem struct {
	Type string
	Path []string
	Size int64
	Used int64
	Free int64
}

func (w *web) systemTransmission(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		config, err := tx.Transmission(ctx)
		if err != nil {
			return nil, err
		}
		return app.Page("Transmission",
			html.Div(
				attr.Class("h-full w-full p-4"),
			)(
				html.Div()(html.Text("Transmission")),
				html.Form(
					attr.Method("post"),
					attr.Action("/do/update-transmission-settings"),
				)(
					html.Div(attr.Class("py-4"))(
						html.Div()(html.Text("RPC URL")),
						input.Text(
							attr.Name("url"),
							attr.Class("max-w-xs"),
							attr.Value(config.BaseURL),
						),
					),
					html.Div(attr.Class("py-4"))(
						html.Div()(html.Text("Download Folder")),
						input.Text(
							attr.Name("path"),
							attr.Class("max-w-xs"),
							attr.Value(config.Path),
						),
					),
					html.Div(attr.Class("py-4"))(
						input.Submit(
							attr.Value("Save"),
						),
					),
				),
			),
		), nil
	})
}

func (w *web) doUpdateTransmissionSettings(req *http.Request) (http.Handler, error) {
	return w.withTxRW(func(tx *model.TxRW) (http.Handler, error) {
		ctx := req.Context()
		err := tx.TransmissionSet(ctx, model.ConfigTransmission{
			Path:    req.FormValue("path"),
			BaseURL: req.FormValue("url"),
		})
		if err != nil {
			return nil, err
		}
		return http.RedirectHandler("/system/transmission", http.StatusSeeOther), nil
	})
}

func (w *web) systemStorage(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		storage, err := tx.StorageList(ctx)
		if err != nil {
			return nil, err
		}
		fsList, err := fsinfo.GetInfo()
		if err != nil {
			return nil, err
		}
		var fs []*filesystem
		for _, f := range fsList {
			if excludeFSType[f.Type] {
				continue
			}
			if f.Size == 0 {
				continue
			}
			path := []string{f.Path[0]}
		outer:
			for _, p := range f.Path[1:] {
				for _, s := range path {
					if filepathHasPrefix(p, s) {
						continue outer
					}
				}
				path = append(path, p)
			}
			fs = append(fs, &filesystem{
				Type: f.Type,
				Path: path,
				Size: int64(f.Size),
				Used: int64(f.Size) - int64(f.Avail),
				Free: int64(f.Avail),
			})
		}

		return app.Page("Storage",
			html.Div(attr.Class("settings/body"))(
				html.H2()(html.Text("storage locations")),
				html.Form(
					attr.Attr("x-target")("dirs"),
					attr.Method("post"),
					attr.Action("/do/add-directory"),
					attr.Style("padding: 1em; border: 1px solid #00000022"),
				)(
					html.H3()(html.Text("add directory")),
					html.Div()(
						html.Label(
							attr.Style("display: block; font-weight: bold"),
						)(
							html.Text("contents"),
						),
						html.Div()(
							html.Input(
								attr.Type("radio"),
								attr.ID("contents-movie"),
								attr.Name("contents"),
								attr.Value("movie"),
							),
							html.Label(attr.For("contents-movie"))(html.Text("movies")),
						),
						html.Div()(
							html.Input(
								attr.Type("radio"),
								attr.ID("contents-series"),
								attr.Name("contents"),
								attr.Value("series"),
							),
							html.Label(attr.For("contents-series"))(html.Text("series")),
						),
						html.Label(
							attr.For("path"),
							attr.Style("display: block; font-weight: bold"),
						)(
							html.Text("filesystem path"),
						),
						input.Text(
							attr.Class("max-w-xs"),
							attr.ID("path"),
							attr.Name("path"),
						),
						input.Submit(attr.Value("add")),
					),
				),
				html.Div(attr.ID("dirs"))(
					html.Range(storage, storageItem),
					html.If(len(storage) == 0, func() html.Node {
						return html.P()(html.I()(html.Text("no directories defined")))
					}),
				),
				html.H2()(html.Text("filesystem info")),
				html.Ul()(
					html.Range(fs, fsItem),
				),
			),
		), nil
	})
}

func (w *web) systemTasks(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tx *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		tasks, err := tx.TaskList(ctx)
		if err != nil {
			return nil, err
		}
		return app.Page("Tasks",
			ScrollArea(
				attr.Class("h-full w-full p-4"),
			)(
				html.Div()(html.Text("Scheduled Tasks")),
				html.Div()(html.Text("Tasks")),
				html.Div()(
					html.Div(
						attr.Class("py-4"),
					)(
						table.Root(attr.Class("max-w-full"))(
							table.Header()(
								table.Row()(
									table.Head()(html.Text("Task")),
									table.Head()(html.Text("ID")),
									table.Head()(html.Text("Args")),
									table.Head()(html.Text("Failures")),
									table.Head()(html.Text("Next Run")),
								),
							),
							table.Body()(
								html.Range(tasks, func(t *model.Task) html.Node {
									s := t.FailureDesc()
									return html.Group(
										html.Tr()(
											table.Cell()(html.Text(t.Type())),
											table.Cell()(html.Text(t.ID())),
											table.Cell()(html.Text(t.Args())),
											table.Cell()(html.Textf("%d", t.Failures())),
											table.Cell()(
												html.Textf("%v", t.NextRun()),
												html.Form(
													attr.Action("/do/run-task/"+t.ID()),
													attr.Method("POST"),
												)(
													Button()(
														html.Text("Run Now"),
													).
														With(ButtonBorderless),
												),
											),
										),
										expr.IfElse(s != "",
											func() html.Node {
												return html.Tr()(
													table.Cell(
														attr.Colspan("5"),
													)(
														html.Div(
															attr.Class("whitespace-pre-wrap"),
														)(
															html.Text(s),
														),
													),
												)
											},
											func() html.Node {
												return html.Group()
											},
										),
									)
								}),
							),
						),
					),
				),
			),
		), nil
	})
}

func (w *web) doRunTask(req *http.Request) (http.Handler, error) {
	ctx := req.Context()
	err := w.model.RunTaskNow(ctx, req.PathValue("id"))
	if err != nil {
		return nil, err
	}
	return http.RedirectHandler("/system/tasks", http.StatusSeeOther), nil
}

func storageItem(it *model.Storage) html.Node {
	return html.Div()(
		html.Text(it.Contents()),
		html.Text(" "),
		html.Text(it.Path()),
	)
}

func fsItem(fs *filesystem) html.Node {
	s := fmt.Sprintf("%s %d %d avail:%d %s", fs.Path, fs.Used, fs.Size, fs.Free, fs.Type)
	return html.Li()(
		html.Text(s),
	)
}

func filepathHasPrefix(p, prefix string) bool {
	prefix = path.Clean(prefix)
	for {
		p = path.Clean(p)
		if p == prefix {
			return true
		}
		if p == "/" {
			return false
		}
		p, _ = filepath.Split(p)
	}
}
