package view

import (
	"fmt"

	"ily.dev/act3/database"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
)

// Degraded renders the schema mismatch page shown when the
// database cannot be opened due to a digest mismatch.
func Degraded(
	sme *database.SchemaMismatchError,
	stats []database.TableStat,
	dbFileSize string,
) html.Node {
	var totalRows int64
	for _, s := range stats {
		totalRows += s.RowCount
	}
	hasData := totalRows > 0

	return base("Schema Mismatch")()(
		FlexCol(Class("v-degraded"), Gap6)(
			Text("Schema Mismatch", Size7, TextBold),
			Text("The database schema does not match "+
				"what this version of the server expects. "+
				"The database must be reinitialized before "+
				"the server can start.", Size3),
			Code(CodeSize2, CodeWrap)(
				html.Pre()(html.Text(fmt.Sprintf(
					"Version:  %s\nStored:   %s\n"+
						"Expected: %s\nDB Path:  %s",
					sme.Version,
					sme.StoredDigest,
					sme.ExpectedDigest,
					sme.DBPath,
				))),
			),
			Text("Database Contents", Size5, TextBold),
			TableRoot(TableSize2)(
				TableHeader()(
					TableRow()(
						TableHead()(html.Text("Table")),
						TableHead()(html.Text("Rows")),
					),
				),
				TableBody()(
					html.Range(stats, func(s database.TableStat) html.Node {
						return TableRow()(
							TableCell()(html.Text(s.Name)),
							TableCell()(html.Text(
								fmt.Sprintf("%d", s.RowCount),
							)),
						)
					}),
					TableRow()(
						TableCell()(Text("Total", TextBold)),
						TableCell()(Text(
							fmt.Sprintf("%d", totalRows),
							TextBold,
						)),
					),
				),
			),
			Text("Database file size: "+dbFileSize, Size2),
			degradedAction(hasData),
		),
	)
}

func degradedAction(hasData bool) html.Node {
	if hasData {
		return Group(
			Card(Destructive, Class("v-degraded-alert"))(
				CardContent()(
					Text("Reinitializing will delete all "+
						"existing data. This cannot be "+
						"undone.", Size3),
				),
			),
			html.Form(
				attr.Method("POST"),
				attr.Action("/-/do/database-reset"),
				Attr("data-turbo")("false"),
			)(
				Button(
					Destructive,
					ButtonSize3,
					Attr("type")("submit"),
					Attr("onclick")(
						"return confirm("+
							"'Delete all data and "+
							"reinitialize the database?')",
					),
				)(html.Text("Delete and Reinitialize")),
			),
		)
	}
	return html.Form(
		attr.Method("POST"),
		attr.Action("/-/do/database-reset"),
		Attr("data-turbo")("false"),
	)(
		Button(
			ButtonSolid,
			ButtonSize3,
			Attr("type")("submit"),
		)(html.Text("Reinitialize Database")),
	)
}
