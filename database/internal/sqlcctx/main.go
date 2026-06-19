// Command sqlcctx post-processes the sqlc-generated schema package so
// that the request Context is held on the Queries struct rather than
// threaded through every query method.
//
// sqlc offers no option to do this, so the tool runs as a go:generate
// step immediately after `sqlc generate`. It makes three changes:
//
//   - adds a ctx field to the Queries struct and threads it through the
//     New constructor and the WithTx method;
//   - drops the leading "ctx context.Context" parameter from every
//     Queries method;
//   - rewrites each q.db.*Context(ctx, …) call to use q.ctx instead.
//
// Edits are made on the AST and emitted with the go/format printer, so
// the output is always valid, gofmt'd Go and comments stay attached to
// their declarations. The transform is a no-op on declarations it does
// not recognize, so it is safe to run over every file in the package.
//
// The leftover unused context import is not pruned here; run
// goimports -w afterward (the go:generate step does).
//
// Usage: sqlcctx [dir]   (dir defaults to "schema")
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: sqlcctx [dir]")
		fmt.Fprintln(os.Stderr, "[dir] should be a sqlc output dir")
		os.Exit(1)
	}
	dir := os.Args[1]
	for _, name := range []string{"db.go", "query.sql.go"} {
		if err := rewrite(filepath.Join(dir, name)); err != nil {
			fmt.Fprintln(os.Stderr, "sqlcctx:", err)
			os.Exit(1)
		}
	}
}

// rewrite applies the context transform to one generated file and
// writes the reprinted result back.
func rewrite(path string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	transform(file)
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func transform(file *ast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			addCtxField(d)
		case *ast.FuncDecl:
			switch {
			case d.Recv == nil && d.Name.Name == "New":
				prependCtxParam(d.Type)
				setQueriesCtx(d.Body, ast.NewIdent("ctx"))
			case isQueriesMethod(d):
				dropCtxParam(d)
				setQueriesCtx(d.Body, queriesCtxSelector(receiverName(d)))
			}
		}
	}
}

// addCtxField adds a ctx field to the Queries struct, if d declares it
// and it has none yet.
func addCtxField(d *ast.GenDecl) {
	if d.Tok != token.TYPE {
		return
	}
	for _, spec := range d.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Queries" {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, f := range st.Fields.List {
			if fieldName(f) == "ctx" {
				return
			}
		}
		st.Fields.List = append(st.Fields.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent("ctx")},
			Type:  contextType(),
		})
	}
}

// prependCtxParam inserts a leading ctx context.Context parameter,
// unless one is already present.
func prependCtxParam(ft *ast.FuncType) {
	if len(ft.Params.List) > 0 && isCtxField(ft.Params.List[0]) {
		return
	}
	ft.Params.List = append([]*ast.Field{{
		Names: []*ast.Ident{ast.NewIdent("ctx")},
		Type:  contextType(),
	}}, ft.Params.List...)
}

// dropCtxParam removes a leading ctx parameter from a Queries method
// and repoints its q.db.*Context calls at q.ctx.
func dropCtxParam(fn *ast.FuncDecl) {
	params := fn.Type.Params
	if params == nil || len(params.List) == 0 || !isCtxField(params.List[0]) {
		return
	}
	recv := receiverName(fn)
	params.List = params.List[1:]
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 || !isDBContextCall(call.Fun, recv) {
			return true
		}
		if id, ok := call.Args[0].(*ast.Ident); ok && id.Name == "ctx" {
			// Reuse the argument's position so the printer keeps
			// surrounding comments correctly ordered.
			call.Args[0] = &ast.SelectorExpr{
				X:   &ast.Ident{NamePos: id.NamePos, Name: recv},
				Sel: ast.NewIdent("ctx"),
			}
		}
		return true
	})
}

// setQueriesCtx ensures every &Queries{…} literal in body sets ctx to
// val.
func setQueriesCtx(body *ast.BlockStmt, val ast.Expr) {
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || !isIdent(lit.Type, "Queries") {
			return true
		}
		for _, elt := range lit.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok && isIdent(kv.Key, "ctx") {
				return true
			}
		}
		lit.Elts = append(lit.Elts, &ast.KeyValueExpr{
			Key:   ast.NewIdent("ctx"),
			Value: val,
		})
		return true
	})
}

func contextType() ast.Expr {
	return &ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")}
}

func queriesCtxSelector(recv string) ast.Expr {
	return &ast.SelectorExpr{X: ast.NewIdent(recv), Sel: ast.NewIdent("ctx")}
}

func isQueriesMethod(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	return ok && isIdent(star.X, "Queries")
}

func receiverName(fn *ast.FuncDecl) string {
	if names := fn.Recv.List[0].Names; len(names) > 0 {
		return names[0].Name
	}
	return "q"
}

func isCtxField(f *ast.Field) bool {
	return len(f.Names) == 1 && f.Names[0].Name == "ctx" && isContextContext(f.Type)
}

func isContextContext(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, "context") && sel.Sel.Name == "Context"
}

// isDBContextCall reports whether fun is a <recv>.db.XxxContext selector.
func isDBContextCall(fun ast.Expr, recv string) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok || !strings.HasSuffix(sel.Sel.Name, "Context") {
		return false
	}
	inner, ok := sel.X.(*ast.SelectorExpr)
	return ok && inner.Sel.Name == "db" && isIdent(inner.X, recv)
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

func fieldName(f *ast.Field) string {
	if len(f.Names) > 0 {
		return f.Names[0].Name
	}
	return ""
}
