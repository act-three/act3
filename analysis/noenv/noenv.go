// Package noenv defines an analyzer that reports direct environment
// variable reads in ily.dev/act3 packages other than main.
//
// The policy is that only package main may read environment variables.
// Other packages should expose APIs to control their behavior,
// keeping unit tests deterministic and free of env var dependencies.
package noenv

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const modulePath = "ily.dev/act3"

// Analyzer reports calls to os.Getenv and related functions
// in module packages other than main.
var Analyzer = &analysis.Analyzer{
	Name:     "noenv",
	Doc:      "reports direct environment variable reads outside of package main",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// banned lists the functions that read environment variables,
// keyed by import path.
var banned = map[string][]string{
	"os":      {"Getenv", "LookupEnv", "Environ", "ExpandEnv"},
	"syscall": {"Getenv"},
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	if !strings.HasPrefix(pkgPath, modulePath) {
		return nil, nil
	}
	if pass.Pkg.Name() == "main" {
		return nil, nil
	}

	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	ins.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		obj := pass.TypesInfo.Uses[sel.Sel]
		if obj == nil {
			return
		}
		fn, ok := obj.(*types.Func)
		if !ok {
			return
		}
		pkg := fn.Pkg()
		if pkg == nil {
			return
		}
		if slices.Contains(banned[pkg.Path()], fn.Name()) {
			pass.Reportf(call.Pos(),
				"direct env var access via %s.%s is not allowed outside of package main",
				pkg.Path(), fn.Name())
			return
		}
	})

	return nil, nil
}
