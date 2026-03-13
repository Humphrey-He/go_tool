package analyzer

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"go_tool/internal/config"
)

type ASTInspector struct{}

func (c *ASTInspector) Collect(ctx context.Context, cfg config.Config) ([]Occurrence, error) {
	_ = ctx
	root := cfg.Scan.Workspace
	if root == "" {
		root = "."
	}

	var occurrences []Occurrence
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		// SQL string collector
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			lit := extractSQLFromCall(call)
			if lit == "" {
				return true
			}

			pos := fset.Position(call.Pos())
			occurrences = append(occurrences, Occurrence{
				File:    path,
				Line:    pos.Line,
				Column:  pos.Column,
				Snippet: lit,
				SQL:     lit,
				Kind:    OccurrenceKindSQL,
			})

			return true
		})

		// GORM chains collector
		chains := ExtractGormChains(file)
		for _, chain := range chains {
			currentTable := ""
			for _, call := range chain.Calls {
				if call.Method == "Model" || call.Method == "Table" {
					if len(call.Args) > 0 {
						if table, ok := ExtractModelTable(call.Args[0]); ok {
							currentTable = table
						}
					}
				}

				if call.Method != "Where" {
					continue
				}
				if len(call.Args) == 0 {
					continue
				}

				if IsDynamicString(call.Args[0]) {
					pos := fset.Position(call.Pos)
					occurrences = append(occurrences, Occurrence{
						File:    path,
						Line:    pos.Line,
						Column:  pos.Column,
						Snippet: "dynamic where clause",
						SQL:     "",
						Kind:    OccurrenceKindGormWarning,
					})
					continue
				}

				field, op, ok := ExtractWhereFields(call.Args[0])
				if !ok {
					continue
				}

				pos := fset.Position(call.Pos)
				occurrences = append(occurrences, Occurrence{
					File:    path,
					Line:    pos.Line,
					Column:  pos.Column,
					Snippet: field,
					SQL:     field,
					Table:   currentTable,
					Op:      op,
					Kind:    OccurrenceKindGormWhere,
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return occurrences, nil
}

func extractSQLFromCall(call *ast.CallExpr) string {
	// 识别 fmt.Sprintf("select ...", ...)
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := fun.X.(*ast.Ident); ok {
			if ident.Name == "fmt" && fun.Sel.Name == "Sprintf" {
				if len(call.Args) == 0 {
					return ""
				}
				return stringLiteral(call.Args[0])
			}
		}
	}

	// 识别常见 Query/Exec/Raw 调用
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		switch fun.Sel.Name {
		case "Query", "QueryContext", "Exec", "ExecContext", "Raw":
			if len(call.Args) == 0 {
				return ""
			}
			return stringLiteral(call.Args[0])
		}
	}

	return ""
}

func stringLiteral(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, "`\"")
}
