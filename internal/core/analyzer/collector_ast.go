package analyzer

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"go_tool/internal/config"
)

type ASTInspector struct{}

func (c *ASTInspector) Collect(ctx context.Context, cfg config.Config) ([]Occurrence, error) {
	root := cfg.Scan.Workspace
	if root == "" {
		root = "."
	}

	files := make([]string, 0, 128)
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
		if !shouldInclude(path, cfg.Scan.Include, cfg.Scan.Exclude) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	workers := cfg.Scan.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	var (
		mu          sync.Mutex
		occurrences []Occurrence
		wg          sync.WaitGroup
		fileCh      = make(chan string)
		errCh       = make(chan error, 1)
	)

	workerFn := func() {
		defer wg.Done()
		fset := token.NewFileSet()
		for path := range fileCh {
			select {
			case <-ctx.Done():
				return
			default:
			}

			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				continue
			}

			var local []Occurrence

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
				local = append(local, Occurrence{
					File:    path,
					Line:    pos.Line,
					Column:  pos.Column,
					Snippet: lit,
					SQL:     lit,
					Kind:    OccurrenceKindSQL,
				})

				return true
			})

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
						local = append(local, Occurrence{
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
					local = append(local, Occurrence{
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

			mu.Lock()
			occurrences = append(occurrences, local...)
			mu.Unlock()
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go workerFn()
	}

	go func() {
		defer close(fileCh)
		for _, path := range files {
			select {
			case <-ctx.Done():
				return
			case fileCh <- path:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	if err := <-errCh; err != nil {
		return nil, err
	}

	return occurrences, nil
}

func shouldInclude(path string, includes, excludes []string) bool {
	if len(excludes) > 0 && matchAny(path, excludes) {
		return false
	}
	if len(includes) == 0 {
		return true
	}
	return matchAny(path, includes)
}

func matchAny(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, pattern := range patterns {
		p := filepath.ToSlash(pattern)
		if strings.Contains(p, "**") {
			trim := strings.ReplaceAll(p, "**", "")
			if trim != "" && strings.Contains(path, trim) {
				return true
			}
		}
		if ok, _ := filepath.Match(p, path); ok {
			return true
		}
	}
	return false
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
