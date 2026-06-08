package view

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/database"
	. "ily.dev/act3/ui"
)

// Degraded renders the schema mismatch page shown when the
// database cannot be opened due to a digest mismatch.
func Degraded(
	sme *database.SchemaMismatchError,
	stats []database.TableStat,
	dbFileSize int64,
) domi.Node {
	var totalRows int64
	for _, s := range stats {
		totalRows += s.RowCount
	}
	hasData := totalRows > 0

	return domi.Fragment(
		FlexCol(Class("v-degraded"), Gap6)(
			Text("Schema Mismatch", Size7, TextBold),
			Text("The database schema does not match "+
				"what this version of the server expects. "+
				"The database must be reinitialized before "+
				"the server can start.", Size3),
			Code(CodeSize2, CodeWrap)(
				html.Pre()(domi.Text(fmt.Sprintf(
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
						TableHead()(domi.Text("Table")),
						TableHead()(domi.Text("Rows")),
					),
				),
				TableBody()(
					rangeNodes(stats, func(s database.TableStat) domi.Node {
						return TableRow()(
							TableCell()(domi.Text(s.Name)),
							TableCell()(domi.Text(
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
			Text("Database file size: "+fmtSize(dbFileSize), Size2),
			degradedAction(hasData),
		),
	)
}

func fmtSize(size int64) string {
	const (
		k = 1000
		M = 1000 * k
		G = 1000 * M
		T = 1000 * G
		P = 1000 * T
	)
	switch {
	case size > P*10:
		return fmt.Sprintf("%dPB", size/P)
	case size > P:
		return fmt.Sprintf("%.1fPB", float64(size)/P)
	case size > T*10:
		return fmt.Sprintf("%dTB", size/T)
	case size > T:
		return fmt.Sprintf("%.1fTB", float64(size)/T)
	case size > G*10:
		return fmt.Sprintf("%dGB", size/G)
	case size > G:
		return fmt.Sprintf("%.1fGB", float64(size)/G)
	case size > M*10:
		return fmt.Sprintf("%dMB", size/M)
	case size > M:
		return fmt.Sprintf("%.1fMB", float64(size)/M)
	case size > k*10:
		return fmt.Sprintf("%dkB", size/k)
	case size > k:
		return fmt.Sprintf("%.1fkB", float64(size)/k)
	}
	return fmt.Sprintf("%d bytes", size)
}

func degradedAction(hasData bool) domi.Node {
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
				)(domi.Text("Delete and Reinitialize")),
			),
		)
	}
	return html.Form(
		attr.Method("POST"),
		attr.Action("/-/do/database-reset"),
	)(
		Button(
			ButtonSolid,
			ButtonSize3,
			Attr("type")("submit"),
		)(domi.Text("Reinitialize Database")),
	)
}
