package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func AppTasks(running []*model.RunningTask, queued []*model.Task) html.Node {
	return app("Tasks",
		ScrollY(attr.Class("v-system"))(
			html.Div()(html.Text("Scheduled Tasks")),

			html.Div(attr.Class("v-system-field"))(
				html.Div()(html.Text("Running")),
				expr.IfElse(len(running) > 0,
					func() html.Node {
						return TableRoot(TableGhost, attr.Class("v-system-input-wide"))(
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
											html.Form(
												attr.Action("/-/do/task-kill/"+t.ID()),
												attr.Method("POST"),
											)(
												Button(Destructive)(
													html.Text("Kill"),
												),
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

			html.Div(attr.Class("v-system-field"))(
				html.Div()(html.Text("Queued")),
				TableRoot(TableGhost, attr.Class("v-system-input-wide"))(
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
							return html.Group(
								html.Tr()(
									TableCell()(html.Text(t.Type())),
									TableCell()(
										html.Text(t.ID()),
										html.Form(
											attr.Action("/-/do/task-delete/"+t.ID()),
											attr.Method("POST"),
										)(
											Button(ButtonGhost)(
												html.Text("Delete"),
											),
										),
									),
									TableCell()(html.Text(t.Args())),
									TableCell()(html.Textf("%d", t.Failures())),
									TableCell()(
										html.Textf("%v", t.NextRun()),
										html.Form(
											attr.Action("/-/do/task-run/"+t.ID()),
											attr.Method("POST"),
										)(
											Button(ButtonGhost)(
												html.Text("Run Now"),
											),
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
		),
	)
}
