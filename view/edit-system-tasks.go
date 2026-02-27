package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func EditSystemTasks(tasks []*model.Task) html.Node {
	return app("Tasks",
		ScrollArea(
			attr.Class("h-full w-full p-4"),
		)(
			html.Div()(html.Text("Scheduled Tasks")),
			html.Div()(html.Text("Tasks")),
			html.Div()(
				html.Div(
					attr.Class("py-4"),
				)(
					TableRoot(attr.Class("max-w-full"))(
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
							html.Range(tasks, func(t *model.Task) html.Node {
								s := t.FailureDesc()
								return html.Group(
									html.Tr()(
										TableCell()(html.Text(t.Type())),
										TableCell()(
											html.Text(t.ID()),
											html.Form(
												attr.Action("/do/delete-task/"+t.ID()),
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
											expr.IfElse(t.IsRunning(),
												func() html.Node {
													return Text("Running")
												},
												func() html.Node {
													return Group(
														html.Textf("%v", t.NextRun()),
														html.Form(
															attr.Action("/do/run-task/"+t.ID()),
															attr.Method("POST"),
														)(
															Button(ButtonGhost)(
																html.Text("Run Now"),
															),
														),
													)
												},
											),
										),
									),
									expr.IfElse(s != "",
										func() html.Node {
											return html.Tr()(
												TableCell(
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
	)
}
