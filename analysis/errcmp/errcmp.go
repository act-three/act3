// Package errcmp defines an analyzer that reports comparing error
// values with == or != instead of errors.Is.
//
// Comparing errors with == only matches the exact sentinel value and
// does not see through wrapping, so it silently breaks when an
// error is wrapped with %w. errors.Is walks the wrap chain and is the
// correct comparison. The analyzer flags any == or != whose operand
// has the error interface type, except comparisons against nil, which
// are the idiomatic way to test for the presence of an error.
package errcmp

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports == and != comparisons of error values.
var Analyzer = &analysis.Analyzer{
	Name: "errcmp",
	Doc:  "reports comparing errors with == or != instead of errors.Is",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			be, ok := n.(*ast.BinaryExpr)
			if !ok || (be.Op != token.EQL && be.Op != token.NEQ) {
				return true
			}
			if !isError(pass.TypesInfo.TypeOf(be.X)) && !isError(pass.TypesInfo.TypeOf(be.Y)) {
				return true
			}
			if isNil(pass.TypesInfo, be.X) || isNil(pass.TypesInfo, be.Y) {
				return true
			}
			pass.Reportf(be.Pos(),
				"comparing errors with %s; use errors.Is instead", be.Op)
			return true
		})
	}
	return nil, nil
}

// isError reports whether t is exactly the predeclared error interface.
func isError(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() == nil && named.Obj().Name() == "error"
}

// isNil reports whether e is the predeclared nil identifier.
func isNil(info *types.Info, e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && info.Uses[id] == types.Universe.Lookup("nil")
}
