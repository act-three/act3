package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

func AppTasks(running []*model.RunningTask, queued, failed []*model.Task) (string, domi.Node) {
	return "Tasks", ScrollY(Class("v-system"))(
		html.Div()(domi.Text("Scheduled Tasks")),

		html.Div(Class("v-system-field"))(
			html.Div()(domi.Text("Running")),
			expr.IfElse(len(running) > 0,
				func() domi.Node {
					return TableRoot(Class("v-system-input-wide"))(
						TableHeader()(
							TableRow()(
								TableHead()(domi.Text("Task")),
								TableHead()(domi.Text("ID")),
								TableHead()(domi.Text("Args")),
								TableHead()(),
							),
						),
						TableBody()(
							rangeNodes(running, func(t *model.RunningTask) domi.Node {
								return html.TR()(
									TableCell()(domi.Text(t.Type())),
									TableCell()(domi.Text(t.ID())),
									TableCell()(domi.Text(t.Args())),
									TableCell()(
										Button(onClick(&msg.TaskKill{ID: t.ID()}), Destructive)(
											domi.Text("Kill"),
										),
									),
								)
							}),
						),
					)
				},
				func() domi.Node {
					return Text("No running tasks")
				},
			),
		),

		html.Div(Class("v-system-field"))(
			html.Div()(domi.Text("Failed")),
			expr.IfElse(len(failed) > 0,
				func() domi.Node {
					return TableRoot(Class("v-system-input-wide"))(
						TableHeader()(
							TableRow()(
								TableHead()(domi.Text("Task")),
								TableHead()(domi.Text("ID")),
								TableHead()(domi.Text("Args")),
								TableHead()(domi.Text("Failures")),
								TableHead()(),
							),
						),
						TableBody()(
							rangeNodes(failed, func(t *model.Task) domi.Node {
								return taskRow(t, "Retry")
							}),
						),
					)
				},
				func() domi.Node {
					return Text("No failed tasks")
				},
			),
		),

		html.Div(Class("v-system-field"))(
			html.Div()(domi.Text("Queued")),
			TableRoot(Class("v-system-input-wide"))(
				TableHeader()(
					TableRow()(
						TableHead()(domi.Text("Task")),
						TableHead()(domi.Text("ID")),
						TableHead()(domi.Text("Args")),
						TableHead()(domi.Text("Failures")),
						TableHead()(domi.Text("Next Run")),
					),
				),
				TableBody()(
					rangeNodes(queued, func(t *model.Task) domi.Node {
						return taskRow(t, "Run Now")
					}),
				),
			),
		),
	)
}

func taskRow(t *model.Task, runLabel string) domi.Node {
	s := t.FailureDesc()
	return Group(
		html.TR()(
			TableCell()(domi.Text(t.Type())),
			TableCell()(
				domi.Text(t.ID()),
				Button(onClick(&msg.TaskDelete{ID: t.ID()}), ButtonGhost)(
					domi.Text("Delete"),
				),
			),
			TableCell()(domi.Text(t.Args())),
			TableCell()(domi.Textf("%d", t.Failures())),
			TableCell()(
				iff(!t.Failed(), func() domi.Node {
					return domi.Textf("%v", t.NextRun())
				}),

				Button(onClick(&msg.TaskRun{ID: t.ID()}), ButtonGhost)(
					domi.Text(runLabel),
				),
			),
		),
		iff(s != "", func() domi.Node {
			return html.TR()(
				TableCell(
					attr.Colspan("5"),
				)(
					Code()(
						domi.Text(s),
					),
				),
			)
		}),
	)
}
