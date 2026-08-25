package ui

import (
	"cmp"
	"fmt"
)

// A GridView arranges its subviews in a two-dimensional layout
// of rows and columns.
//
// The number of columns is determined by a [GridLayout].
// The grid places its subviews in order
// into rows of equal-width cells.
// It fills each row from leading edge to trailing edge
// before starting the next row.
type GridView interface {
	View

	// Alignment aligns each subview within its cell on both axes.
	//
	// The default alignment is Center.
	Alignment(Alignment) GridView

	// Gap sets the distance between adjacent rows
	// and between adjacent columns. The default gap is 8 px.
	Gap(px float64) GridView
}

// Grid arranges the given views in the grid described by g.
func Grid(g GridLayout, v ...View) GridView {
	return gridView{base{gridNode{g, v, defaultGap, Center}}}
}

// A GridLayout describes the columns of a [Grid].
type GridLayout interface {
	columns() string
	// fills is the layout's own fill request,
	// for a layout defined in terms of available space.
	fills() AxisSet
}

// Columns creates n columns of equal width.
func Columns(n int) GridLayout {
	if n < 1 {
		panic("ui: Columns requires at least one column")
	}
	return gridColumns(n)
}

type gridColumns int

// The explicit zero minimum keeps the tracks equal: a bare fraction
// has a content-sized minimum, so a wide subview would widen its
// own track at the others' expense. Instead it overflows its cell.
func (n gridColumns) columns() string {
	return fmt.Sprintf("repeat(%d, minmax(0, 1fr))", int(n))
}

func (gridColumns) fills() AxisSet { return 0 }

// ColumnMinWidth expands the grid
// to the available horizontal space,
// and divides that space into columns of equal width.
// It creates as many columns as possible
// while each column is at least px pixels wide.
func ColumnMinWidth(px float64) GridLayout {
	if !(px > 0) {
		panic("ui: CellMinWidth requires a positive width")
	}
	return gridCellMinWidth(px)
}

type gridCellMinWidth float64

func (px gridCellMinWidth) columns() string {
	return "repeat(auto-fill, minmax(" + cssPx(float64(px)) + ", 1fr))"
}

func (gridCellMinWidth) fills() AxisSet { return Horizontal }

type gridView struct{ base }

func (v gridView) Alignment(a Alignment) GridView {
	n := v.base[0].(gridNode)
	n.align = a
	v.base = base{n}
	return v
}

func (v gridView) Gap(px float64) GridView {
	n := v.base[0].(gridNode)
	n.gap = px
	v.base = base{n}
	return v
}

type gridNode struct {
	layout   GridLayout
	subviews []View
	gap      float64
	align    Alignment
}

func (g gridNode) render(env environment) box {
	inner := env
	inner.lc = layoutContext{}
	inner.container = containerGrid
	p := subviewsRendered(inner, g.subviews...)
	p.fills |= g.layout.fills()
	env.tag = cmp.Or(env.tag, "ui-grid")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", g.layout.columns())
	env.style.Set("gap", cssPx(g.gap))
	env.style.Set("place-items", g.align.placeItems())
	return build(env, p)
}
