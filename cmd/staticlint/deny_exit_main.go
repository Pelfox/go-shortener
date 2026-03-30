package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// DenyExitInMainAnalyzer запрещает использование функции Exit пакета os в main
// функции пакета main.
//
// Использование данной функции разрешено в других файлах кода, и проверка
// выполняется лишь для main.main.
var DenyExitInMainAnalyzer = &analysis.Analyzer{
	Name: "deny_exit_main",
	Doc:  "Запрещает использовать os.Exit в main",
	Run:  denyExitInMain,
}

func denyExitInMain(pass *analysis.Pass) (any, error) {
	// проверяем лишь main пакет
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			// проверяем только функции с названием main
			if !ok || fn.Name.Name != "main" {
				continue
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel == nil {
					return true
				}

				if selector.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "direct call to os.Exit in main.main is forbidden")
				}

				return true
			})
		}
	}

	return nil, nil
}
