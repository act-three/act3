package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/table"
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
										table.Cell()(
											html.Text(t.ID()),
											html.Form(
												attr.Action("/do/delete-task/"+t.ID()),
												attr.Method("POST"),
											)(
												Button()(
													html.Text("Delete"),
												).
													With(ButtonBorderless),
											),
										),
										table.Cell()(html.Text(t.Args())),
										table.Cell()(html.Textf("%d", t.Failures())),
										table.Cell()(
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
															Button()(
																html.Text("Run Now"),
															).
																With(ButtonBorderless),
														),
													)
												},
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
	)
}
