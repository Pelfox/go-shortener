package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// DenyPanicOutsideMain запрещает использование os.Exit, log.Fatal и panic вне
// функции main.main.
var DenyPanicOutsideMain = &analysis.Analyzer{
	Name: "deny_panic_outside_main",
	Doc:  "Запрещает использовать os.Exit, log.Fatal и panic вне main.main.",
	Run:  denyPanicOutsideMain,
}

func denyPanicOutsideMain(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			// вызывать панику/exit в main.main можно
			if !ok || (pass.Pkg.Name() == "main" && fn.Name.Name == "main") {
				continue
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				// проверка на вызов функции panic
				if ident, ok := call.Fun.(*ast.Ident); ok {
					if ident.Name == "panic" {
						pass.Reportf(call.Pos(), "panic() call is forbidden outside main.main")
					}
					return true
				}

				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel == nil {
					return true
				}

				// получение левой части выражения (до точки)
				ident, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}

				// выбрасываем предупреждение только на os.Exit
				if ident.Name == "os" && selector.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "call to os.Exit() outside main.main is forbidden")
				}

				// проверяем на log.Fatal
				if ident.Name == "log" && selector.Sel.Name == "Fatal" {
					pass.Reportf(call.Pos(), "call to log.Fatal() outside main.main is forbidden")
				}

				return true
			})
		}
	}

	return nil, nil
}
