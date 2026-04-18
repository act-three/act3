package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func AppTasks(running []*model.RunningTask, queued []*model.Task) (string, html.Node) {
	return "Tasks", ScrollY(Class("v-system"))(
		html.Div()(html.Text("Scheduled Tasks")),

		html.Div(Class("v-system-field"))(
			html.Div()(html.Text("Running")),
			expr.IfElse(len(running) > 0,
				func() html.Node {
					return TableRoot(Class("v-system-input-wide"))(
						TableHeader()(
							TableRow()(
								TableHead()(html.Text("Task")),
								TableHead()(html.Text("ID")),
								TableHead()(html.Text("Args")),
								TableHead()(),
							),
						),
						TableBody()(
							html.Range(running, func(t *model.RunningTask) html.Node {
								return html.Tr()(
									TableCell()(html.Text(t.Type())),
									TableCell()(html.Text(t.ID())),
									TableCell()(html.Text(t.Args())),
									TableCell()(
										ActionButton("/-/do/task-kill/"+t.ID(), nil, Destructive)(
											html.Text("Kill"),
										),
									),
								)
							}),
						),
					)
				},
				func() html.Node {
					return Text("No running tasks")
				},
			),
		),

		html.Div(Class("v-system-field"))(
			html.Div()(html.Text("Queued")),
			TableRoot(Class("v-system-input-wide"))(
				TableHeader()(
					TableRow()(
						TableHead()(html.Text("Task")),
						TableHead()(html.Text("ID")),
						TableHead()(html.Text("Args")),
						TableHead()(html.Text("Failures")),
						TableHead()(html.Text("Next Run")),
					),
				),
				TableBody()(
					html.Range(queued, func(t *model.Task) html.Node {
						s := t.FailureDesc()
						return Group(
							html.Tr()(
								TableCell()(html.Text(t.Type())),
								TableCell()(
									html.Text(t.ID()),
									ActionButton("/-/do/task-delete/"+t.ID(), nil, ButtonGhost)(
										html.Text("Delete"),
									),
								),
								TableCell()(html.Text(t.Args())),
								TableCell()(html.Textf("%d", t.Failures())),
								TableCell()(
									html.Textf("%v", t.NextRun()),
									ActionButton("/-/do/task-run/"+t.ID(), nil, ButtonGhost)(
										html.Text("Run Now"),
									),
								),
							),
							html.If(s != "", func() html.Node {
								return html.Tr()(
									TableCell(
										attr.Colspan("5"),
									)(
										Code()(
											html.Text(s),
										),
									),
								)
							}),
						)
					}),
				),
			),
		),
	)
}
